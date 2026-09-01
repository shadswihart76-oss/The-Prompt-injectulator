// Package handler provides HTTP handlers for the Prompt Injectulator API.
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/shadswihart76-oss/the-prompt-injectulator/internal/auth"
	"github.com/shadswihart76-oss/the-prompt-injectulator/internal/llm"
	"github.com/shadswihart76-oss/the-prompt-injectulator/internal/usage"
)

// maxRequestBodyBytes is the largest request body the server will read.
// Protects against large-body DoS before JSON decoding.
const maxRequestBodyBytes = 1 << 20 // 1 MiB

// Server holds shared dependencies for all handlers.
type Server struct {
	Cfg     *auth.Config
	Store   *auth.Store
	Tracker *usage.Tracker
}

// -----------------------------------------------------------------------
// Middleware helpers
// -----------------------------------------------------------------------

// setSecurityHeaders applies a conservative set of security headers suitable
// for this simple same-origin SPA.
func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

// setAuthResponseHeaders adds headers appropriate for token-bearing / auth responses.
func setAuthResponseHeaders(w http.ResponseWriter) {
	setSecurityHeaders(w)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}

// -----------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, dst any) error {
	limited := io.LimitReader(r.Body, maxRequestBodyBytes)
	d := json.NewDecoder(limited)
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		return err
	}
	// Reject trailing garbage after the JSON value.
	if _, err := d.Token(); err != io.EOF {
		return errors.New("unexpected data after JSON value")
	}
	return nil
}

// requireAdmin parses the JWT from r, re-validates the caller's admin status
// against the server-side config, and returns the claims on success.
// Returns (nil, false) and writes the appropriate HTTP error on failure.
func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) (*auth.Claims, bool) {
	claims, err := s.Cfg.Tokens.FromRequest(r)
	if err != nil {
		// Return a generic message; do not expose raw JWT parsing errors.
		setSecurityHeaders(w)
		writeError(w, http.StatusUnauthorized, "authentication required")
		return nil, false
	}
	// Authorization re-derived from server-side config on every request.
	if !s.Cfg.IsAdmin(claims.Email) {
		setSecurityHeaders(w)
		writeError(w, http.StatusForbidden, "admin access required")
		return nil, false
	}
	return claims, true
}

// requireAuth parses the JWT from r and returns the claims on success.
// Returns (nil, false) and writes the appropriate HTTP error on failure.
func (s *Server) requireAuth(w http.ResponseWriter, r *http.Request) (*auth.Claims, bool) {
	claims, err := s.Cfg.Tokens.FromRequest(r)
	if err != nil {
		setSecurityHeaders(w)
		writeError(w, http.StatusUnauthorized, "authentication required")
		return nil, false
	}
	return claims, true
}

// -----------------------------------------------------------------------
// POST /api/auth/register
// -----------------------------------------------------------------------

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register creates a new user account.
func (s *Server) Register(w http.ResponseWriter, r *http.Request) {
	setAuthResponseHeaders(w)
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not hash password")
		return
	}

	// Role is determined server-side from the admin allowlist; never from client input.
	role := auth.RoleUser
	if s.Cfg.IsAdmin(req.Email) {
		role = auth.RoleAdmin
	}

	if err := s.Store.Register(req.Email, hash, role); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	tok, err := s.Cfg.Tokens.IssueToken(req.Email, role, 24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not issue token")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"token": tok,
		"role":  role,
		"email": req.Email,
	})
}

// -----------------------------------------------------------------------
// POST /api/auth/login
// -----------------------------------------------------------------------

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login authenticates an existing user.
func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	setAuthResponseHeaders(w)
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Dev-mode shortcut: accept the bootstrapped test account even if it hasn't
	// been explicitly registered via the /register endpoint.
	if s.Cfg.DevMode && req.Email == s.Cfg.DevTestEmail {
		if err := auth.CheckPassword(s.Cfg.DevTestPasswordHash, req.Password); err != nil {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		// Ensure the dev account is in the store so other lookups work.
		_ = s.Store.Register(s.Cfg.DevTestEmail, s.Cfg.DevTestPasswordHash, auth.RoleAdmin)
		tok, err := s.Cfg.Tokens.IssueToken(s.Cfg.DevTestEmail, auth.RoleAdmin, 24*time.Hour)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not issue token")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"token": tok,
			"role":  auth.RoleAdmin,
			"email": s.Cfg.DevTestEmail,
		})
		return
	}

	u, ok := s.Store.FindByEmail(req.Email)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err := auth.CheckPassword(u.PasswordHash, req.Password); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Re-evaluate role from the server-side allowlist (covers promotions/demotions
	// made after the account was created without a restart).
	role := auth.RoleUser
	if s.Cfg.IsAdmin(u.Email) {
		role = auth.RoleAdmin
	}

	tok, err := s.Cfg.Tokens.IssueToken(u.Email, role, 24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not issue token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token": tok,
		"role":  role,
		"email": u.Email,
	})
}

// -----------------------------------------------------------------------
// POST /api/llm/complete
// -----------------------------------------------------------------------

type completeRequest struct {
	Provider  string `json:"provider"`
	Prompt    string `json:"prompt"`
	MaxTokens int    `json:"max_tokens,omitempty"`
}

// Complete sends a prompt to the requested LLM provider and returns the response.
// Authorization is enforced on the server side:
//   - Only server-authorized admin users (from config, not JWT claim) may select the mock provider.
//   - Cloud providers are only available when CLOUD_API_ENABLED=true.
//   - All users are subject to per-user token budget limits.
func (s *Server) Complete(w http.ResponseWriter, r *http.Request) {
	setSecurityHeaders(w)
	claims, ok := s.requireAuth(w, r)
	if !ok {
		return
	}

	var req completeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	// Validate max_tokens before any usage estimation.
	if err := usage.ValidateMaxTokens(req.MaxTokens); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// ------------------------------------------------------------------
	// Authorization: server-side policy, re-derived from config each request.
	// A valid JWT with role=admin for an email absent from ADMIN_EMAILS is denied.
	// ------------------------------------------------------------------
	isAdmin := s.Cfg.IsAdmin(claims.Email)

	if req.Provider == llm.ProviderMock && !isAdmin {
		writeError(w, http.StatusForbidden, "mock provider is available to admin users only")
		return
	}

	// Default provider selection.
	provider := req.Provider
	if provider == "" {
		if isAdmin {
			provider = llm.ProviderMock
		} else {
			provider = llm.ProviderOpenAI
		}
	}

	// Cloud provider gate: must be explicitly enabled server-side.
	if provider == llm.ProviderOpenAI && !s.Cfg.CloudAPIEnabled {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error": "cloud provider is disabled; set CLOUD_API_ENABLED=true to enable paid API access",
			"code":  "cloud_disabled",
		})
		return
	}

	var p llm.Provider
	switch provider {
	case llm.ProviderMock:
		p = &llm.MockProvider{}
	case llm.ProviderOpenAI:
		p = &llm.OpenAIProvider{APIKey: os.Getenv("OPENAI_API_KEY")}
	default:
		writeError(w, http.StatusBadRequest, "unknown provider: "+provider)
		return
	}

	// Estimate cost and reserve atomically before making the call.
	estimatedTokens := usage.EstimateTokens(len(req.Prompt), req.MaxTokens)

	// Skip budget checks for mock provider (zero cost).
	if provider != llm.ProviderMock {
		if err := s.Tracker.Reserve(claims.Email, estimatedTokens); err != nil {
			writeJSON(w, http.StatusPaymentRequired, map[string]any{
				"error":      err.Error(),
				"limit_info": usageInfo(s.Tracker, claims.Email),
			})
			return
		}
	}

	resp, err := p.Complete(r.Context(), llm.Request{
		Prompt:    req.Prompt,
		MaxTokens: req.MaxTokens,
	})
	if err != nil {
		// Release the reservation; provider call failed, no tokens consumed.
		if provider != llm.ProviderMock {
			s.Tracker.Release(claims.Email, estimatedTokens)
		}
		writeError(w, http.StatusBadGateway, "provider error: "+err.Error())
		return
	}

	// Commit actual token usage (real providers only).
	if provider != llm.ProviderMock {
		s.Tracker.Commit(claims.Email, estimatedTokens, resp.TokensUsed)
	}

	used, limit := s.Tracker.Status(claims.Email)
	writeJSON(w, http.StatusOK, map[string]any{
		"provider":    resp.Provider,
		"text":        resp.Text,
		"tokens_used": resp.TokensUsed,
		"usage": map[string]any{
			"used":  used,
			"limit": limit,
			"mode":  modeLabel(provider),
		},
	})
}

// -----------------------------------------------------------------------
// GET /api/admin/status  (admin only)
// -----------------------------------------------------------------------

// AdminStatus returns configuration and usage info for the admin user.
// Role is derived from server-side config, not the JWT claim.
func (s *Server) AdminStatus(w http.ResponseWriter, r *http.Request) {
	setSecurityHeaders(w)
	claims, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	used, limit := s.Tracker.Status(claims.Email)
	writeJSON(w, http.StatusOK, map[string]any{
		"dev_mode":     s.Cfg.DevMode,
		"email":        claims.Email,
		"role":         auth.RoleAdmin, // role is server-derived, not from JWT claim
		"mock_enabled": true,
		"cloud_enabled": s.Cfg.CloudAPIEnabled,
		"usage":        map[string]any{"used": used, "limit": limit},
	})
}

// -----------------------------------------------------------------------
// GET /api/usage  (authenticated)
// -----------------------------------------------------------------------

// UsageStatus returns the calling user's current token budget status.
func (s *Server) UsageStatus(w http.ResponseWriter, r *http.Request) {
	setSecurityHeaders(w)
	claims, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, usageInfo(s.Tracker, claims.Email))
}

// -----------------------------------------------------------------------
// POST /api/admin/reset-usage  (admin only)
// -----------------------------------------------------------------------

type resetUsageRequest struct {
	Email string `json:"email"`
}

// ResetUsage clears the token budget for the specified user (admin only).
func (s *Server) ResetUsage(w http.ResponseWriter, r *http.Request) {
	setSecurityHeaders(w)
	claims, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	_ = claims // identity verified; not needed further
	var req resetUsageRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	s.Tracker.Reset(req.Email)
	writeJSON(w, http.StatusOK, map[string]string{"status": "usage reset for " + req.Email})
}

// -----------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------

func usageInfo(t *usage.Tracker, email string) map[string]any {
	used, limit := t.Status(email)
	remaining := limit - used
	if remaining < 0 {
		remaining = 0
	}
	return map[string]any{
		"used":      used,
		"limit":     limit,
		"remaining": remaining,
	}
}

func modeLabel(provider string) string {
	if provider == llm.ProviderMock {
		return "mock (no cost)"
	}
	return "real (paid API active)"
}

// NewServer is a convenience constructor that reads USER_TOKEN_LIMIT from env.
func NewServer(cfg *auth.Config, store *auth.Store) *Server {
	limit := usage.DefaultUserLimit
	if v := os.Getenv("USER_TOKEN_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	return &Server{
		Cfg:     cfg,
		Store:   store,
		Tracker: usage.NewTracker(limit),
	}
}

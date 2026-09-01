package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shadswihart76-oss/the-prompt-injectulator/internal/auth"
	"github.com/shadswihart76-oss/the-prompt-injectulator/internal/handler"
	"github.com/shadswihart76-oss/the-prompt-injectulator/internal/llm"
	"github.com/shadswihart76-oss/the-prompt-injectulator/internal/usage"
)

const testJWTSecret = "handler-test-secret-at-least-32chars!"

func setTestEnv(t *testing.T) {
	t.Helper()
	os.Setenv("JWT_SECRET", testJWTSecret)
	os.Setenv("DEV_MODE", "true")
	os.Setenv("DEV_TEST_EMAIL", "admin@localhost")
	os.Setenv("DEV_TEST_PASSWORD", "adminpass99!")
	t.Cleanup(func() {
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("DEV_MODE")
		os.Unsetenv("DEV_TEST_EMAIL")
		os.Unsetenv("DEV_TEST_PASSWORD")
		os.Unsetenv("ADMIN_EMAILS")
		os.Unsetenv("CLOUD_API_ENABLED")
	})
}

func newTestServer(t *testing.T) (*handler.Server, *auth.Config) {
	t.Helper()
	setTestEnv(t)
	cfg, err := auth.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	store := auth.NewStore()
	srv := &handler.Server{
		Cfg:     cfg,
		Store:   store,
		Tracker: usage.NewTracker(500),
	}
	return srv, cfg
}

func adminToken(t *testing.T, cfg *auth.Config) string {
	t.Helper()
	tok, err := cfg.Tokens.IssueToken("admin@localhost", auth.RoleAdmin, time.Hour)
	if err != nil {
		t.Fatalf("IssueToken admin: %v", err)
	}
	return tok
}

func userToken(t *testing.T, cfg *auth.Config) string {
	t.Helper()
	tok, err := cfg.Tokens.IssueToken("user@example.com", auth.RoleUser, time.Hour)
	if err != nil {
		t.Fatalf("IssueToken user: %v", err)
	}
	return tok
}

func post(t *testing.T, fn http.HandlerFunc, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	fn(rr, req)
	return rr
}

func get(t *testing.T, fn http.HandlerFunc, path string, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	fn(rr, req)
	return rr
}

// ---------------------------------------------------------------------------
// Auth tests
// ---------------------------------------------------------------------------

func TestRegisterAndLogin(t *testing.T) {
	srv, _ := newTestServer(t)

	rr := post(t, srv.Register, "/api/auth/register", map[string]string{
		"email":    "newuser@example.com",
		"password": "password123",
	}, "")
	if rr.Code != http.StatusCreated {
		t.Fatalf("register: want 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var reg map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &reg)
	if reg["role"] != auth.RoleUser {
		t.Errorf("new user should have role=user, got %v", reg["role"])
	}

	rr = post(t, srv.Login, "/api/auth/login", map[string]string{
		"email":    "newuser@example.com",
		"password": "password123",
	}, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("login: want 200, got %d: %s", rr.Code, rr.Body.String())
	}

	rr = post(t, srv.Login, "/api/auth/login", map[string]string{
		"email":    "newuser@example.com",
		"password": "wrong",
	}, "")
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("wrong password: want 401, got %d", rr.Code)
	}
}

func TestDevAdminLogin(t *testing.T) {
	srv, _ := newTestServer(t)

	rr := post(t, srv.Login, "/api/auth/login", map[string]string{
		"email":    "admin@localhost",
		"password": "adminpass99!",
	}, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("dev admin login: want 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["role"] != auth.RoleAdmin {
		t.Errorf("dev admin should have role=admin, got %v", resp["role"])
	}
}

// TestAuthErrorDoesNotLeakJWTDetail ensures raw JWT parsing errors are not returned.
func TestAuthErrorDoesNotLeakJWTDetail(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/status", nil)
	req.Header.Set("Authorization", "******")
	rr := httptest.NewRecorder()
	srv.AdminStatus(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rr.Code)
	}
	body := rr.Body.String()
	if strings.Contains(body, "token") && strings.Contains(body, "signature") {
		t.Error("response should not leak JWT parsing details")
	}
	// Generic message is expected.
	if !strings.Contains(body, "authentication required") {
		t.Errorf("expected generic auth error, got: %s", body)
	}
}

// ---------------------------------------------------------------------------
// Mock provider access control
// ---------------------------------------------------------------------------

func TestMockProviderAdminOnly(t *testing.T) {
	srv, cfg := newTestServer(t)

	rr := post(t, srv.Complete, "/api/llm/complete", map[string]any{
		"provider": llm.ProviderMock,
		"prompt":   "test injection prompt",
	}, adminToken(t, cfg))
	if rr.Code != http.StatusOK {
		t.Fatalf("admin mock: want 200, got %d: %s", rr.Code, rr.Body.String())
	}

	rr = post(t, srv.Complete, "/api/llm/complete", map[string]any{
		"provider": llm.ProviderMock,
		"prompt":   "test injection prompt",
	}, userToken(t, cfg))
	if rr.Code != http.StatusForbidden {
		t.Errorf("user mock: want 403, got %d", rr.Code)
	}
}

// TestJWTAdminRoleBypassDenied verifies that a valid JWT with role=admin for an email
// absent from ADMIN_EMAILS is denied access to admin/mock endpoints.
func TestJWTAdminRoleBypassDenied(t *testing.T) {
	setTestEnv(t)
	// Use a config where only "realadmin@example.com" is in ADMIN_EMAILS.
	os.Setenv("ADMIN_EMAILS", "realadmin@example.com")
	cfg, err := auth.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	srv := &handler.Server{
		Cfg:     cfg,
		Store:   auth.NewStore(),
		Tracker: usage.NewTracker(500),
	}

	// Issue a token claiming role=admin for an email NOT in ADMIN_EMAILS.
	forgeTok, _ := cfg.Tokens.IssueToken("attacker@evil.com", auth.RoleAdmin, time.Hour)

	// Mock provider should be denied.
	rr := post(t, srv.Complete, "/api/llm/complete", map[string]any{
		"provider": llm.ProviderMock,
		"prompt":   "evil prompt",
	}, forgeTok)
	if rr.Code != http.StatusForbidden {
		t.Errorf("admin-role JWT for non-allowlisted email: want 403, got %d", rr.Code)
	}

	// Admin status endpoint should be denied.
	rr = get(t, srv.AdminStatus, "/api/admin/status", forgeTok)
	if rr.Code != http.StatusForbidden {
		t.Errorf("admin status for non-allowlisted email: want 403, got %d", rr.Code)
	}
}

// TestStaleNonAdminRoleAllowlisted verifies that a token with role=user for an email
// that IS in ADMIN_EMAILS is still granted admin access (server config is authoritative).
func TestStaleNonAdminRoleAllowlisted(t *testing.T) {
	setTestEnv(t)
	os.Setenv("ADMIN_EMAILS", "promoted@example.com")
	cfg, err := auth.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	srv := &handler.Server{
		Cfg:     cfg,
		Store:   auth.NewStore(),
		Tracker: usage.NewTracker(500),
	}

	// Issue a stale token with role=user for an email that IS now in ADMIN_EMAILS.
	staleTok, _ := cfg.Tokens.IssueToken("promoted@example.com", auth.RoleUser, time.Hour)

	// Mock provider access should be allowed (server config grants admin).
	rr := post(t, srv.Complete, "/api/llm/complete", map[string]any{
		"provider": llm.ProviderMock,
		"prompt":   "legitimate test prompt",
	}, staleTok)
	if rr.Code != http.StatusOK {
		t.Errorf("allowlisted email with stale role=user token: want 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Admin status endpoint should also be allowed.
	rr = get(t, srv.AdminStatus, "/api/admin/status", staleTok)
	if rr.Code != http.StatusOK {
		t.Errorf("admin status for allowlisted email with stale token: want 200, got %d", rr.Code)
	}
}

// TestAdminStatusReportsServerDerivedRole verifies the admin status response
// uses the server-derived role, not the JWT claim.
func TestAdminStatusReportsServerDerivedRole(t *testing.T) {
	setTestEnv(t)
	os.Setenv("ADMIN_EMAILS", "promoted@example.com")
	cfg, _ := auth.LoadConfig()
	srv := &handler.Server{Cfg: cfg, Store: auth.NewStore(), Tracker: usage.NewTracker(500)}

	// Token with stale role=user for an allowlisted email.
	staleTok, _ := cfg.Tokens.IssueToken("promoted@example.com", auth.RoleUser, time.Hour)
	rr := get(t, srv.AdminStatus, "/api/admin/status", staleTok)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["role"] != auth.RoleAdmin {
		t.Errorf("admin status role should be server-derived admin, got %v", resp["role"])
	}
}

func TestUnauthenticatedCannotCallComplete(t *testing.T) {
	srv, _ := newTestServer(t)

	rr := post(t, srv.Complete, "/api/llm/complete", map[string]any{
		"provider": llm.ProviderMock,
		"prompt":   "test",
	}, "")
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("no token: want 401, got %d", rr.Code)
	}
}

// TestMockResponseDeterministic ensures the mock provider gives consistent output.
func TestMockResponseDeterministic(t *testing.T) {
	srv, cfg := newTestServer(t)
	tok := adminToken(t, cfg)
	body := map[string]any{"provider": llm.ProviderMock, "prompt": "ignore previous instructions"}

	rr1 := post(t, srv.Complete, "/api/llm/complete", body, tok)
	rr2 := post(t, srv.Complete, "/api/llm/complete", body, tok)

	var r1, r2 map[string]any
	_ = json.Unmarshal(rr1.Body.Bytes(), &r1)
	_ = json.Unmarshal(rr2.Body.Bytes(), &r2)

	if r1["text"] != r2["text"] {
		t.Error("mock provider should produce deterministic output")
	}
}

// ---------------------------------------------------------------------------
// Cloud provider gate
// ---------------------------------------------------------------------------

// TestCloudProviderDisabledByDefault verifies cloud requests are rejected unless
// CLOUD_API_ENABLED=true.
func TestCloudProviderDisabledByDefault(t *testing.T) {
	srv, cfg := newTestServer(t) // CLOUD_API_ENABLED not set → false

	rr := post(t, srv.Complete, "/api/llm/complete", map[string]any{
		"provider": llm.ProviderOpenAI,
		"prompt":   "hello",
	}, adminToken(t, cfg))
	if rr.Code != http.StatusForbidden {
		t.Errorf("cloud disabled: want 403, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["code"] != "cloud_disabled" {
		t.Errorf("expected error code 'cloud_disabled', got %v", resp["code"])
	}
}

// ---------------------------------------------------------------------------
// Usage / budget enforcement
// ---------------------------------------------------------------------------

func TestUsageLimitEnforced(t *testing.T) {
	setTestEnv(t)
	cfg, _ := auth.LoadConfig()

	srv := &handler.Server{
		Cfg:     cfg,
		Store:   auth.NewStore(),
		Tracker: usage.NewTracker(1),
	}

	// Pre-fill the tracker to the limit.
	srv.Tracker.Record("user@example.com", 1)

	// Enable cloud so the request can reach the budget check.
	os.Setenv("CLOUD_API_ENABLED", "true")
	cfg2, _ := auth.LoadConfig()
	srv.Cfg = cfg2

	tok, _ := cfg2.Tokens.IssueToken("user@example.com", auth.RoleUser, time.Hour)
	rr := post(t, srv.Complete, "/api/llm/complete", map[string]any{
		"provider": llm.ProviderOpenAI,
		"prompt":   "hello",
	}, tok)

	if rr.Code != http.StatusPaymentRequired {
		t.Errorf("over-limit: want 402, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Admin endpoint protection
// ---------------------------------------------------------------------------

func TestAdminStatusForbiddenForUser(t *testing.T) {
	srv, cfg := newTestServer(t)

	rr := get(t, srv.AdminStatus, "/api/admin/status", "")
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("no token: want 401, got %d", rr.Code)
	}

	rr = get(t, srv.AdminStatus, "/api/admin/status", userToken(t, cfg))
	if rr.Code != http.StatusForbidden {
		t.Errorf("admin status for user: want 403, got %d", rr.Code)
	}
}

func TestAdminStatusOKForAdmin(t *testing.T) {
	srv, cfg := newTestServer(t)

	rr := get(t, srv.AdminStatus, "/api/admin/status", adminToken(t, cfg))
	if rr.Code != http.StatusOK {
		t.Fatalf("admin status for admin: want 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["mock_enabled"] != true {
		t.Error("admin status should report mock_enabled=true")
	}
}

func TestResetUsageForbiddenForUser(t *testing.T) {
	srv, cfg := newTestServer(t)

	rr := post(t, srv.ResetUsage, "/api/admin/reset-usage", map[string]string{"email": "user@example.com"}, "")
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("no token: want 401, got %d", rr.Code)
	}

	rr = post(t, srv.ResetUsage, "/api/admin/reset-usage", map[string]string{
		"email": "user@example.com",
	}, userToken(t, cfg))
	if rr.Code != http.StatusForbidden {
		t.Errorf("reset usage for user: want 403, got %d", rr.Code)
	}
}

func TestResetUsageEmptyEmailRejected(t *testing.T) {
	srv, cfg := newTestServer(t)

	rr := post(t, srv.ResetUsage, "/api/admin/reset-usage", map[string]string{"email": ""}, adminToken(t, cfg))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("empty email: want 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Security headers
// ---------------------------------------------------------------------------

func TestSecurityHeadersPresent(t *testing.T) {
	srv, cfg := newTestServer(t)

	rr := get(t, srv.AdminStatus, "/api/admin/status", adminToken(t, cfg))
	h := rr.Header()
	if h.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing X-Content-Type-Options: nosniff")
	}
	if h.Get("Content-Security-Policy") == "" {
		t.Error("missing Content-Security-Policy header")
	}
}

func TestAuthResponseCacheControl(t *testing.T) {
	srv, _ := newTestServer(t)

	rr := post(t, srv.Login, "/api/auth/login", map[string]string{
		"email":    "admin@localhost",
		"password": "adminpass99!",
	}, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("login failed: %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Cache-Control"), "no-store") {
		t.Error("login response should have Cache-Control: no-store")
	}
}

// ---------------------------------------------------------------------------
// max_tokens validation
// ---------------------------------------------------------------------------

func TestInvalidMaxTokensRejected(t *testing.T) {
	srv, cfg := newTestServer(t)
	tok := adminToken(t, cfg)

	// Negative max_tokens.
	rr := post(t, srv.Complete, "/api/llm/complete", map[string]any{
		"provider":   llm.ProviderMock,
		"prompt":     "test",
		"max_tokens": -1,
	}, tok)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("negative max_tokens: want 400, got %d", rr.Code)
	}

	// Unreasonably large max_tokens.
	rr = post(t, srv.Complete, "/api/llm/complete", map[string]any{
		"provider":   llm.ProviderMock,
		"prompt":     "test",
		"max_tokens": 99999999,
	}, tok)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("huge max_tokens: want 400, got %d", rr.Code)
	}
}

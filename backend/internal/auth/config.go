// Package auth – server-side configuration for admin/privileged users.
package auth

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// minJWTSecretLen is the minimum acceptable length for JWT_SECRET.
const minJWTSecretLen = 32

// minDevPasswordLen is the minimum acceptable length for DEV_TEST_PASSWORD.
const minDevPasswordLen = 8

// rejectedPasswords is a list of well-known insecure placeholder passwords
// that must never be accepted as a development credential.
var rejectedPasswords = []string{
	"changeme", "password", "password123", "admin", "secret",
	"letmein", "qwerty", "123456", "default",
}

// Config holds runtime security configuration derived from environment variables.
type Config struct {
	// DevMode enables the development safety-net (test account, relaxed secrets).
	// Must NOT be true in production.
	DevMode bool

	// CloudAPIEnabled controls whether paid cloud providers (e.g. OpenAI) may be used.
	// Defaults to false; must be explicitly set to true in the environment.
	CloudAPIEnabled bool

	// AdminEmails is the server-side allowlist of privileged email addresses.
	AdminEmails map[string]struct{}

	// DevTestEmail is only populated when DevMode is true.
	DevTestEmail string

	// DevTestPasswordHash is only populated when DevMode is true.
	DevTestPasswordHash string

	// Tokens provides JWT signing/parsing backed by the validated secret.
	Tokens *TokenService
}

// LoadConfig reads environment variables and returns a validated Config.
//
// Required env vars in ALL modes:
//   - JWT_SECRET – HS256 signing key (min 32 chars)
//
// Required env vars in production (DEV_MODE unset or "false"):
//   - ADMIN_EMAILS – comma-separated list of privileged email addresses
//
// Required env vars when DEV_MODE=true:
//   - DEV_TEST_EMAIL    – bootstrap admin email
//   - DEV_TEST_PASSWORD – bootstrap admin password (min 8 chars, no common placeholders)
func LoadConfig() (*Config, error) {
	devMode := strings.ToLower(os.Getenv("DEV_MODE")) == "true"

	// JWT_SECRET is required in all modes and must meet the minimum length.
	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < minJWTSecretLen {
		return nil, fmt.Errorf(
			"JWT_SECRET must be set and at least %d characters (got %d); generate one with: openssl rand -base64 48",
			minJWTSecretLen, len(jwtSecret),
		)
	}

	// Production hard-fail: ADMIN_EMAILS must be present.
	if !devMode && os.Getenv("ADMIN_EMAILS") == "" {
		return nil, errors.New("ADMIN_EMAILS must be set in production (DEV_MODE is not true)")
	}

	// DEV_MODE settings must not bleed into production.
	if !devMode && (os.Getenv("DEV_TEST_EMAIL") != "" || os.Getenv("DEV_TEST_PASSWORD") != "") {
		return nil, errors.New("DEV_TEST_EMAIL and DEV_TEST_PASSWORD must not be set when DEV_MODE is false")
	}

	cfg := &Config{
		DevMode:         devMode,
		CloudAPIEnabled: strings.ToLower(os.Getenv("CLOUD_API_ENABLED")) == "true",
		AdminEmails:     make(map[string]struct{}),
		Tokens:          &TokenService{secret: []byte(jwtSecret)},
	}

	for _, e := range strings.Split(os.Getenv("ADMIN_EMAILS"), ",") {
		e = strings.TrimSpace(strings.ToLower(e))
		if e != "" {
			cfg.AdminEmails[e] = struct{}{}
		}
	}

	if devMode {
		email := os.Getenv("DEV_TEST_EMAIL")
		if email == "" {
			return nil, errors.New("DEV_TEST_EMAIL must be set when DEV_MODE=true")
		}
		pw := os.Getenv("DEV_TEST_PASSWORD")
		if pw == "" {
			return nil, errors.New("DEV_TEST_PASSWORD must be set when DEV_MODE=true")
		}
		if len(pw) < minDevPasswordLen {
			return nil, fmt.Errorf("DEV_TEST_PASSWORD must be at least %d characters", minDevPasswordLen)
		}
		if isRejectedPassword(pw) {
			return nil, errors.New("DEV_TEST_PASSWORD is a known insecure placeholder; choose a unique local password")
		}
		h, err := HashPassword(pw)
		if err != nil {
			return nil, fmt.Errorf("hashing dev password: %w", err)
		}
		cfg.DevTestEmail = strings.ToLower(email)
		cfg.DevTestPasswordHash = h
		// Dev test account is always admin.
		cfg.AdminEmails[cfg.DevTestEmail] = struct{}{}
	}

	return cfg, nil
}

// IsAdmin returns true if email is in the server-side admin allowlist.
// Comparison is case-insensitive.
func (c *Config) IsAdmin(email string) bool {
	_, ok := c.AdminEmails[strings.ToLower(email)]
	return ok
}

// isRejectedPassword returns true if the password matches a known insecure placeholder.
func isRejectedPassword(pw string) bool {
	lower := strings.ToLower(pw)
	for _, bad := range rejectedPasswords {
		if lower == bad {
			return true
		}
	}
	return false
}

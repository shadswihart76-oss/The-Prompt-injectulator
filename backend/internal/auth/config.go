// Package auth – server-side configuration for admin/privileged users.
package auth

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Config holds runtime security configuration derived from environment variables.
type Config struct {
	// DevMode enables the development safety-net (test account, relaxed secrets).
	// Must NOT be true in production.
	DevMode bool

	// AdminEmails is the server-side allowlist of privileged email addresses.
	AdminEmails map[string]struct{}

	// DevTestEmail / DevTestPasswordHash are only populated when DevMode is true.
	DevTestEmail        string
	DevTestPasswordHash string
}

// LoadConfig reads environment variables and returns a validated Config.
//
// Required env vars in production (DEV_MODE unset or "false"):
//   - JWT_SECRET – HS256 signing key (min 32 chars recommended)
//   - ADMIN_EMAILS – comma-separated list of privileged email addresses
//
// Development-only env vars (only read when DEV_MODE=true):
//   - DEV_TEST_EMAIL    (default: admin@localhost)
//   - DEV_TEST_PASSWORD (default: changeme)
//   - ADMIN_EMAILS may be omitted in dev mode; the dev test account is
//     automatically privileged.
func LoadConfig() (*Config, error) {
	devMode := strings.ToLower(os.Getenv("DEV_MODE")) == "true"

	// Production hard-fail: JWT_SECRET must be set explicitly.
	if !devMode {
		if os.Getenv("JWT_SECRET") == "" {
			return nil, errors.New("JWT_SECRET must be set in production (DEV_MODE is not true)")
		}
		if os.Getenv("ADMIN_EMAILS") == "" {
			return nil, errors.New("ADMIN_EMAILS must be set in production (DEV_MODE is not true)")
		}
	}

	cfg := &Config{
		DevMode:     devMode,
		AdminEmails: make(map[string]struct{}),
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
			email = "admin@localhost"
		}
		pw := os.Getenv("DEV_TEST_PASSWORD")
		if pw == "" {
			pw = "changeme"
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

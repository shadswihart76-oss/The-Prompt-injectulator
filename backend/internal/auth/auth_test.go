package auth_test

import (
	"os"
	"testing"

	"github.com/shadswihart76-oss/the-prompt-injectulator/internal/auth"
)

// TestLoadConfig_ProductionFailsClosed ensures production mode refuses to start
// without required environment variables.
func TestLoadConfig_ProductionFailsClosed(t *testing.T) {
	// Clear everything
	os.Unsetenv("DEV_MODE")
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("ADMIN_EMAILS")

	_, err := auth.LoadConfig()
	if err == nil {
		t.Fatal("expected error when JWT_SECRET is unset in production mode")
	}

	// Set JWT_SECRET but not ADMIN_EMAILS
	os.Setenv("JWT_SECRET", "super-secret-32-chars-minimum-12345")
	defer os.Unsetenv("JWT_SECRET")

	_, err = auth.LoadConfig()
	if err == nil {
		t.Fatal("expected error when ADMIN_EMAILS is unset in production mode")
	}
}

// TestLoadConfig_ProductionSucceeds verifies production mode works with required vars.
func TestLoadConfig_ProductionSucceeds(t *testing.T) {
	os.Setenv("JWT_SECRET", "super-secret-32-chars-minimum-12345")
	os.Setenv("ADMIN_EMAILS", "admin@example.com,ops@example.com")
	os.Unsetenv("DEV_MODE")
	defer func() {
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("ADMIN_EMAILS")
	}()

	cfg, err := auth.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.IsAdmin("admin@example.com") {
		t.Error("admin@example.com should be admin")
	}
	if !cfg.IsAdmin("OPS@EXAMPLE.COM") { // case-insensitive
		t.Error("email comparison should be case-insensitive")
	}
	if cfg.IsAdmin("attacker@evil.com") {
		t.Error("unknown email should not be admin")
	}
}

// TestLoadConfig_DevMode verifies the dev bootstrap account is created only in dev mode.
func TestLoadConfig_DevMode(t *testing.T) {
	os.Setenv("DEV_MODE", "true")
	os.Setenv("DEV_TEST_EMAIL", "testadmin@localhost")
	os.Setenv("DEV_TEST_PASSWORD", "testpassword123")
	defer func() {
		os.Unsetenv("DEV_MODE")
		os.Unsetenv("DEV_TEST_EMAIL")
		os.Unsetenv("DEV_TEST_PASSWORD")
	}()

	cfg, err := auth.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.DevMode {
		t.Error("DevMode should be true")
	}
	if cfg.DevTestEmail != "testadmin@localhost" {
		t.Errorf("unexpected dev email: %s", cfg.DevTestEmail)
	}
	if !cfg.IsAdmin("testadmin@localhost") {
		t.Error("dev test email should be admin")
	}
}

// TestLoadConfig_DevModeNoLeakInProd ensures dev credentials are not present
// when DEV_MODE is false.
func TestLoadConfig_DevModeNoLeakInProd(t *testing.T) {
	os.Setenv("JWT_SECRET", "super-secret-32-chars-minimum-12345")
	os.Setenv("ADMIN_EMAILS", "admin@example.com")
	os.Setenv("DEV_MODE", "false")
	defer func() {
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("ADMIN_EMAILS")
		os.Unsetenv("DEV_MODE")
	}()

	cfg, err := auth.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DevTestEmail != "" {
		t.Error("dev test email should be empty in production mode")
	}
	if cfg.DevTestPasswordHash != "" {
		t.Error("dev test password hash should be empty in production mode")
	}
}

// TestJWTRoundTrip ensures tokens can be issued and re-parsed.
func TestJWTRoundTrip(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-for-unit-tests-abcdefgh")
	defer os.Unsetenv("JWT_SECRET")

	// Re-init the jwtSecret by calling IssueToken (secret is loaded at package init;
	// we need to use a stable secret so tests don't depend on init ordering).
	// For simplicity, just test the round-trip with the current secret.
	tok, err := auth.IssueToken("user@test.com", auth.RoleUser, 3600*1e9)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	claims, err := auth.ParseToken(tok)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if claims.Email != "user@test.com" {
		t.Errorf("wrong email: %s", claims.Email)
	}
	if claims.Role != auth.RoleUser {
		t.Errorf("wrong role: %s", claims.Role)
	}
	if claims.IsAdmin() {
		t.Error("RoleUser should not be admin")
	}
}

// TestPasswordHash ensures bcrypt hashing and comparison work.
func TestPasswordHash(t *testing.T) {
	hash, err := auth.HashPassword("my-secure-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if err := auth.CheckPassword(hash, "my-secure-password"); err != nil {
		t.Error("correct password should pass")
	}
	if err := auth.CheckPassword(hash, "wrong-password"); err == nil {
		t.Error("wrong password should fail")
	}
}

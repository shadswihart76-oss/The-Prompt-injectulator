package auth_test

import (
	"os"
	"testing"

	"github.com/shadswihart76-oss/the-prompt-injectulator/internal/auth"
)

// testSecret is a valid 32+ char secret used throughout tests.
const testSecret = "test-secret-for-unit-tests-abcdef"

func setDevEnv() {
	os.Setenv("JWT_SECRET", testSecret)
	os.Setenv("DEV_MODE", "true")
	os.Setenv("DEV_TEST_EMAIL", "testadmin@localhost")
	os.Setenv("DEV_TEST_PASSWORD", "secureTestPw99!")
}

func clearEnv() {
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("DEV_MODE")
	os.Unsetenv("DEV_TEST_EMAIL")
	os.Unsetenv("DEV_TEST_PASSWORD")
	os.Unsetenv("ADMIN_EMAILS")
}

// TestLoadConfig_MissingJWTSecretFails ensures JWT_SECRET is always required.
func TestLoadConfig_MissingJWTSecretFails(t *testing.T) {
	clearEnv()
	os.Setenv("ADMIN_EMAILS", "admin@example.com")

	_, err := auth.LoadConfig()
	if err == nil {
		t.Fatal("expected error when JWT_SECRET is missing")
	}
}

// TestLoadConfig_ShortJWTSecretFails ensures JWT_SECRET must be at least 32 chars.
func TestLoadConfig_ShortJWTSecretFails(t *testing.T) {
	clearEnv()
	os.Setenv("JWT_SECRET", "tooshort")
	os.Setenv("ADMIN_EMAILS", "admin@example.com")

	_, err := auth.LoadConfig()
	if err == nil {
		t.Fatal("expected error when JWT_SECRET is shorter than 32 chars")
	}
}

// TestLoadConfig_ProductionFailsWithoutAdminEmails verifies production mode fails
// when ADMIN_EMAILS is unset.
func TestLoadConfig_ProductionFailsWithoutAdminEmails(t *testing.T) {
	clearEnv()
	os.Setenv("JWT_SECRET", testSecret)
	defer clearEnv()

	_, err := auth.LoadConfig()
	if err == nil {
		t.Fatal("expected error when ADMIN_EMAILS is unset in production mode")
	}
}

// TestLoadConfig_ProductionSucceeds verifies production mode with required vars.
func TestLoadConfig_ProductionSucceeds(t *testing.T) {
	clearEnv()
	os.Setenv("JWT_SECRET", testSecret)
	os.Setenv("ADMIN_EMAILS", "admin@example.com,ops@example.com")
	defer clearEnv()

	cfg, err := auth.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.IsAdmin("admin@example.com") {
		t.Error("admin@example.com should be admin")
	}
	if !cfg.IsAdmin("OPS@EXAMPLE.COM") {
		t.Error("email comparison should be case-insensitive")
	}
	if cfg.IsAdmin("attacker@evil.com") {
		t.Error("unknown email should not be admin")
	}
}

// TestLoadConfig_DevModeRequiresCredentials verifies that dev mode fails when
// DEV_TEST_EMAIL or DEV_TEST_PASSWORD are missing.
func TestLoadConfig_DevModeRequiresCredentials(t *testing.T) {
	clearEnv()
	os.Setenv("JWT_SECRET", testSecret)
	os.Setenv("DEV_MODE", "true")
	defer clearEnv()

	_, err := auth.LoadConfig()
	if err == nil {
		t.Fatal("expected error when DEV_TEST_EMAIL is missing in dev mode")
	}

	os.Setenv("DEV_TEST_EMAIL", "admin@localhost")
	_, err = auth.LoadConfig()
	if err == nil {
		t.Fatal("expected error when DEV_TEST_PASSWORD is missing in dev mode")
	}
}

// TestLoadConfig_DevModeRejectsShortPassword ensures passwords below minimum length fail.
func TestLoadConfig_DevModeRejectsShortPassword(t *testing.T) {
	clearEnv()
	os.Setenv("JWT_SECRET", testSecret)
	os.Setenv("DEV_MODE", "true")
	os.Setenv("DEV_TEST_EMAIL", "admin@localhost")
	os.Setenv("DEV_TEST_PASSWORD", "short")
	defer clearEnv()

	_, err := auth.LoadConfig()
	if err == nil {
		t.Fatal("expected error for short DEV_TEST_PASSWORD")
	}
}

// TestLoadConfig_DevModeRejectsPlaceholderPassword ensures common insecure passwords are rejected.
func TestLoadConfig_DevModeRejectsPlaceholderPassword(t *testing.T) {
	clearEnv()
	os.Setenv("JWT_SECRET", testSecret)
	os.Setenv("DEV_MODE", "true")
	os.Setenv("DEV_TEST_EMAIL", "admin@localhost")
	defer clearEnv()

	placeholders := []string{"changeme", "password", "password123", "admin", "secret"}
	for _, pw := range placeholders {
		os.Setenv("DEV_TEST_PASSWORD", pw)
		_, err := auth.LoadConfig()
		if err == nil {
			t.Errorf("expected error for placeholder password %q", pw)
		}
	}
}

// TestLoadConfig_DevMode verifies the dev bootstrap account is created correctly.
func TestLoadConfig_DevMode(t *testing.T) {
	clearEnv()
	setDevEnv()
	defer clearEnv()

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

// TestLoadConfig_DevModeNoLeakInProd ensures dev credentials are absent
// when DEV_MODE is false.
func TestLoadConfig_DevModeNoLeakInProd(t *testing.T) {
	clearEnv()
	os.Setenv("JWT_SECRET", testSecret)
	os.Setenv("ADMIN_EMAILS", "admin@example.com")
	os.Setenv("DEV_MODE", "false")
	defer clearEnv()

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

// TestLoadConfig_ProdIgnoresDevVars ensures production mode rejects when dev vars are set.
func TestLoadConfig_ProdIgnoresDevVars(t *testing.T) {
	clearEnv()
	os.Setenv("JWT_SECRET", testSecret)
	os.Setenv("ADMIN_EMAILS", "admin@example.com")
	os.Setenv("DEV_MODE", "false")
	os.Setenv("DEV_TEST_EMAIL", "admin@localhost")
	os.Setenv("DEV_TEST_PASSWORD", "somepassword")
	defer clearEnv()

	_, err := auth.LoadConfig()
	if err == nil {
		t.Fatal("expected error when DEV_TEST_* vars are set in production mode")
	}
}

// TestTokenServiceRoundTrip ensures tokens can be issued and re-parsed via TokenService.
func TestTokenServiceRoundTrip(t *testing.T) {
	clearEnv()
	os.Setenv("JWT_SECRET", testSecret)
	os.Setenv("ADMIN_EMAILS", "admin@example.com")
	defer clearEnv()

	cfg, err := auth.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	tok, err := cfg.Tokens.IssueToken("user@test.com", auth.RoleUser, 3600*1e9)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	claims, err := cfg.Tokens.ParseToken(tok)
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

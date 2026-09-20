package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any relevant env vars
	os.Unsetenv("PORT")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("CLAIM_NAME")
	os.Unsetenv("ROLE_FORMAT")
	os.Unsetenv("LOWERCASE_ROLES")
	os.Unsetenv("FILTER_PROJECT_ID")
	os.Unsetenv("ZITADEL_SIGNING_KEY")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.ClaimName != "groups" {
		t.Errorf("ClaimName = %q, want groups", cfg.ClaimName)
	}
	if cfg.RoleFormat != "bare" {
		t.Errorf("RoleFormat = %q, want bare", cfg.RoleFormat)
	}
	if !cfg.LowercaseRoles {
		t.Errorf("LowercaseRoles = false, want true")
	}
	if cfg.FilterProjectID != "" {
		t.Errorf("FilterProjectID = %q, want empty", cfg.FilterProjectID)
	}
	if cfg.ZitadelSigningKey != "" {
		t.Errorf("ZitadelSigningKey = %q, want empty", cfg.ZitadelSigningKey)
	}
}

func TestLoad_CustomEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("LOG_LEVEL", "DEBUG")
	t.Setenv("CLAIM_NAME", "roles")
	t.Setenv("ROLE_FORMAT", "PREFIXED")
	t.Setenv("LOWERCASE_ROLES", "false")
	t.Setenv("FILTER_PROJECT_ID", "proj-999")
	t.Setenv("ZITADEL_SIGNING_KEY", "secret-key-123")

	cfg := Load()

	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want 9090", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", cfg.LogLevel)
	}
	if cfg.ClaimName != "roles" {
		t.Errorf("ClaimName = %q, want roles", cfg.ClaimName)
	}
	if cfg.RoleFormat != "prefixed" {
		t.Errorf("RoleFormat = %q, want prefixed", cfg.RoleFormat)
	}
	if cfg.LowercaseRoles {
		t.Errorf("LowercaseRoles = true, want false")
	}
	if cfg.FilterProjectID != "proj-999" {
		t.Errorf("FilterProjectID = %q, want proj-999", cfg.FilterProjectID)
	}
	if cfg.ZitadelSigningKey != "secret-key-123" {
		t.Errorf("ZitadelSigningKey = %q, want secret-key-123", cfg.ZitadelSigningKey)
	}
}

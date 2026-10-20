package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	cfg := Load()

	if cfg.Server.Port != "8081" {
		t.Errorf("Expected default port 8081, got %s", cfg.Server.Port)
	}
	if cfg.Server.GinMode != "debug" {
		t.Errorf("Expected default gin mode debug, got %s", cfg.Server.GinMode)
	}
	if cfg.Database.URI != "mongodb://localhost:27017" {
		t.Errorf("Expected default MongoDB URI, got %s", cfg.Database.URI)
	}
	if cfg.Database.Name != "vaultix" {
		t.Errorf("Expected default database name vaultix, got %s", cfg.Database.Name)
	}
	if cfg.JWT.Issuer != "vaultix-api" {
		t.Errorf("Expected default issuer vaultix-api, got %s", cfg.JWT.Issuer)
	}
	if cfg.JWT.AccessTTLMinutes != 15 {
		t.Errorf("Expected default access TTL 15 minutes, got %d", cfg.JWT.AccessTTLMinutes)
	}
	if cfg.JWT.RefreshTTLHours != 720 {
		t.Errorf("Expected default refresh TTL 720 hours, got %d", cfg.JWT.RefreshTTLHours)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	os.Setenv("APP_PORT", "9090")
	os.Setenv("MONGODB_DATABASE", "testdb")
	os.Setenv("JWT_ISSUER", "test-issuer")
	defer os.Unsetenv("APP_PORT")
	defer os.Unsetenv("MONGODB_DATABASE")
	defer os.Unsetenv("JWT_ISSUER")

	cfg := Load()

	if cfg.Server.Port != "9090" {
		t.Errorf("Expected port 9090, got %s", cfg.Server.Port)
	}
	if cfg.Database.Name != "testdb" {
		t.Errorf("Expected database name testdb, got %s", cfg.Database.Name)
	}
	if cfg.JWT.Issuer != "test-issuer" {
		t.Errorf("Expected issuer test-issuer, got %s", cfg.JWT.Issuer)
	}
}

func TestLoad_CORS(t *testing.T) {
	os.Setenv("CORS_ALLOWED_ORIGINS", "http://a.com,http://b.com , http://c.com")
	defer os.Unsetenv("CORS_ALLOWED_ORIGINS")

	cfg := Load()

	if len(cfg.CORS.AllowedOrigins) != 3 {
		t.Errorf("Expected 3 allowed origins, got %d", len(cfg.CORS.AllowedOrigins))
	}
	if cfg.CORS.AllowedOrigins[0] != "http://a.com" {
		t.Errorf("Expected first origin http://a.com, got %s", cfg.CORS.AllowedOrigins[0])
	}
	if cfg.CORS.AllowedOrigins[1] != "http://b.com" {
		t.Errorf("Expected second origin http://b.com (trimmed), got %s", cfg.CORS.AllowedOrigins[1])
	}
}

func TestLoad_FieldEncryptionKey(t *testing.T) {
	// 32 bytes, base64-encoded
	os.Setenv("FIELD_ENCRYPTION_KEY", "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=")
	defer os.Unsetenv("FIELD_ENCRYPTION_KEY")

	cfg := Load()

	if len(cfg.Encryption.FieldKey) != 32 {
		t.Errorf("Expected 32-byte key, got %d bytes", len(cfg.Encryption.FieldKey))
	}
}

func TestLoad_EmptyFieldEncryptionKey(t *testing.T) {
	cfg := Load()

	if cfg.Encryption.FieldKey != nil {
		t.Errorf("Expected nil key when env is empty, got %v", cfg.Encryption.FieldKey)
	}
}

func TestLoad_InvalidIntFallsBackToDefault(t *testing.T) {
	os.Setenv("APP_PORT", "not-a-number")
	defer os.Unsetenv("APP_PORT")

	cfg := Load()

	// APP_PORT is a string, so this tests getEnvAsInt indirectly via another var
	os.Setenv("MONGODB_MAX_POOL_SIZE", "abc")
	defer os.Unsetenv("MONGODB_MAX_POOL_SIZE")

	cfg = Load()
	if cfg.Database.MaxPoolSize != 100 {
		t.Errorf("Expected fallback 100, got %d", cfg.Database.MaxPoolSize)
	}
}

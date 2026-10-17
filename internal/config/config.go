package config

import (
	"encoding/base64"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	JWT        JWTConfig
	CORS       CORSConfig
	Encryption EncryptionConfig
	App        AppConfig
}

type ServerConfig struct {
	Port    string
	GinMode string
}

type AppConfig struct {
	BaseURL string
}

type DatabaseConfig struct {
	URI            string
	Name           string
	MaxPoolSize    int
	MinPoolSize    int
	MaxIdleMinutes int
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	PoolSize int
}

type JWTConfig struct {
	AccessSecret     string
	RefreshSecret    string
	AccessTTLMinutes int
	RefreshTTLHours  int
	Issuer           string
}

type CORSConfig struct {
	AllowedOrigins []string
}

type EncryptionConfig struct {
	FieldKey []byte
}

const (
	defaultAccessSecret  = "change-me-access-secret-min-32-chars!!"
	defaultRefreshSecret = "change-me-refresh-secret-min-32-chars!!"
)

// Load reads configuration from environment variables.
func Load() *Config {
	cfg := &Config{
		Server: ServerConfig{
			Port:    getEnv("APP_PORT", "8080"),
			GinMode: getEnv("GIN_MODE", "debug"),
		},
		Database: DatabaseConfig{
			URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Name:           getEnv("MONGODB_DATABASE", "vaultix"),
			MaxPoolSize:    getEnvAsInt("MONGODB_MAX_POOL_SIZE", 100),
			MinPoolSize:    getEnvAsInt("MONGODB_MIN_POOL_SIZE", 10),
			MaxIdleMinutes: getEnvAsInt("MONGODB_CONN_MAX_IDLE_MINUTES", 30),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
			PoolSize: getEnvAsInt("REDIS_POOL_SIZE", 10),
		},
		JWT: JWTConfig{
			AccessSecret:     getEnv("JWT_ACCESS_SECRET", defaultAccessSecret),
			RefreshSecret:    getEnv("JWT_REFRESH_SECRET", defaultRefreshSecret),
			AccessTTLMinutes: getEnvAsInt("JWT_ACCESS_TTL_MINUTES", 15),
			RefreshTTLHours:  getEnvAsInt("JWT_REFRESH_TTL_HOURS", 720),
			Issuer:           getEnv("JWT_ISSUER", "vaultix-api"),
		},
		CORS: CORSConfig{
			AllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")),
		},
		Encryption: EncryptionConfig{
			FieldKey: mustDecodeBase64(getEnv("FIELD_ENCRYPTION_KEY", "")),
		},
		App: AppConfig{
			BaseURL: getEnv("APP_BASE_URL", "http://localhost:8080"),
		},
	}

	// Validate critical secrets in production
	if cfg.Server.GinMode == "release" {
		if cfg.JWT.AccessSecret == defaultAccessSecret || cfg.JWT.RefreshSecret == defaultRefreshSecret {
			slog.Error("FATAL: JWT secrets are using default values. Set strong random secrets in production.")
			os.Exit(1)
		}
	}

	slog.Info("config loaded",
		"port", cfg.Server.Port,
		"db_name", cfg.Database.Name,
		"redis_addr", cfg.Redis.Addr,
		"issuer", cfg.JWT.Issuer,
	)

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
		slog.Warn("invalid integer for env var", "key", key, "default", defaultValue)
	}
	return defaultValue
}

func splitCSV(value string) []string {
	var result []string
	if value == "" {
		return result
	}
	parts := strings.Split(value, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func mustDecodeBase64(s string) []byte {
	if s == "" {
		return nil
	}
	key, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		slog.Warn("FIELD_ENCRYPTION_KEY is not valid base64", "error", err)
		return nil
	}
	return key
}

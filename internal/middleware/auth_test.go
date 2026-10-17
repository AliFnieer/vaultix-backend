package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AliFnieer/vaultix-backend/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func setupTestConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			AccessSecret:  "test-access-secret-32-characters!!",
			RefreshSecret: "test-refresh-secret-32-characters!",
			Issuer:        "vaultix-api",
		},
	}
}

func createTestToken(t *testing.T, cfg *config.Config) string {
	t.Helper()
	claims := jwt.MapClaims{
		"user_id": "507f1f77bcf86cd799439011",
		"email":   "test@example.com",
		"role":    "user",
		"iss":     cfg.JWT.Issuer,
		"exp":     time.Now().Add(time.Hour).Unix(),
		"iat":     time.Now().Unix(),
		"nbf":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.JWT.AccessSecret))
	if err != nil {
		t.Fatalf("Failed to sign test token: %v", err)
	}
	return signed
}

func setupRouter(cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/protected", AuthMiddleware(cfg), func(c *gin.Context) {
		userID, _ := GetUserID(c)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})
	return router
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	cfg := setupTestConfig()
	router := setupRouter(cfg)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_MalformedHeader(t *testing.T) {
	cfg := setupTestConfig()
	router := setupRouter(cfg)

	tests := []string{
		"Bearer",
		"Basic abc123",
		"Bearer ",
		"invalid-format",
	}

	for _, header := range tests {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", header)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Header %q: expected status %d, got %d", header, http.StatusUnauthorized, w.Code)
		}
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	cfg := setupTestConfig()
	router := setupRouter(cfg)
	token := createTestToken(t, cfg)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAuthMiddleware_WrongSecret(t *testing.T) {
	cfg := setupTestConfig()
	wrongCfg := setupTestConfig()
	wrongCfg.JWT.AccessSecret = "wrong-secret-32-characters-long!!"
	token := createTestToken(t, wrongCfg)

	router := setupRouter(cfg)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	cfg := setupTestConfig()
	claims := jwt.MapClaims{
		"user_id": "507f1f77bcf86cd799439011",
		"iss":     cfg.JWT.Issuer,
		"exp":     time.Now().Add(-time.Hour).Unix(),
		"iat":     time.Now().Add(-2 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(cfg.JWT.AccessSecret))

	router := setupRouter(cfg)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_WrongIssuer(t *testing.T) {
	cfg := setupTestConfig()
	claims := jwt.MapClaims{
		"user_id": "507f1f77bcf86cd799439011",
		"iss":     "wrong-issuer",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(cfg.JWT.AccessSecret))

	router := setupRouter(cfg)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestGetUserID_Present(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("user_id", "abc123")

	id, ok := GetUserID(c)
	if !ok {
		t.Error("GetUserID should return ok=true when user_id is set")
	}
	if id != "abc123" {
		t.Errorf("Expected user_id %q, got %q", "abc123", id)
	}
}

func TestGetUserID_Absent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	id, ok := GetUserID(c)
	if ok {
		t.Error("GetUserID should return ok=false when user_id is not set")
	}
	if id != "" {
		t.Errorf("Expected empty user_id, got %q", id)
	}
}

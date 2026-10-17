package middleware

import (
	"net/http"
	"strings"

	"github.com/AliFnieer/vaultix-backend/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	// IssuerVaultixAPI is the expected JWT issuer for this service.
	IssuerVaultixAPI = "vaultix-api"
)

// userIDKey is the context key used to store the authenticated user's id.
const userIDKey = "user_id"

// AuthMiddleware validates the JWT token in the Authorization header.
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header is required"})
			c.Abort()
			return
		}

		// Expected format: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.JWT.AccessSecret), nil
		})
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		issuer, _ := claims["iss"].(string)
		if issuer != cfg.JWT.Issuer {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token issuer"})
			c.Abort()
			return
		}

		// Store user ID in context for downstream handlers
		c.Set(userIDKey, claims["user_id"])
		c.Next()
	}
}

// GetUserID returns the authenticated user id stored by AuthMiddleware.
func GetUserID(c *gin.Context) (string, bool) {
	value, exists := c.Get(userIDKey)
	if !exists {
		return "", false
	}
	id, ok := value.(string)
	return id, ok
}

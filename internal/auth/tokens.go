package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	errInvalidToken = errors.New("invalid token")
	errTokenExpired = errors.New("token expired")
)

// IssueAccess creates a signed access token for a user.
func IssueAccess(userID, email, role, secret, issuer string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"role":    role,
		"iss":     issuer,
		"iat":     now.Unix(),
		"exp":     now.Add(ttl).Unix(),
		"nbf":     now.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// IssueRefresh creates a signed refresh token for a user.
func IssueRefresh(userID, secret, issuer string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"user_id": userID,
		"iss":     issuer,
		"iat":     now.Unix(),
		"exp":     now.Add(ttl).Unix(),
		"nbf":     now.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateAccess parses and verifies an access token, returning its claims.
func ValidateAccess(tokenString, secret, issuer string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errTokenExpired
		}
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errInvalidToken
	}
	if iss, _ := claims["iss"].(string); iss != issuer {
		return nil, errInvalidToken
	}
	return claims, nil
}

// ValidateRefresh parses and verifies a refresh token, returning the user ID.
func ValidateRefresh(tokenString, secret, issuer string) (string, error) {
	claims, err := ValidateAccess(tokenString, secret, issuer)
	if err != nil {
		return "", err
	}
	userID, _ := claims["user_id"].(string)
	return userID, nil
}

// HashToken hashes a raw token for storage (SHA-256 hex).
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

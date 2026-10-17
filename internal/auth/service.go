package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/AliFnieer/vaultix-backend/internal/apperr"
	"github.com/AliFnieer/vaultix-backend/internal/config"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"
)

// Service handles auth business logic.
type Service struct {
	db  *mongo.Database
	cfg *config.Config
}

// RegisterRequest is the payload for user registration.
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8,max=128"`
	FirstName string `json:"first_name" binding:"required,min=1,max=100"`
	LastName  string `json:"last_name" binding:"required,min=1,max=100"`
}

// LoginRequest is the payload for user login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshRequest is the payload for token refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// AuthResponse is returned on successful registration/login.
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	User         User   `json:"user"`
}

// NewService creates a new auth service.
func NewService(db *mongo.Database, cfg *config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

// Register creates a new user account and returns token pair.
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	normalizedEmail := strings.TrimSpace(strings.ToLower(req.Email))

	// Check if email is already registered
	var existing User
	if err := s.db.Collection(User{}.TableName()).FindOne(ctx, bson.M{"email": normalizedEmail}).Decode(&existing); err == nil {
		return nil, apperr.Conflict("email already registered")
	} else if err != mongo.ErrNoDocuments {
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to check existing user")
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create the user
	now := time.Now().UTC()
	user := User{
		ID:           bson.NewObjectID(),
		Email:        normalizedEmail,
		PasswordHash: string(hashedPassword),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         "user",
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if _, err := s.db.Collection(User{}.TableName()).InsertOne(ctx, user); err != nil {
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to create user")
	}

	return s.issueTokenPair(ctx, &user)
}

// Login authenticates a user and returns token pair.
func (s *Service) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	normalizedEmail := strings.TrimSpace(strings.ToLower(req.Email))

	var user User
	if err := s.db.Collection(User{}.TableName()).FindOne(ctx, bson.M{"email": normalizedEmail}).Decode(&user); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperr.Unauthorized("invalid email or password")
		}
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to find user")
	}
	if !user.IsActive {
		return nil, apperr.Forbidden("account is deactivated")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, apperr.Unauthorized("invalid email or password")
	}

	return s.issueTokenPair(ctx, &user)
}

// Refresh rotates the refresh token and issues a new access token.
func (s *Service) Refresh(ctx context.Context, req *RefreshRequest) (*AuthResponse, error) {
	userID, err := ValidateRefresh(req.RefreshToken, s.cfg.JWT.RefreshSecret, s.cfg.JWT.Issuer)
	if err != nil {
		return nil, apperr.Unauthorized("invalid or expired refresh token")
	}

	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, apperr.Unauthorized("invalid or expired refresh token")
	}

	// Revoke the used refresh token (rotation)
	var existing RefreshToken
	if err := s.db.Collection(RefreshToken{}.TableName()).FindOne(ctx, bson.M{"user_id": oid}).Decode(&existing); err == nil {
		_ = s.db.Collection(RefreshToken{}.TableName()).FindOneAndDelete(ctx, bson.M{"_id": existing.ID})
	}

	user, err := s.GetByID(ctx, userID)
	if err != nil || !user.IsActive {
		return nil, apperr.Unauthorized("invalid or expired refresh token")
	}

	return s.issueTokenPair(ctx, user)
}

// Logout revokes all refresh tokens for a user.
func (s *Service) Logout(ctx context.Context, userID string) error {
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return apperr.BadRequest("invalid user id")
	}
	_, err = s.db.Collection(RefreshToken{}.TableName()).DeleteMany(ctx, bson.M{"user_id": oid})
	if err != nil {
		return apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to revoke tokens")
	}
	return nil
}

// GetByID retrieves a user by their ID.
func (s *Service) GetByID(ctx context.Context, userID string) (*User, error) {
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, apperr.NotFound("user not found")
	}
	var user User
	if err := s.db.Collection(User{}.TableName()).FindOne(ctx, bson.M{"_id": oid}).Decode(&user); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperr.NotFound("user not found")
		}
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to get user")
	}
	return &user, nil
}

// issueTokenPair creates a new access + refresh token pair and stores the refresh token.
func (s *Service) issueTokenPair(ctx context.Context, user *User) (*AuthResponse, error) {
	accessToken, err := IssueAccess(user.ID.Hex(), user.Email, user.Role, s.cfg.JWT.AccessSecret, s.cfg.JWT.Issuer, time.Duration(s.cfg.JWT.AccessTTLMinutes)*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := IssueRefresh(user.ID.Hex(), s.cfg.JWT.RefreshSecret, s.cfg.JWT.Issuer, time.Duration(s.cfg.JWT.RefreshTTLHours)*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	ttl := time.Duration(s.cfg.JWT.RefreshTTLHours) * time.Hour
	stored := RefreshToken{
		ID:        bson.NewObjectID(),
		UserID:    user.ID,
		TokenHash: HashToken(refreshToken),
		ExpiresAt: time.Now().UTC().Add(ttl),
		CreatedAt: time.Now().UTC(),
	}
	if _, err := s.db.Collection(RefreshToken{}.TableName()).InsertOne(ctx, stored); err != nil {
		slog.Warn("failed to store refresh token", "error", err)
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    s.cfg.JWT.AccessTTLMinutes * 60,
		User:         *user,
	}, nil
}

// CleanupExpiredRefreshTokens removes expired refresh tokens.
func (s *Service) CleanupExpiredRefreshTokens(ctx context.Context) (int64, error) {
	result, err := s.db.Collection(RefreshToken{}.TableName()).DeleteMany(ctx, bson.M{
		"expires_at": bson.M{"$lt": time.Now().UTC()},
	})
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

package auth

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// User represents a registered user in the system.
type User struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Email        string        `bson:"email" json:"email"`
	PasswordHash string        `bson:"password_hash" json:"-"`
	FirstName    string        `bson:"first_name" json:"first_name"`
	LastName     string        `bson:"last_name" json:"last_name"`
	Role         string        `bson:"role" json:"role"`
	IsActive     bool          `bson:"is_active" json:"is_active"`
	CreatedAt    time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time     `bson:"updated_at" json:"updated_at"`
}

// TableName returns the collection name for users.
func (User) TableName() string { return "users" }

// RefreshToken represents a stored refresh token for session management.
type RefreshToken struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"-"`
	UserID    bson.ObjectID `bson:"user_id" json:"-"`
	TokenHash string        `bson:"token_hash" json:"-"`
	ExpiresAt time.Time     `bson:"expires_at" json:"-"`
	CreatedAt time.Time     `bson:"created_at" json:"-"`
}

// TableName returns the collection name for refresh tokens.
func (RefreshToken) TableName() string { return "refresh_tokens" }

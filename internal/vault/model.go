package vault

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Vault represents a user's vault for organizing secrets.
type Vault struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      bson.ObjectID `bson:"user_id" json:"user_id"`
	Name        string        `bson:"name" json:"name"`
	Description string        `bson:"description,omitempty" json:"description,omitempty"`
	Color       string        `bson:"color,omitempty" json:"color,omitempty"`
	Icon        string        `bson:"icon,omitempty" json:"icon,omitempty"`
	IsDefault   bool          `bson:"is_default" json:"is_default"`
	SortOrder   int           `bson:"sort_order" json:"sort_order"`
	CreatedAt   time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time     `bson:"updated_at" json:"updated_at"`
}

// TableName returns the collection name for vaults.
func (Vault) TableName() string { return "vaults" }

// CreateVaultRequest is the payload for creating a vault.
type CreateVaultRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
	Color       string `json:"color" binding:"omitempty,max=7"`
	Icon        string `json:"icon" binding:"max=50"`
	SortOrder   int    `json:"sort_order"`
}

// UpdateVaultRequest is the payload for updating a vault.
type UpdateVaultRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
	Color       string `json:"color" binding:"omitempty,max=7"`
	Icon        string `json:"icon" binding:"max=50"`
	SortOrder   int    `json:"sort_order"`
}

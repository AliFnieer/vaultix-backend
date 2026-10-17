package secret

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Secret represents an encrypted secret entry.
type Secret struct {
	ID          bson.ObjectID     `bson:"_id,omitempty" json:"id"`
	UserID      bson.ObjectID     `bson:"user_id" json:"user_id"`
	VaultID     bson.ObjectID     `bson:"vault_id" json:"vault_id"`
	Title       string            `bson:"title" json:"title"`
	Type        string            `bson:"type" json:"type"` // login, note, card, api_key, custom
	Fields      map[string]string `bson:"fields" json:"fields"`
	Tags        []string          `bson:"tags,omitempty" json:"tags,omitempty"`
	Favorite    bool              `bson:"favorite" json:"favorite"`
	IsArchived  bool              `bson:"is_archived" json:"is_archived"`
	IsTrashed   bool              `bson:"is_trashed" json:"is_trashed"`
	PasswordAge *time.Time        `bson:"password_age,omitempty" json:"password_age,omitempty"`
	CreatedAt   time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time         `bson:"updated_at" json:"updated_at"`
}

// TableName returns the collection name for secrets.
func (Secret) TableName() string { return "secrets" }

// Secret types.
const (
	TypeLogin  = "login"
	TypeNote   = "note"
	TypeCard   = "card"
	TypeAPIKey = "api_key"
	TypeCustom = "custom"
)

// CreateSecretRequest is the payload for creating a secret.
type CreateSecretRequest struct {
	Title    string            `json:"title" binding:"required,min=1,max=200"`
	Type     string            `json:"type" binding:"required,oneof=login note card api_key custom"`
	VaultID  string            `json:"vault_id" binding:"required"`
	Fields   map[string]string `json:"fields" binding:"required"`
	Tags     []string          `json:"tags" binding:"omitempty,max=20,dive,max=50"`
	Favorite bool              `json:"favorite"`
}

// UpdateSecretRequest is the payload for updating a secret.
type UpdateSecretRequest struct {
	Title    string            `json:"title" binding:"required,min=1,max=200"`
	Type     string            `json:"type" binding:"required,oneof=login note card api_key custom"`
	VaultID  string            `json:"vault_id" binding:"required"`
	Fields   map[string]string `json:"fields" binding:"required"`
	Tags     []string          `json:"tags" binding:"omitempty,max=20,dive,max=50"`
	Favorite bool              `json:"favorite"`
}

// ListSecretsResponse is the response for listing secrets.
type ListSecretsResponse struct {
	Secrets []Secret `json:"secrets"`
	Total   int64    `json:"total"`
	Page    int      `json:"page"`
	Limit   int      `json:"limit"`
}

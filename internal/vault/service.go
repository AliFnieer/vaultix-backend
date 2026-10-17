package vault

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/AliFnieer/vaultix-backend/internal/apperr"
	"github.com/AliFnieer/vaultix-backend/internal/config"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Service handles vault business logic.
type Service struct {
	db  *mongo.Database
	cfg *config.Config
}

// NewService creates a new vault service.
func NewService(db *mongo.Database, cfg *config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

// Create creates a new vault for a user.
func (s *Service) Create(ctx context.Context, userID string, req *CreateVaultRequest) (*Vault, error) {
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, apperr.BadRequest("invalid user id")
	}

	name := strings.TrimSpace(req.Name)

	// Check for duplicate vault name per user
	var existing Vault
	if err := s.db.Collection(Vault{}.TableName()).FindOne(ctx, bson.M{
		"user_id": oid,
		"name":    name,
	}).Decode(&existing); err == nil {
		return nil, apperr.Conflict("vault with this name already exists")
	} else if err != mongo.ErrNoDocuments {
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to check existing vault")
	}

	// Count user's vaults for sort order
	count, _ := s.db.Collection(Vault{}.TableName()).CountDocuments(ctx, bson.M{"user_id": oid})

	now := time.Now().UTC()
	vault := Vault{
		ID:          bson.NewObjectID(),
		UserID:      oid,
		Name:        name,
		Description: req.Description,
		Color:       req.Color,
		Icon:        req.Icon,
		IsDefault:   count == 0,
		SortOrder:   req.SortOrder,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if _, err := s.db.Collection(Vault{}.TableName()).InsertOne(ctx, vault); err != nil {
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to create vault")
	}

	return &vault, nil
}

// ListByUser returns all vaults for a user.
func (s *Service) ListByUser(ctx context.Context, userID string) ([]Vault, error) {
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, apperr.BadRequest("invalid user id")
	}

	cursor, err := s.db.Collection(Vault{}.TableName()).Find(ctx, bson.M{"user_id": oid})
	if err != nil {
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to fetch vaults")
	}
	defer cursor.Close(ctx)

	var vaults []Vault
	if err := cursor.All(ctx, &vaults); err != nil {
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to decode vaults")
	}

	return vaults, nil
}

// GetByID retrieves a vault by ID, verifying ownership.
func (s *Service) GetByID(ctx context.Context, userID, vaultID string) (*Vault, error) {
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, apperr.BadRequest("invalid user id")
	}

	vaultOID, err := bson.ObjectIDFromHex(vaultID)
	if err != nil {
		return nil, apperr.BadRequest("invalid vault id")
	}

	var vault Vault
	if err := s.db.Collection(Vault{}.TableName()).FindOne(ctx, bson.M{
		"_id":     vaultOID,
		"user_id": oid,
	}).Decode(&vault); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperr.NotFound("vault not found")
		}
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to fetch vault")
	}

	return &vault, nil
}

// Update updates a vault's details.
func (s *Service) Update(ctx context.Context, userID, vaultID string, req *UpdateVaultRequest) (*Vault, error) {
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, apperr.BadRequest("invalid user id")
	}

	vaultOID, err := bson.ObjectIDFromHex(vaultID)
	if err != nil {
		return nil, apperr.BadRequest("invalid vault id")
	}

	// Verify ownership
	var vault Vault
	if err := s.db.Collection(Vault{}.TableName()).FindOne(ctx, bson.M{
		"_id":     vaultOID,
		"user_id": oid,
	}).Decode(&vault); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperr.NotFound("vault not found")
		}
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to fetch vault")
	}

	// Check for name conflicts (if name changed)
	if req.Name != vault.Name {
		var existing Vault
		if err := s.db.Collection(Vault{}.TableName()).FindOne(ctx, bson.M{
			"user_id": oid,
			"name":    req.Name,
			"_id":     bson.M{"$ne": vaultOID},
		}).Decode(&existing); err == nil {
			return nil, apperr.Conflict("vault with this name already exists")
		}
	}

	now := time.Now().UTC()
	update := bson.M{
		"$set": bson.M{
			"name":        strings.TrimSpace(req.Name),
			"description": req.Description,
			"color":       req.Color,
			"icon":        req.Icon,
			"sort_order":  req.SortOrder,
			"updated_at":  now,
		},
	}

	result := s.db.Collection(Vault{}.TableName()).FindOneAndUpdate(ctx, bson.M{
		"_id":     vaultOID,
		"user_id": oid,
	}, update, optionsFindOneAndUpdateReturnAfter())

	var updated Vault
	if err := result.Decode(&updated); err != nil {
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to update vault")
	}

	return &updated, nil
}

// Delete deletes a vault and moves its secrets to the user's default vault.
func (s *Service) Delete(ctx context.Context, userID, vaultID string) error {
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return apperr.BadRequest("invalid user id")
	}

	vaultOID, err := bson.ObjectIDFromHex(vaultID)
	if err != nil {
		return apperr.BadRequest("invalid vault id")
	}

	// Verify ownership
	var vault Vault
	if err := s.db.Collection(Vault{}.TableName()).FindOne(ctx, bson.M{
		"_id":     vaultOID,
		"user_id": oid,
	}).Decode(&vault); err != nil {
		if err == mongo.ErrNoDocuments {
			return apperr.NotFound("vault not found")
		}
		return apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to fetch vault")
	}

	if vault.IsDefault {
		return apperr.BadRequest("cannot delete default vault")
	}

	// Find user's default vault to move secrets to
	var defaultVault Vault
	if err := s.db.Collection(Vault{}.TableName()).FindOne(ctx, bson.M{
		"user_id":    oid,
		"is_default": true,
	}).Decode(&defaultVault); err != nil {
		return apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to find default vault")
	}

	// Move secrets to default vault
	if _, err := s.db.Collection("secrets").UpdateMany(ctx, bson.M{
		"user_id":  oid,
		"vault_id": vaultOID,
	}, bson.M{
		"$set": bson.M{"vault_id": defaultVault.ID},
	}); err != nil {
		return apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to move secrets")
	}

	// Delete the vault
	if _, err := s.db.Collection(Vault{}.TableName()).DeleteOne(ctx, bson.M{
		"_id":     vaultOID,
		"user_id": oid,
	}); err != nil {
		return apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to delete vault")
	}

	return nil
}

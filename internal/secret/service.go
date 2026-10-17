package secret

import (
	"context"
	"net/http"
	"time"

	"github.com/AliFnieer/vaultix-backend/internal/apperr"
	"github.com/AliFnieer/vaultix-backend/internal/config"
	"github.com/AliFnieer/vaultix-backend/internal/crypto"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Service handles secret business logic.
type Service struct {
	db        *mongo.Database
	cfg       *config.Config
	encryptor *crypto.Encryptor
}

// NewService creates a new secret service.
func NewService(db *mongo.Database, cfg *config.Config) (*Service, error) {
	encryptor, err := crypto.NewEncryptor(cfg.Encryption.FieldKey)
	if err != nil {
		return nil, err
	}
	return &Service{db: db, cfg: cfg, encryptor: encryptor}, nil
}

// Create creates a new secret with encrypted fields.
func (s *Service) Create(ctx context.Context, userID string, req *CreateSecretRequest) (*Secret, error) {
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, apperr.BadRequest("invalid user id")
	}

	vaultOID, err := bson.ObjectIDFromHex(req.VaultID)
	if err != nil {
		return nil, apperr.BadRequest("invalid vault id")
	}

	// Verify vault ownership
	var vault struct {
		ID bson.ObjectID `bson:"_id"`
	}
	if err := s.db.Collection("vaults").FindOne(ctx, bson.M{
		"_id":     vaultOID,
		"user_id": oid,
	}).Decode(&vault); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperr.NotFound("vault not found")
		}
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to verify vault")
	}

	// Check for duplicate title in same vault
	var existing Secret
	if err := s.db.Collection(Secret{}.TableName()).FindOne(ctx, bson.M{
		"user_id":  oid,
		"vault_id": vaultOID,
		"title":    req.Title,
	}).Decode(&existing); err == nil {
		return nil, apperr.Conflict("secret with this title already exists in this vault")
	} else if err != mongo.ErrNoDocuments {
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to check existing secret")
	}

	// Encrypt sensitive fields
	encryptedFields, err := s.encryptFields(req.Fields)
	if err != nil {
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to encrypt fields")
	}

	now := time.Now().UTC()
	secret := Secret{
		ID:        bson.NewObjectID(),
		UserID:    oid,
		VaultID:   vaultOID,
		Title:     req.Title,
		Type:      req.Type,
		Fields:    encryptedFields,
		Tags:      req.Tags,
		Favorite:  req.Favorite,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if _, err := s.db.Collection(Secret{}.TableName()).InsertOne(ctx, secret); err != nil {
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to create secret")
	}

	return &secret, nil
}

// ListByUser returns paginated secrets for a user with optional filters.
func (s *Service) ListByUser(ctx context.Context, userID string, page, limit int, vaultID, search string, includeArchived, includeTrashed bool) (*ListSecretsResponse, error) {
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, apperr.BadRequest("invalid user id")
	}

	filter := bson.M{"user_id": oid}

	// Apply vault filter
	if vaultID != "" {
		vaultOID, err := bson.ObjectIDFromHex(vaultID)
		if err != nil {
			return nil, apperr.BadRequest("invalid vault id")
		}
		filter["vault_id"] = vaultOID
	}

	// Apply archive/trash filters
	if !includeArchived {
		filter["is_archived"] = false
	}
	if !includeTrashed {
		filter["is_trashed"] = false
	}

	// Apply search filter
	if search != "" {
		filter["title"] = bson.M{"$regex": search, "$options": "i"}
	}

	// Count total
	total, err := s.db.Collection(Secret{}.TableName()).CountDocuments(ctx, filter)
	if err != nil {
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to count secrets")
	}

	// Fetch page
	skip := int64((page - 1) * limit)
	opts := options.Find().
		SetSkip(skip).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := s.db.Collection(Secret{}.TableName()).Find(ctx, filter, opts)
	if err != nil {
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to fetch secrets")
	}
	defer cursor.Close(ctx)

	var secrets []Secret
	if err := cursor.All(ctx, &secrets); err != nil {
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to decode secrets")
	}

	// Decrypt fields for each secret
	for i := range secrets {
		decrypted, err := s.decryptFields(secrets[i].Fields)
		if err != nil {
			return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to decrypt fields")
		}
		secrets[i].Fields = decrypted
	}

	return &ListSecretsResponse{
		Secrets: secrets,
		Total:   total,
		Page:    page,
		Limit:   limit,
	}, nil
}

// GetByID retrieves a secret by ID with decrypted fields.
func (s *Service) GetByID(ctx context.Context, userID, secretID string) (*Secret, error) {
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, apperr.BadRequest("invalid user id")
	}

	secretOID, err := bson.ObjectIDFromHex(secretID)
	if err != nil {
		return nil, apperr.BadRequest("invalid secret id")
	}

	var secret Secret
	if err := s.db.Collection(Secret{}.TableName()).FindOne(ctx, bson.M{
		"_id":     secretOID,
		"user_id": oid,
	}).Decode(&secret); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperr.NotFound("secret not found")
		}
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to fetch secret")
	}

	// Decrypt fields
	decrypted, err := s.decryptFields(secret.Fields)
	if err != nil {
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to decrypt fields")
	}
	secret.Fields = decrypted

	return &secret, nil
}

// Update updates a secret's details with re-encrypted fields.
func (s *Service) Update(ctx context.Context, userID, secretID string, req *UpdateSecretRequest) (*Secret, error) {
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, apperr.BadRequest("invalid user id")
	}
	secretOID, err := bson.ObjectIDFromHex(secretID)
	if err != nil {
		return nil, apperr.BadRequest("invalid secret id")
	}
	vaultOID, err := bson.ObjectIDFromHex(req.VaultID)
	if err != nil {
		return nil, apperr.BadRequest("invalid vault id")
	}
	var existing Secret
	if err := s.db.Collection(Secret{}.TableName()).FindOne(ctx, bson.M{"_id": secretOID, "user_id": oid}).Decode(&existing); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperr.NotFound("secret not found")
		}
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to fetch secret")
	}
	encryptedFields, err := s.encryptFields(req.Fields)
	if err != nil {
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to encrypt fields")
	}
	now := time.Now().UTC()
	update := bson.M{"$set": bson.M{"title": req.Title, "type": req.Type, "vault_id": vaultOID, "fields": encryptedFields, "tags": req.Tags, "favorite": req.Favorite, "updated_at": now}}
	result := s.db.Collection(Secret{}.TableName()).FindOneAndUpdate(ctx, bson.M{"_id": secretOID, "user_id": oid}, update, optionsFindOneAndUpdateReturnAfter())
	var updated Secret
	if err := result.Decode(&updated); err != nil {
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to update secret")
	}
	decrypted, err := s.decryptFields(updated.Fields)
	if err != nil {
		return nil, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to decrypt fields")
	}
	updated.Fields = decrypted
	return &updated, nil
}

// Delete permanently deletes a secret.
func (s *Service) Delete(ctx context.Context, userID, secretID string) error {
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return apperr.BadRequest("invalid user id")
	}
	secretOID, err := bson.ObjectIDFromHex(secretID)
	if err != nil {
		return apperr.BadRequest("invalid secret id")
	}
	result, err := s.db.Collection(Secret{}.TableName()).DeleteOne(ctx, bson.M{"_id": secretOID, "user_id": oid})
	if err != nil {
		return apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to delete secret")
	}
	if result.DeletedCount == 0 {
		return apperr.NotFound("secret not found")
	}
	return nil
}

// Archive marks a secret as archived.
func (s *Service) Archive(ctx context.Context, userID, secretID string) error {
	return s.setFlag(ctx, userID, secretID, "is_archived", true)
}

// Unarchive marks a secret as not archived.
func (s *Service) Unarchive(ctx context.Context, userID, secretID string) error {
	return s.setFlag(ctx, userID, secretID, "is_archived", false)
}

// Trash marks a secret as trashed.
func (s *Service) Trash(ctx context.Context, userID, secretID string) error {
	return s.setFlag(ctx, userID, secretID, "is_trashed", true)
}

// Restore removes a secret from trash.
func (s *Service) Restore(ctx context.Context, userID, secretID string) error {
	return s.setFlag(ctx, userID, secretID, "is_trashed", false)
}

// ToggleFavorite toggles the favorite flag on a secret.
func (s *Service) ToggleFavorite(ctx context.Context, userID, secretID string) (bool, error) {
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return false, apperr.BadRequest("invalid user id")
	}
	secretOID, err := bson.ObjectIDFromHex(secretID)
	if err != nil {
		return false, apperr.BadRequest("invalid secret id")
	}
	var secret Secret
	if err := s.db.Collection(Secret{}.TableName()).FindOne(ctx, bson.M{"_id": secretOID, "user_id": oid}).Decode(&secret); err != nil {
		if err == mongo.ErrNoDocuments {
			return false, apperr.NotFound("secret not found")
		}
		return false, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to fetch secret")
	}
	newValue := !secret.Favorite
	if _, err := s.db.Collection(Secret{}.TableName()).UpdateOne(ctx, bson.M{"_id": secretOID, "user_id": oid}, bson.M{"$set": bson.M{"favorite": newValue}}); err != nil {
		return false, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to update favorite")
	}
	return newValue, nil
}

func (s *Service) setFlag(ctx context.Context, userID, secretID, field string, value bool) error {
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return apperr.BadRequest("invalid user id")
	}
	secretOID, err := bson.ObjectIDFromHex(secretID)
	if err != nil {
		return apperr.BadRequest("invalid secret id")
	}
	result, err := s.db.Collection(Secret{}.TableName()).UpdateOne(ctx, bson.M{"_id": secretOID, "user_id": oid}, bson.M{"$set": bson.M{field: value, "updated_at": time.Now().UTC()}})
	if err != nil {
		return apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to update secret")
	}
	if result.MatchedCount == 0 {
		return apperr.NotFound("secret not found")
	}
	return nil
}

// encryptFields encrypts all values in the fields map.
func (s *Service) encryptFields(fields map[string]string) (map[string]string, error) {
	encrypted := make(map[string]string, len(fields))
	for k, v := range fields {
		enc, err := s.encryptor.Encrypt(v)
		if err != nil {
			return nil, err
		}
		encrypted[k] = enc
	}
	return encrypted, nil
}

// decryptFields decrypts all values in the fields map.
func (s *Service) decryptFields(fields map[string]string) (map[string]string, error) {
	decrypted := make(map[string]string, len(fields))
	for k, v := range fields {
		dec, err := s.encryptor.Decrypt(v)
		if err != nil {
			return nil, err
		}
		decrypted[k] = dec
	}
	return decrypted, nil
}

func optionsFindOneAndUpdateReturnAfter() *options.FindOneAndUpdateOptionsBuilder {
	return options.FindOneAndUpdate().SetReturnDocument(options.After)
}

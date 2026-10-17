package audit

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Log represents an audit log entry.
type Log struct {
	ID           bson.ObjectID     `bson:"_id,omitempty" json:"id"`
	UserID       bson.ObjectID     `bson:"user_id" json:"user_id"`
	Action       string            `bson:"action" json:"action"`
	ResourceType string            `bson:"resource_type" json:"resource_type"`
	ResourceID   bson.ObjectID     `bson:"resource_id,omitempty" json:"resource_id,omitempty"`
	Metadata     map[string]string `bson:"metadata,omitempty" json:"metadata,omitempty"`
	IPAddress    string            `bson:"ip_address,omitempty" json:"ip_address,omitempty"`
	UserAgent    string            `bson:"user_agent,omitempty" json:"user_agent,omitempty"`
	CreatedAt    time.Time         `bson:"created_at" json:"created_at"`
}

// TableName returns the collection name for audit logs.
func (Log) TableName() string { return "audit_logs" }

// Action constants for audit logging.
const (
	ActionSecretCreate     = "secret.create"
	ActionSecretRead       = "secret.read"
	ActionSecretUpdate     = "secret.update"
	ActionSecretDelete     = "secret.delete"
	ActionSecretRestore    = "secret.restore"
	ActionSecretShare      = "secret.share"
	ActionVaultCreate      = "vault.create"
	ActionVaultRead        = "vault.read"
	ActionVaultUpdate      = "vault.update"
	ActionVaultDelete      = "vault.delete"
	ActionUserLogin        = "user.login"
	ActionUserRegister     = "user.register"
	ActionUserRefreshToken = "user.refresh_token"
	ActionUserLogout       = "user.logout"
)

// Resource type constants.
const (
	ResourceSecret = "secret"
	ResourceVault  = "vault"
	ResourceUser   = "user"
)

// Write is an interface for services that support audit logging.
type Writer interface {
	Write(ctx context.Context, log *Log) error
}

// Service handles audit log operations.
type Service struct {
	writer Writer
}

// NewService creates a new audit service.
func NewService(writer Writer) *Service {
	return &Service{writer: writer}
}

// Log records an audit log entry.
func (s *Service) Log(ctx context.Context, userID bson.ObjectID, action, resourceType string, resourceID *bson.ObjectID, metadata map[string]string, ip, userAgent string) error {
	entry := &Log{
		ID:           bson.NewObjectID(),
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   getObjIDOrNil(resourceID),
		Metadata:     metadata,
		IPAddress:    ip,
		UserAgent:    userAgent,
		CreatedAt:    time.Now().UTC(),
	}
	return s.writer.Write(ctx, entry)
}

func getObjIDOrNil(id *bson.ObjectID) bson.ObjectID {
	if id == nil {
		return bson.NilObjectID
	}
	return *id
}

// DBWriter implements Writer using a MongoDB database.
type DBWriter struct {
	db interface {
		Collection(name string) CollectionInterface
	}
}

// CollectionInterface abstracts the mongo collection for testability.
type CollectionInterface interface {
	InsertOne(ctx context.Context, doc interface{}) (interface{}, error)
}

// Write inserts an audit log entry into MongoDB.
func (w *DBWriter) Write(ctx context.Context, log *Log) error {
	_, err := w.db.Collection(Log{}.TableName()).InsertOne(ctx, log)
	return err
}

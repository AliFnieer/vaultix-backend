package audit

import (
	"context"
	"net/http"
	"time"

	"github.com/AliFnieer/vaultix-backend/internal/apperr"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// AuditQueryService provides read access to audit logs.
type AuditQueryService struct {
	db *mongo.Database
}

// NewAuditQueryService creates a new audit query service.
func NewAuditQueryService(db *mongo.Database) *AuditQueryService {
	return &AuditQueryService{db: db}
}

// ListByUser returns paginated audit logs for a user.
func (s *AuditQueryService) ListByUser(ctx context.Context, userID string, page, limit int, action, resourceType string) ([]Log, int64, error) {
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, apperr.BadRequest("invalid user id")
	}

	filter := bson.M{"user_id": oid}
	if action != "" {
		filter["action"] = action
	}
	if resourceType != "" {
		filter["resource_type"] = resourceType
	}

	// Count total
	total, err := s.db.Collection(Log{}.TableName()).CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to count audit logs")
	}

	// Fetch page
	skip := int64((page - 1) * limit)
	opts := options.Find().
		SetSkip(skip).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := s.db.Collection(Log{}.TableName()).Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to fetch audit logs")
	}
	defer cursor.Close(ctx)

	var logs []Log
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, 0, apperr.Wrap(err, http.StatusInternalServerError, apperr.CodeInternal, "failed to decode audit logs")
	}

	return logs, total, nil
}

// CleanupOldLogs removes audit logs older than the specified duration.
func (s *AuditQueryService) CleanupOldLogs(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-olderThan)
	result, err := s.db.Collection(Log{}.TableName()).DeleteMany(ctx, bson.M{
		"created_at": bson.M{"$lt": cutoff},
	})
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

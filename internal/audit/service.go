package audit

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// MongoWriter implements Writer using a MongoDB database.
type MongoWriter struct {
	db *mongo.Database
}

// NewMongoWriter creates a MongoDB-backed audit writer.
func NewMongoWriter(db *mongo.Database) *MongoWriter {
	return &MongoWriter{db: db}
}

// Write inserts an audit log entry into MongoDB.
func (w *MongoWriter) Write(ctx context.Context, log *Log) error {
	_, err := w.db.Collection(Log{}.TableName()).InsertOne(ctx, log)
	if err != nil {
		slog.Warn("failed to write audit log", "error", err, "action", log.Action)
		// Don't fail the main operation on audit log failure
	}
	return nil
}

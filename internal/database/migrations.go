package database

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// EnsureIndexes creates the indexes required by the vaultix collections.
// It is idempotent: indexes that already exist are safely skipped.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	if err := uniqueIndex(ctx, db.Collection("users"), "uniq_email", bson.D{{Key: "email", Value: 1}}); err != nil {
		return err
	}
	if err := uniqueIndex(ctx, db.Collection("vaults"), "uniq_vault_user_name", bson.D{
		{Key: "user_id", Value: 1},
		{Key: "name", Value: 1},
	}); err != nil {
		return err
	}
	if err := uniqueIndex(ctx, db.Collection("secrets"), "uniq_secret_user_title", bson.D{
		{Key: "user_id", Value: 1},
		{Key: "title", Value: 1},
	}); err != nil {
		return err
	}
	for _, idx := range []struct {
		name string
		keys bson.D
		opts *options.IndexOptionsBuilder
	}{
		{name: "idx_secret_user_archived", keys: bson.D{{Key: "user_id", Value: 1}, {Key: "is_archived", Value: 1}}},
		{name: "idx_secret_user_vault", keys: bson.D{{Key: "user_id", Value: 1}, {Key: "vault_id", Value: 1}}},
		{
			name: "idx_refresh_expires_at",
			keys: bson.D{{Key: "expires_at", Value: 1}},
			opts: options.Index().SetExpireAfterSeconds(0), // TTL: MongoDB deletes expired docs
		},
		{name: "idx_audit_user_created", keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
	} {
		if err := createIndex(ctx, db.Collection(collectionOf(idx.name)), idx.name, idx.keys, idx.opts); err != nil {
			return err
		}
	}
	return nil
}

func collectionOf(indexName string) string {
	switch indexName {
	case "idx_refresh_expires_at":
		return "refresh_tokens"
	case "idx_audit_user_created":
		return "audit_logs"
	default:
		return "secrets"
	}
}

// uniqueIndex creates a unique ascending index (idempotent).
func uniqueIndex(ctx context.Context, col *mongo.Collection, name string, keys bson.D) error {
	return createIndex(ctx, col, name, keys, options.Index().SetUnique(true))
}

// createIndex creates a single index, ignoring "already exists" errors.
func createIndex(ctx context.Context, col *mongo.Collection, name string, keys bson.D, opts *options.IndexOptionsBuilder) error {
	if opts == nil {
		opts = options.Index()
	}
	opts.SetName(name)
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: keys, Options: opts})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("create index %s on %s: %w", name, col.Name(), err)
	}
	return nil
}

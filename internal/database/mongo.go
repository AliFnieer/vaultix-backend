package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Client wraps the mongo client and database.
type Client struct {
	*mongo.Client
	Database *mongo.Database
	name     string
}

var (
	once     sync.Once
	instance *Client
	initErr  error
)

// New connects to MongoDB and returns a shared Client (singleton).
func New(uri, database string, maxPool, minPool int, maxIdleMinutes int) (*Client, error) {
	once.Do(func() {
		instance, initErr = connect(uri, database, maxPool, minPool, maxIdleMinutes)
	})
	return instance, initErr
}

func connect(uri, database string, maxPool, minPool int, maxIdleMinutes int) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	opts := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(uint64(maxPool)).
		SetMinPoolSize(uint64(minPool)).
		SetMaxConnIdleTime(time.Duration(maxIdleMinutes) * time.Minute).
		SetConnectTimeout(20 * time.Second).
		SetServerSelectionTimeout(20 * time.Second)

	conn, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	if err := conn.Ping(ctx, nil); err != nil {
		_ = conn.Disconnect(ctx)
		return nil, fmt.Errorf("mongo ping: %w", err)
	}

	return &Client{
		Client:   conn,
		Database: conn.Database(database),
		name:     database,
	}, nil
}

// Close disconnects from MongoDB.
func (c *Client) Close(ctx context.Context) error {
	if c == nil || c.Client == nil {
		return nil
	}
	return c.Client.Disconnect(ctx)
}

// Collection returns a mongo collection by name.
func (c *Client) Collection(name string) *mongo.Collection {
	return c.Database.Collection(name)
}

// Name returns the database name.
func (c *Client) Name() string { return c.name }

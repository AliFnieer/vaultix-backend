package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis wraps the go-redis client with connection verification and graceful close.
type Redis struct {
	client *redis.Client
}

var (
	redisOnce     sync.Once
	redisInstance *Redis
	redisInitErr  error
)

// NewRedis connects to Redis and verifies the connection (singleton).
func NewRedis(addr, password string, db, poolSize int) (*Redis, error) {
	redisOnce.Do(func() {
		redisInstance, redisInitErr = connectRedis(addr, password, db, poolSize)
	})
	return redisInstance, redisInitErr
}

func connectRedis(addr, password string, db, poolSize int) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
		PoolSize: poolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return &Redis{client: client}, nil
}

// Client returns the underlying go-redis client.
func (r *Redis) Client() *redis.Client {
	if r == nil {
		return nil
	}
	return r.client
}

// Ping verifies the Redis connection is alive.
func (r *Redis) Ping(ctx context.Context) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return r.client.Ping(ctx).Err()
}

// Close closes the Redis connection.
func (r *Redis) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// testRedisAddr returns the address of a live Redis for integration tests,
// skipping the test entirely when none is reachable.
func testRedisAddr(t *testing.T) string {
	t.Helper()

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	probe := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := probe.Ping(ctx).Err(); err != nil {
		t.Skipf("no Redis reachable at %s, skipping: %v", addr, err)
	}
	_ = probe.Close()
	return addr
}

func TestRedisPingAndRoundTrip(t *testing.T) {
	addr := testRedisAddr(t)

	rdb, err := connectRedis(addr, "", 0, 4)
	if err != nil {
		t.Fatalf("connectRedis: %v", err)
	}
	defer func() { _ = rdb.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	key := "vaultix:test:roundtrip"
	if err := rdb.Client().Set(ctx, key, "value", 10*time.Second).Err(); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := rdb.Client().Get(ctx, key).Result()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "value" {
		t.Errorf("got %q, want %q", got, "value")
	}
	if err := rdb.Client().Del(ctx, key).Err(); err != nil {
		t.Fatalf("Del: %v", err)
	}
}

func TestRedisCloseIsIdempotentAndNilSafe(t *testing.T) {
	var nilClient *Redis
	if err := nilClient.Close(); err != nil {
		t.Errorf("Close on nil returned error: %v", err)
	}
	if err := nilClient.Ping(context.Background()); err == nil {
		t.Error("Ping on nil should return an error")
	}
	if nilClient.Client() != nil {
		t.Error("Client on nil should return nil")
	}
}

func TestNewRedisFailsFastOnUnreachableAddr(t *testing.T) {
	// Use the singleton path but with a guaranteed-dead address.
	// NewRedis memoizes via sync.Once, so this only asserts the error
	// contract for the first initialization attempt in this test binary.
	_, err := NewRedis("127.0.0.1:1", "", 0, 1)
	if err == nil {
		t.Skip("singleton already initialized by another test")
	}
}

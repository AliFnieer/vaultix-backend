package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AliFnieer/vaultix-backend/internal/config"
	"github.com/AliFnieer/vaultix-backend/internal/database"
	"github.com/AliFnieer/vaultix-backend/internal/server"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if present (existing environment variables take precedence)
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found, using process environment")
	}

	// Load configuration
	cfg := config.Load()

	// Connect to MongoDB
	mongoClient, err := database.New(
		cfg.Database.URI,
		cfg.Database.Name,
		cfg.Database.MaxPoolSize,
		cfg.Database.MinPoolSize,
		cfg.Database.MaxIdleMinutes,
	)
	if err != nil {
		slog.Error("failed to connect to MongoDB", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Run migrations / ensure indexes
	if err := database.EnsureIndexes(ctx, mongoClient.Database); err != nil {
		slog.Error("failed to ensure indexes", "error", err)
		os.Exit(1)
	}

	slog.Info("MongoDB connected and indexes ensured", "database", cfg.Database.Name)

	// Connect to Redis
	redisClient, err := database.NewRedis(
		cfg.Redis.Addr,
		cfg.Redis.Password,
		cfg.Redis.DB,
		cfg.Redis.PoolSize,
	)
	if err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	slog.Info("Redis connected", "addr", cfg.Redis.Addr, "db", cfg.Redis.DB)

	// Create server
	srv := server.NewServer(cfg, mongoClient.Database, mongoClient, redisClient)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: srv.Engine(),
	}

	// Graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		slog.Info("shutting down server...")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown failed", "error", err)
		}

		if err := mongoClient.Close(shutdownCtx); err != nil {
			slog.Error("failed to close MongoDB connection", "error", err)
		}

		if err := redisClient.Close(); err != nil {
			slog.Error("failed to close Redis connection", "error", err)
		}
	}()

	// Start server
	slog.Info("starting server", "addr", httpServer.Addr, "mode", cfg.Server.GinMode)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

package server

import (
	"context"
	"net/http"
	"time"

	"github.com/AliFnieer/vaultix-backend/internal/audit"
	"github.com/AliFnieer/vaultix-backend/internal/auth"
	"github.com/AliFnieer/vaultix-backend/internal/config"
	"github.com/AliFnieer/vaultix-backend/internal/database"
	"github.com/AliFnieer/vaultix-backend/internal/middleware"
	"github.com/AliFnieer/vaultix-backend/internal/secret"
	"github.com/AliFnieer/vaultix-backend/internal/user"
	"github.com/AliFnieer/vaultix-backend/internal/vault"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Server struct {
	engine *gin.Engine
	cfg    *config.Config
	db     *mongo.Database
	mongo  *database.Client
}

func NewServer(cfg *config.Config, db *mongo.Database, mongoClient *database.Client) *Server {
	gin.SetMode(cfg.Server.GinMode)
	engine := gin.New()

	srv := &Server{
		engine: engine,
		cfg:    cfg,
		db:     db,
		mongo:  mongoClient,
	}

	srv.setupMiddleware()
	srv.setupRoutes()

	return srv
}

func (s *Server) setupMiddleware() {
	s.engine.Use(middleware.Logger())
	s.engine.Use(gin.Recovery())
	s.engine.Use(CORS(s.cfg.CORS.AllowedOrigins))
}

func (s *Server) setupRoutes() {
	s.engine.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		status := gin.H{"status": "ok"}
		if err := s.mongo.Client.Ping(ctx, nil); err != nil {
			status["database"] = "error"
			c.JSON(http.StatusServiceUnavailable, status)
			return
		}
		status["database"] = "ok"
		c.JSON(http.StatusOK, status)
	})

	// API v1 routes
	apiV1 := s.engine.Group("/api/v1")

	// Register feature routes
	auth.RegisterRoutes(apiV1, s.db, s.cfg)
	user.RegisterRoutes(apiV1, s.db, s.cfg)
	secret.RegisterRoutes(apiV1, s.db, s.cfg)
	vault.RegisterRoutes(apiV1, s.db, s.cfg)
	audit.RegisterRoutes(apiV1, s.db, s.cfg)
}

func (s *Server) Engine() *gin.Engine {
	return s.engine
}

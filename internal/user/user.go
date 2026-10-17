package user

import (
	"context"
	"net/http"

	"github.com/AliFnieer/vaultix-backend/internal/apperr"
	"github.com/AliFnieer/vaultix-backend/internal/auth"
	"github.com/AliFnieer/vaultix-backend/internal/config"
	"github.com/AliFnieer/vaultix-backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Service handles user-specific business logic.
type Service struct {
	db  *mongo.Database
	cfg *config.Config
}

// NewService creates a new user service.
func NewService(db *mongo.Database, cfg *config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

// Controller handles user HTTP requests.
type Controller struct {
	service *Service
}

// NewController creates a new user controller.
func NewController(service *Service) *Controller {
	return &Controller{service: service}
}

// GetProfile handles GET /api/v1/users/profile
func (c *Controller) GetProfile(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	userIDStr, ok := userID.(string)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return
	}

	// Use auth service to get user
	authService := auth.NewService(c.service.db, c.service.cfg)
	user, err := authService.GetByID(ctx.Request.Context(), userIDStr)
	if err != nil {
		appErr := apperr.FromError(err)
		ctx.JSON(appErr.Status, gin.H{"error": appErr.Message, "code": appErr.Code})
		return
	}

	ctx.JSON(http.StatusOK, user)
}

// RegisterRoutes registers user routes on the given router group.
func RegisterRoutes(router *gin.RouterGroup, db *mongo.Database, cfg *config.Config) {
	service := NewService(db, cfg)
	controller := NewController(service)

	// Protected routes (require JWT)
	usersGroup := router.Group("/users")
	usersGroup.Use(middleware.AuthMiddleware(cfg))
	{
		usersGroup.GET("/profile", controller.GetProfile)
	}
}

var _ = context.Background

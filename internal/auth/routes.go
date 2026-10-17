package auth

import (
	"github.com/AliFnieer/vaultix-backend/internal/config"
	"github.com/AliFnieer/vaultix-backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// RegisterRoutes registers auth routes on the given router group.
func RegisterRoutes(router *gin.RouterGroup, db *mongo.Database, cfg *config.Config) {
	service := NewService(db, cfg)
	controller := NewController(service)

	// Public routes
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", controller.Register)
		authGroup.POST("/login", controller.Login)
		authGroup.POST("/refresh", controller.Refresh)
	}

	// Protected routes (require JWT)
	protected := router.Group("")
	protected.Use(middleware.AuthMiddleware(cfg))
	{
		protected.GET("/auth/me", controller.Me)
		protected.GET("/users/:id", controller.LookupUser)
		protected.POST("/auth/logout", controller.Logout)
	}
}

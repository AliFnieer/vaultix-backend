package vault

import (
	"github.com/AliFnieer/vaultix-backend/internal/config"
	"github.com/AliFnieer/vaultix-backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// RegisterRoutes registers vault routes on the given router group.
func RegisterRoutes(router *gin.RouterGroup, db *mongo.Database, cfg *config.Config) {
	service := NewService(db, cfg)
	controller := NewController(service)

	// Protected routes (require JWT)
	vaultsGroup := router.Group("/vaults")
	vaultsGroup.Use(middleware.AuthMiddleware(cfg))
	{
		vaultsGroup.POST("", controller.Create)
		vaultsGroup.GET("", controller.List)
		vaultsGroup.GET("/:id", controller.GetByID)
		vaultsGroup.PUT("/:id", controller.Update)
		vaultsGroup.DELETE("/:id", controller.Delete)
	}
}

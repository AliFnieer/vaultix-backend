package secret

import (
	"github.com/AliFnieer/vaultix-backend/internal/config"
	"github.com/AliFnieer/vaultix-backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// RegisterRoutes registers secret routes on the given router group.
func RegisterRoutes(router *gin.RouterGroup, db *mongo.Database, cfg *config.Config) {
	service, err := NewService(db, cfg)
	if err != nil {
		panic("failed to create secret service: " + err.Error())
	}
	controller := NewController(service)

	// Protected routes (require JWT)
	secretsGroup := router.Group("/secrets")
	secretsGroup.Use(middleware.AuthMiddleware(cfg))
	{
		secretsGroup.POST("", controller.Create)
		secretsGroup.GET("", controller.List)
		secretsGroup.GET("/:id", controller.GetByID)
		secretsGroup.PUT("/:id", controller.Update)
		secretsGroup.DELETE("/:id", controller.Delete)
		secretsGroup.POST("/:id/archive", controller.Archive)
		secretsGroup.POST("/:id/unarchive", controller.Unarchive)
		secretsGroup.POST("/:id/trash", controller.Trash)
		secretsGroup.POST("/:id/restore", controller.Restore)
		secretsGroup.POST("/:id/favorite", controller.ToggleFavorite)
	}
}

package audit

import (
	"github.com/AliFnieer/vaultix-backend/internal/config"
	"github.com/AliFnieer/vaultix-backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// RegisterRoutes registers audit routes on the given router group.
func RegisterRoutes(router *gin.RouterGroup, db *mongo.Database, cfg *config.Config) {
	writer := NewMongoWriter(db)
	_ = NewService(writer) // audit writer service
	queryService := NewAuditQueryService(db)
	controller := NewController(queryService)

	// Protected routes (require JWT)
	auditGroup := router.Group("/audit")
	auditGroup.Use(middleware.AuthMiddleware(cfg))
	{
		auditGroup.GET("/logs", controller.List)
	}
}

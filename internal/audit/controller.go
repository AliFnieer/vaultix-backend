package audit

import (
	"net/http"
	"strconv"

	"github.com/AliFnieer/vaultix-backend/internal/apperr"
	"github.com/gin-gonic/gin"
)

// Controller handles audit HTTP requests.
type Controller struct {
	service *AuditQueryService
}

// NewController creates a new audit controller.
func NewController(service *AuditQueryService) *Controller {
	return &Controller{service: service}
}

// List handles GET /api/v1/audit/logs
func (c *Controller) List(ctx *gin.Context) {
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

	// Parse pagination
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Optional filters
	action := ctx.Query("action")
	resourceType := ctx.Query("resource_type")

	logs, total, err := c.service.ListByUser(ctx.Request.Context(), userIDStr, page, limit, action, resourceType)
	if err != nil {
		appErr := apperr.FromError(err)
		ctx.JSON(appErr.Status, gin.H{"error": appErr.Message, "code": appErr.Code})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

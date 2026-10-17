package secret

import (
	"net/http"
	"strconv"

	"github.com/AliFnieer/vaultix-backend/internal/apperr"
	"github.com/gin-gonic/gin"
)

// Controller handles secret HTTP requests.
type Controller struct {
	service *Service
}

// NewController creates a new secret controller.
func NewController(service *Service) *Controller {
	return &Controller{service: service}
}

// getAuthenticatedUserID extracts the user ID from the Gin context.
func getAuthenticatedUserID(ctx *gin.Context) (string, bool) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return "", false
	}
	userIDStr, ok := userID.(string)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return "", false
	}
	return userIDStr, true
}

// respondError writes an error response using AppError.
func respondError(ctx *gin.Context, err error) {
	appErr := apperr.FromError(err)
	ctx.JSON(appErr.Status, gin.H{"error": appErr.Message, "code": appErr.Code})
}

// Create handles POST /api/v1/secrets
func (c *Controller) Create(ctx *gin.Context) {
	userIDStr, ok := getAuthenticatedUserID(ctx)
	if !ok {
		return
	}

	var req CreateSecretRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	secret, err := c.service.Create(ctx.Request.Context(), userIDStr, &req)
	if err != nil {
		respondError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, secret)
}

// List handles GET /api/v1/secrets
func (c *Controller) List(ctx *gin.Context) {
	userIDStr, ok := getAuthenticatedUserID(ctx)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	vaultID := ctx.Query("vault_id")
	search := ctx.Query("search")
	includeArchived := ctx.Query("include_archived") == "true"
	includeTrashed := ctx.Query("include_trashed") == "true"

	resp, err := c.service.ListByUser(ctx.Request.Context(), userIDStr, page, limit, vaultID, search, includeArchived, includeTrashed)
	if err != nil {
		respondError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// GetByID handles GET /api/v1/secrets/:id
func (c *Controller) GetByID(ctx *gin.Context) {
	userIDStr, ok := getAuthenticatedUserID(ctx)
	if !ok {
		return
	}

	secretID := ctx.Param("id")
	secret, err := c.service.GetByID(ctx.Request.Context(), userIDStr, secretID)
	if err != nil {
		respondError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, secret)
}

// Update handles PUT /api/v1/secrets/:id
func (c *Controller) Update(ctx *gin.Context) {
	userIDStr, ok := getAuthenticatedUserID(ctx)
	if !ok {
		return
	}

	secretID := ctx.Param("id")
	var req UpdateSecretRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	secret, err := c.service.Update(ctx.Request.Context(), userIDStr, secretID, &req)
	if err != nil {
		respondError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, secret)
}

// Delete handles DELETE /api/v1/secrets/:id
func (c *Controller) Delete(ctx *gin.Context) {
	userIDStr, ok := getAuthenticatedUserID(ctx)
	if !ok {
		return
	}

	secretID := ctx.Param("id")
	if err := c.service.Delete(ctx.Request.Context(), userIDStr, secretID); err != nil {
		respondError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "secret deleted"})
}

// Archive handles POST /api/v1/secrets/:id/archive
func (c *Controller) Archive(ctx *gin.Context) {
	userIDStr, ok := getAuthenticatedUserID(ctx)
	if !ok {
		return
	}

	secretID := ctx.Param("id")
	if err := c.service.Archive(ctx.Request.Context(), userIDStr, secretID); err != nil {
		respondError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "secret archived"})
}

// Unarchive handles POST /api/v1/secrets/:id/unarchive
func (c *Controller) Unarchive(ctx *gin.Context) {
	userIDStr, ok := getAuthenticatedUserID(ctx)
	if !ok {
		return
	}

	secretID := ctx.Param("id")
	if err := c.service.Unarchive(ctx.Request.Context(), userIDStr, secretID); err != nil {
		respondError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "secret unarchived"})
}

// Trash handles POST /api/v1/secrets/:id/trash
func (c *Controller) Trash(ctx *gin.Context) {
	userIDStr, ok := getAuthenticatedUserID(ctx)
	if !ok {
		return
	}

	secretID := ctx.Param("id")
	if err := c.service.Trash(ctx.Request.Context(), userIDStr, secretID); err != nil {
		respondError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "secret moved to trash"})
}

// Restore handles POST /api/v1/secrets/:id/restore
func (c *Controller) Restore(ctx *gin.Context) {
	userIDStr, ok := getAuthenticatedUserID(ctx)
	if !ok {
		return
	}

	secretID := ctx.Param("id")
	if err := c.service.Restore(ctx.Request.Context(), userIDStr, secretID); err != nil {
		respondError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "secret restored"})
}

// ToggleFavorite handles POST /api/v1/secrets/:id/favorite
func (c *Controller) ToggleFavorite(ctx *gin.Context) {
	userIDStr, ok := getAuthenticatedUserID(ctx)
	if !ok {
		return
	}

	secretID := ctx.Param("id")
	favorite, err := c.service.ToggleFavorite(ctx.Request.Context(), userIDStr, secretID)
	if err != nil {
		respondError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"favorite": favorite})
}

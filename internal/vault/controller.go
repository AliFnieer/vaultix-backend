package vault

import (
	"net/http"

	"github.com/AliFnieer/vaultix-backend/internal/apperr"
	"github.com/gin-gonic/gin"
)

// Controller handles vault HTTP requests.
type Controller struct {
	service *Service
}

// NewController creates a new vault controller.
func NewController(service *Service) *Controller {
	return &Controller{service: service}
}

// Create handles POST /api/v1/vaults
func (c *Controller) Create(ctx *gin.Context) {
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

	var req CreateVaultRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vault, err := c.service.Create(ctx.Request.Context(), userIDStr, &req)
	if err != nil {
		appErr := apperr.FromError(err)
		ctx.JSON(appErr.Status, gin.H{"error": appErr.Message, "code": appErr.Code})
		return
	}

	ctx.JSON(http.StatusCreated, vault)
}

// List handles GET /api/v1/vaults
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

	vaults, err := c.service.ListByUser(ctx.Request.Context(), userIDStr)
	if err != nil {
		appErr := apperr.FromError(err)
		ctx.JSON(appErr.Status, gin.H{"error": appErr.Message, "code": appErr.Code})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"vaults": vaults})
}

// GetByID handles GET /api/v1/vaults/:id
func (c *Controller) GetByID(ctx *gin.Context) {
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

	vaultID := ctx.Param("id")
	vault, err := c.service.GetByID(ctx.Request.Context(), userIDStr, vaultID)
	if err != nil {
		appErr := apperr.FromError(err)
		ctx.JSON(appErr.Status, gin.H{"error": appErr.Message, "code": appErr.Code})
		return
	}

	ctx.JSON(http.StatusOK, vault)
}

// Update handles PUT /api/v1/vaults/:id
func (c *Controller) Update(ctx *gin.Context) {
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

	vaultID := ctx.Param("id")
	var req UpdateVaultRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vault, err := c.service.Update(ctx.Request.Context(), userIDStr, vaultID, &req)
	if err != nil {
		appErr := apperr.FromError(err)
		ctx.JSON(appErr.Status, gin.H{"error": appErr.Message, "code": appErr.Code})
		return
	}

	ctx.JSON(http.StatusOK, vault)
}

// Delete handles DELETE /api/v1/vaults/:id
func (c *Controller) Delete(ctx *gin.Context) {
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

	vaultID := ctx.Param("id")
	if err := c.service.Delete(ctx.Request.Context(), userIDStr, vaultID); err != nil {
		appErr := apperr.FromError(err)
		ctx.JSON(appErr.Status, gin.H{"error": appErr.Message, "code": appErr.Code})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "vault deleted"})
}

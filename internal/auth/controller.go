package auth

import (
	"net/http"
	"strconv"

	"github.com/AliFnieer/vaultix-backend/internal/apperr"
	"github.com/gin-gonic/gin"
	log "log/slog"
)

// Controller handles auth HTTP requests.
type Controller struct {
	service *Service
}

// NewController creates a new auth controller.
func NewController(service *Service) *Controller {
	return &Controller{service: service}
}

// Register handles POST /api/v1/auth/register
func (c *Controller) Register(ctx *gin.Context) {
	var req RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := c.service.Register(ctx.Request.Context(), &req)
	if err != nil {
		appErr := apperr.FromError(err)
		log.Error("auth register failed", "error", err)
		ctx.JSON(appErr.Status, gin.H{"error": appErr.Message, "code": appErr.Code})
		return
	}

	ctx.JSON(http.StatusCreated, resp)
}

// Login handles POST /api/v1/auth/login
func (c *Controller) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := c.service.Login(ctx.Request.Context(), &req)
	if err != nil {
		appErr := apperr.FromError(err)
		log.Error("auth login failed", "error", err)
		ctx.JSON(appErr.Status, gin.H{"error": appErr.Message, "code": appErr.Code})
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// Refresh handles POST /api/v1/auth/refresh
func (c *Controller) Refresh(ctx *gin.Context) {
	var req RefreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := c.service.Refresh(ctx.Request.Context(), &req)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// Logout handles POST /api/v1/auth/logout
func (c *Controller) Logout(ctx *gin.Context) {
	userID, ok := middlewareGetUserID(ctx)
	if !ok {
		return
	}

	if err := c.service.Logout(ctx.Request.Context(), userID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

// Me handles GET /api/v1/auth/me
func (c *Controller) Me(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id in context"})
		return
	}

	user, err := c.service.GetByID(ctx.Request.Context(), userIDStr)
	if err != nil {
		appErr := apperr.FromError(err)
		ctx.JSON(appErr.Status, gin.H{"error": appErr.Message, "code": appErr.Code})
		return
	}

	ctx.JSON(http.StatusOK, user)
}

// LookupUser handles GET /api/v1/users/:id
func (c *Controller) LookupUser(ctx *gin.Context) {
	idStr := ctx.Param("id")
	_, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id parameter"})
		return
	}

	// ObjectID hex is 24 chars
	if len(idStr) != 24 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id parameter"})
		return
	}

	user, err := c.service.GetByID(ctx.Request.Context(), idStr)
	if err != nil {
		appErr := apperr.FromError(err)
		ctx.JSON(appErr.Status, gin.H{"error": appErr.Message, "code": appErr.Code})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"id":         user.ID.Hex(),
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"email":      user.Email,
		"role":       user.Role,
	})
}

func middlewareGetUserID(c *gin.Context) (string, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return "", false
	}
	id, ok := value.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id in context"})
		return "", false
	}
	return id, ok
}

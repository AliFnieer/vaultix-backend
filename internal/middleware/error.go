package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/AliFnieer/vaultix-backend/internal/apperr"
	"github.com/gin-gonic/gin"
)

// ErrorHandler is a Gin middleware that recovers from panics and converts
// domain errors (AppError) into standardized JSON error responses.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				debug.PrintStack()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				c.Abort()
				return
			}
		}()
		c.Next()

		// Handle errors stored in the context (c.Error / AbortWithError).
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			appErr := apperr.FromError(err)
			c.JSON(appErr.Status, gin.H{"error": appErr.Message, "code": appErr.Code})
		}
	}
}

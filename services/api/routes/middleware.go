package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
)

// AuthMiddleware extracts tenant ID from authentication context
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// In production, this would validate JWT, mTLS, or session
		// For now, we extract tenant from a header or context
		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID == "" {
			// Try to get from context (set by auth middleware)
			tenantID = c.MustGet("tenant_id").(string)
		}

		if tenantID == "" {
			c.JSON(http.StatusUnauthorized, errorResponse{Error: "tenant identifier required"})
			c.Abort()
			return
		}

		c.Set("tenant_id", tenantID)
		c.Next()
	}
}

// ErrorResponse returns a JSON error response
func ErrorResponse(c *gin.Context, status int, message string) {
	c.JSON(status, errorResponse{Error: message})
}

// SuccessResponse returns a JSON success response
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}
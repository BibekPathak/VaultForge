package routes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vaultforge/vaultforge/services/api/core"
)

// AuthMiddleware extracts tenant ID from authentication context.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID == "" {
			c.JSON(http.StatusUnauthorized, errorResponse{Error: "tenant identifier required"})
			c.Abort()
			return
		}

		c.Set("tenant_id", tenantID)
		c.Next()
	}
}

// RequestIDMiddleware assigns a unique request ID if not already present.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// MetricsMiddleware records request metrics and latency.
func MetricsMiddleware(metrics *core.MetricsCollector) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		elapsed := time.Since(start)
		metrics.IncrRequestCount()
		metrics.RecordLatency(elapsed)
		if c.Writer.Status() >= 400 {
			metrics.IncrRequestErrors()
		}
	}
}

// LoggingMiddleware logs structured request/response information.
func LoggingMiddleware(logger *core.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID, _ := c.Get("request_id")
		reqIDStr, _ := requestID.(string)

		logger.LogRequestStart(reqIDStr, c.Request.Method, c.Request.URL.Path)

		c.Next()

		elapsed := time.Since(start)
		logger.LogRequestComplete(
			reqIDStr,
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			elapsed,
		)
	}
}

// ErrorResponse returns a JSON error response.
func ErrorResponse(c *gin.Context, status int, message string) {
	c.JSON(status, errorResponse{Error: message})
}

// SuccessResponse returns a JSON success response.
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

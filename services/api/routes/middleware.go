package routes

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vaultforge/vaultforge/services/api/core"
)

// ApiError is a structured error response with code and request correlation.
type ApiError struct {
	Error     string `json:"error"`
	Code      string `json:"code"`
	RequestID string `json:"request_id,omitempty"`
}

// Error codes used across the API.
const (
	CodeBadRequest       = "BAD_REQUEST"
	CodeUnauthorized     = "UNAUTHORIZED"
	CodeForbidden        = "FORBIDDEN"
	CodeNotFound         = "NOT_FOUND"
	CodeConflict         = "CONFLICT"
	CodeInternalError    = "INTERNAL_ERROR"
	CodePolicyDenied     = "POLICY_DENIED"
	CodeZKFailed         = "ZK_VERIFICATION_FAILED"
	CodeSigningFailed    = "SIGNING_FAILED"
	CodeSubmissionFailed = "SUBMISSION_FAILED"
	CodeExpired          = "INTENT_EXPIRED"
	CodeRateLimited      = "RATE_LIMITED"
	CodeTimeout          = "REQUEST_TIMEOUT"
)

// AuthMiddleware extracts tenant ID from authentication context.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID == "" {
			respondError(c, http.StatusUnauthorized, CodeUnauthorized, "tenant identifier required")
			c.Abort()
			return
		}

		c.Set("tenant_id", tenantID)
		c.Next()
	}
}

// CORSMiddleware sets CORS headers for browser-based clients.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID, X-Request-ID, X-Actor")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID, X-RateLimit-Remaining")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RateLimitMiddleware enforces per-tenant rate limiting.
func RateLimitMiddleware(limiter *core.RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID == "" {
			tenantID = "__anonymous__"
		}
		if !limiter.Allow(tenantID) {
			respondError(c, http.StatusTooManyRequests, CodeRateLimited, "rate limit exceeded")
			c.Abort()
			return
		}
		c.Next()
	}
}

// BodyLimitMiddleware wraps core.MaxBodyBytesMiddleware for use with Gin.
func BodyLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
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

// TimeoutMiddleware enforces a per-request timeout.
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()

		if ctx.Err() == context.DeadlineExceeded {
			respondError(c, http.StatusGatewayTimeout, CodeTimeout, "request timed out")
			c.Abort()
		}
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

// respondError writes a structured API error response.
func respondError(c *gin.Context, status int, code, message string) {
	requestID, _ := c.Get("request_id")
	reqIDStr, _ := requestID.(string)
	c.JSON(status, ApiError{
		Error:     message,
		Code:      code,
		RequestID: reqIDStr,
	})
}

// ErrorResponse returns a structured JSON error response (legacy helper).
func ErrorResponse(c *gin.Context, status int, message string) {
	respondError(c, status, CodeInternalError, message)
}

// SuccessResponse returns a JSON success response.
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

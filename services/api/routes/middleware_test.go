package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/vaultforge/vaultforge/services/api/core"
)

func TestAuthMiddleware_MissingTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_PresentTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware())
	r.GET("/test", func(c *gin.Context) {
		tenantID, _ := c.Get("tenant_id")
		c.JSON(200, gin.H{"tenant": tenantID})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestIDMiddleware())
	r.GET("/test", func(c *gin.Context) {
		rid, _ := c.Get("request_id")
		c.JSON(200, gin.H{"request_id": rid})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	headerID := w.Header().Get("X-Request-ID")
	if headerID == "" {
		t.Error("X-Request-ID header should be set")
	}
}

func TestRequestIDMiddleware_PreservesExisting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestIDMiddleware())
	r.GET("/test", func(c *gin.Context) {
		rid, _ := c.Get("request_id")
		c.JSON(200, gin.H{"request_id": rid})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "my-custom-id")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") != "my-custom-id" {
		t.Errorf("expected custom ID preserved, got %s", w.Header().Get("X-Request-ID"))
	}
}

func TestMetricsMiddleware_Counts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	metrics := core.NewMetricsCollector()
	r := gin.New()
	r.Use(MetricsMiddleware(metrics))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	snap := metrics.Snapshot()
	if snap.RequestCount != 1 {
		t.Errorf("expected request count=1, got %d", snap.RequestCount)
	}
}

func TestMetricsMiddleware_ErrorCount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	metrics := core.NewMetricsCollector()
	r := gin.New()
	r.Use(MetricsMiddleware(metrics))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(500, gin.H{"error": "fail"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	snap := metrics.Snapshot()
	if snap.RequestCount != 1 {
		t.Errorf("expected request count=1, got %d", snap.RequestCount)
	}
	if snap.RequestErrors != 1 {
		t.Errorf("expected error count=1, got %d", snap.RequestErrors)
	}
}

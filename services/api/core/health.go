package core

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthChecker provides liveness and readiness probe endpoints.
type HealthChecker struct {
	db *gorm.DB
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(db *gorm.DB) *HealthChecker {
	return &HealthChecker{db: db}
}

// HealthStatus represents the health of a component.
type HealthStatus struct {
	Status  string            `json:"status"`
	Checks  map[string]string `json:"checks,omitempty"`
}

// RegisterRoutes registers health check endpoints on the given router group.
func (h *HealthChecker) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/health", h.liveness)
	r.GET("/ready", h.readiness)
}

// liveness returns 200 if the process is alive.
func (h *HealthChecker) liveness(c *gin.Context) {
	c.JSON(http.StatusOK, HealthStatus{Status: "ok"})
}

// readiness returns 200 only if all critical dependencies are reachable.
func (h *HealthChecker) readiness(c *gin.Context) {
	checks := make(map[string]string)
	healthy := true

	if err := h.checkDB(); err != nil {
		checks["database"] = "error: " + err.Error()
		healthy = false
	} else {
		checks["database"] = "ok"
	}

	status := http.StatusOK
	if !healthy {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, HealthStatus{
		Status: map[bool]string{true: "ok", false: "degraded"}[healthy],
		Checks: checks,
	})
}

func (h *HealthChecker) checkDB() error {
	sqlDB, err := h.db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(5 * time.Second)
	return sqlDB.Ping()
}

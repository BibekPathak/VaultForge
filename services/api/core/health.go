package core

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthChecker provides liveness and readiness probe endpoints.
type HealthChecker struct {
	db           *gorm.DB
	solanaClient SolanaSubmitter
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(db *gorm.DB, solanaClient SolanaSubmitter) *HealthChecker {
	return &HealthChecker{db: db, solanaClient: solanaClient}
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

	if err := h.checkSolana(); err != nil {
		checks["solana_rpc"] = "error: " + err.Error()
		healthy = false
	} else {
		checks["solana_rpc"] = "ok"
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

func (h *HealthChecker) checkSolana() error {
	if h.solanaClient == nil {
		return nil
	}
	_, err := h.solanaClient.GetRecentBlockhash()
	return err
}

// DBPoolStats returns current connection pool statistics.
type DBPoolStats struct {
	OpenConnections int `json:"open_connections"`
	InUse           int `json:"in_use"`
	Idle            int `json:"idle"`
	WaitCount       int64 `json:"wait_count"`
	WaitDuration    string `json:"wait_duration"`
	MaxIdleClosed   int64 `json:"max_idle_closed"`
	MaxIdleTimeClosed int64 `json:"max_idle_time_closed"`
	MaxLifetimeClosed  int64 `json:"max_lifetime_closed"`
}

// GetDBPoolStats returns the current database connection pool statistics.
func GetDBPoolStats(db *gorm.DB) *DBPoolStats {
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Failed to get underlying SQL DB for stats: %v", err)
		return nil
	}
	stats := sqlDB.Stats()
	return &DBPoolStats{
		OpenConnections:   stats.OpenConnections,
		InUse:             stats.InUse,
		Idle:              stats.Idle,
		WaitCount:         stats.WaitCount,
		WaitDuration:      stats.WaitDuration.String(),
		MaxIdleClosed:     stats.MaxIdleClosed,
		MaxIdleTimeClosed: stats.MaxIdleTimeClosed,
		MaxLifetimeClosed: stats.MaxLifetimeClosed,
	}
}

// sqlDB is an alias for the database/sql.DB type used in pool stats.
type sqlDB = sql.DB

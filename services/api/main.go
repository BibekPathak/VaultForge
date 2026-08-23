package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vaultforge/vaultforge/services/api/core"
	"github.com/vaultforge/vaultforge/services/api/routes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load and validate configuration
	cfg := core.LoadConfig()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	logger := core.NewLogger(cfg.Environment, cfg.LogLevel)
	logger.Info("starting VaultForge API",
		"port", cfg.Port,
		"environment", cfg.Environment,
		"solana_rpc", cfg.SolanaRPCURL,
	)

	// Initialize database
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("failed to get underlying SQL DB", "error", err)
		os.Exit(1)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// Migrate schema
	err = db.AutoMigrate(
		&core.Tenant{},
		&core.Wallet{},
		&core.Intent{},
		&core.Transaction{},
		&core.AuditEvent{},
		&core.Policy{},
		&core.MPCShareRecord{},
		&core.ReplayKey{},
		&core.WebhookEndpoint{},
	)
	if err != nil {
		logger.Error("failed to migrate database", "error", err)
		os.Exit(1)
	}
	logger.Info("database migration complete")

	// Initialize metrics collector
	metrics := core.NewMetricsCollector()

	// Initialize stores
	intentStore := core.NewPostgresIntentStore(db)

	// Initialize core services
	dbAdapter := core.NewDBAdapter(db)
	policyEngine := core.NewPolicyEngine(dbAdapter)
	zkVerifier := core.NewZKVerifier()
	mpcSigner := core.NewMPCSigner(db)
	reconciler := core.NewReconciler(db)
	audit := core.NewAuditLogger(db)
	solanaClient := core.NewSolanaClient(cfg.SolanaRPCURL)
	webhooks := core.NewWebhookNotifier(db)
	txStore := core.NewPostgresTransactionStore(db)
	healthChecker := core.NewHealthChecker(db, solanaClient)

	// Initialize rate limiter (100 req/s per tenant, burst of 200)
	rateLimiter := core.NewRateLimiter(100, 200)

	// Create HTTP router with middleware stack
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(routes.CORSMiddleware())
	r.Use(routes.RequestIDMiddleware())
	r.Use(routes.TimeoutMiddleware(30 * time.Second))
	r.Use(routes.BodyLimitMiddleware(cfg.MaxBodyBytes))
	r.Use(routes.RateLimitMiddleware(rateLimiter))
	r.Use(routes.MetricsMiddleware(metrics))
	r.Use(routes.LoggingMiddleware(logger))
	r.Use(routes.AuthMiddleware())

	// Health check endpoints (no auth required)
	healthGroup := r.Group("")
	healthChecker.RegisterRoutes(healthGroup)

	// Metrics endpoint with DB pool stats
	r.GET("/metrics", func(c *gin.Context) {
		poolStats := core.GetDBPoolStats(db)
		c.JSON(http.StatusOK, metrics.SnapshotWithDB(poolStats))
	})

	// Create handler and register routes
	handler := routes.NewIntentHandler(
		intentStore,
		policyEngine,
		zkVerifier,
		mpcSigner,
		reconciler,
		audit,
		solanaClient,
		webhooks,
		txStore,
	)

	v1 := r.Group("/v1")
	handler.RegisterRoutes(v1)

	// Configure HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("VaultForge API listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("shutdown signal received", "signal", sig.String(),
		"goroutines", runtime.NumGoroutine())

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	// Wait briefly for in-flight requests to drain
	time.Sleep(500 * time.Millisecond)

	logger.Info("draining complete", "goroutines_remaining", runtime.NumGoroutine())

	// Close database connection
	if err := sqlDB.Close(); err != nil {
		logger.Error("failed to close database connection", "error", err)
	}

	logger.Info("VaultForge API stopped gracefully")
}

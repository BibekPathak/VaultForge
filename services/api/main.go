package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vaultforge/vaultforge/services/api"
	"github.com/vaultforge/vaultforge/services/api/core"
	"github.com/vaultforge/vaultforge/services/api/routes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Read configuration
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=vaultforge password=vaultforge dbname=vaultforge port=5432 sslmode=disable"
	}

	// Initialize database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Migrate schema
	err = db.AutoMigrate(
		&core.Tenant{},
		&core.Wallet{},
		&core.Intent{},
		&core.Transaction{},
		&core.AuditEvent{},
		&core.Policy{},
		&core.MPCShare{},
	)
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// Initialize core services
	intentStore := core.NewPostgresIntentStore(db)
	walletStore := core.NewPostgresWalletStore(db)
	transactionStore := core.NewPostgresTransactionStore(db)
	auditStore := core.NewPostgresAuditStore(db)
	policyEngine := core.NewPolicyEngine(db)
	zkVerifier := core.NewZKVerifier()
	mpcSigner := core.NewMPCSigner(db)
	reconciler := core.NewReconciler(db)

	// Create HTTP router
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(api.AuthMiddleware())

	// Create handler and register routes
	handler := api.NewIntentHandler(
		intentStore,
		&policyEngine,
		zkVerifier,
		mpcSigner,
		reconciler,
	)

	routes.RegisterRoutes(r, handler, walletStore, transactionStore)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &gin.Server{
		Addr: ":" + port,
	}

	go func() {
		log.Printf("VaultForge API starting on %s", port)
		if err := srv.ListenAndServe(); err != nil && err != gin.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	// ... (signal handling)
}
package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
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
		&core.MPCShareRecord{},
		&core.ReplayKey{},
		&core.WebhookEndpoint{},
	)
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// Initialize stores
	intentStore := core.NewPostgresIntentStore(db)

	// Initialize core services
	dbAdapter := core.NewDBAdapter(db)
	policyEngine := core.NewPolicyEngine(dbAdapter)
	zkVerifier := core.NewZKVerifier()
	mpcSigner := core.NewMPCSigner(db)
	reconciler := core.NewReconciler(db)
	audit := core.NewAuditLogger(db)
	solanaClient := core.NewSolanaClient(os.Getenv("SOLANA_RPC_URL"))
	webhooks := core.NewWebhookNotifier(db)
	txStore := core.NewPostgresTransactionStore(db)

	// Create HTTP router
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(routes.AuthMiddleware())

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

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("VaultForge API starting on %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

package core

import (
	"log"
	"time"

	"gorm.io/gorm"
)

// ReconcilerImpl implements Reconciler.
// Compares internal DB state against Solana chain state.
type ReconcilerImpl struct {
	db *gorm.DB
}

func NewReconciler(db *gorm.DB) *ReconcilerImpl {
	return &ReconcilerImpl{db: db}
}

// Start begins reconciliation for a given intent.
// In production, this would poll Solana RPC for transaction confirmation
// and update the internal state accordingly.
func (r *ReconcilerImpl) Start(intentID string) {
	log.Printf("Starting reconciliation for intent=%s", intentID)

	// Look up the intent
	var intent Intent
	if err := r.db.Where("id = ?", intentID).First(&intent).Error; err != nil {
		log.Printf("Reconciliation failed: intent %s not found: %v", intentID, err)
		return
	}

	// Only reconcile intents in submitted state
	if intent.Status != "submitted" {
		log.Printf("Reconciliation skipped for intent=%s: status=%s", intentID, intent.Status)
		return
	}

	// Simulate chain confirmation with a delay
	// In production: poll Solana RPC getSignatureStatuses
	time.Sleep(5 * time.Second)

	// Update intent to confirmed
	now := time.Now().UTC()
	intent.Status = "confirmed"
	intent.ConfirmedAt = &now
	if err := r.db.Save(&intent).Error; err != nil {
		log.Printf("Reconciliation failed to update intent=%s: %v", intentID, err)
		return
	}

	// Create audit event
	auditEvent := AuditEvent{
		ID:        generateUUID(),
		TenantID:  intent.TenantID,
		Actor:     "system",
		Action:    "confirmed",
		Resource:  "intent",
		IntentID:  intentID,
		RequestID: generateUUID(),
		Result:    "success",
		CreatedAt: now,
	}
	r.db.Create(&auditEvent)

	log.Printf("Reconciliation complete for intent=%s: confirmed", intentID)
}

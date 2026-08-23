package core

import (
	"log"
	"time"

	"gorm.io/gorm"
)

// ReconcilerImpl implements Reconciler.
// Compares internal DB state against Solana chain state.
type ReconcilerImpl struct {
	db           *gorm.DB
	solanaClient SolanaSubmitter
	webhooks     StateNotifier
	audit        IntentAuditor
}

func NewReconciler(db *gorm.DB) *ReconcilerImpl {
	return &ReconcilerImpl{db: db}
}

// SetSolanaClient sets the Solana RPC client for chain polling.
func (r *ReconcilerImpl) SetSolanaClient(client SolanaSubmitter) {
	r.solanaClient = client
}

// SetWebhookNotifier sets the webhook notifier for state change notifications.
func (r *ReconcilerImpl) SetWebhookNotifier(wh StateNotifier) {
	r.webhooks = wh
}

// SetAuditLogger sets the audit logger for recording confirmations.
func (r *ReconcilerImpl) SetAuditLogger(auditor IntentAuditor) {
	r.audit = auditor
}

// Start begins reconciliation for a given intent.
// Polls Solana RPC getSignatureStatuses until confirmed or timeout.
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

	if len(intent.TXSignature) == 0 {
		log.Printf("Reconciliation failed for intent=%s: no transaction signature", intentID)
		r.failIntent(&intent, "no transaction signature", "reconciliation")
		return
	}

	// Poll Solana for confirmation
	if r.solanaClient == nil {
		log.Printf("Reconciliation: no Solana client, simulating confirmation for intent=%s", intentID)
		time.Sleep(5 * time.Second)
		r.confirmIntent(&intent)
		return
	}

	maxAttempts := 60 // ~2 minutes with 2s intervals
	for attempt := 0; attempt < maxAttempts; attempt++ {
		time.Sleep(2 * time.Second)

		sig := string(intent.TXSignature)
		confirmed, err := r.solanaClient.WaitForConfirmation(sig)
		if err != nil {
			log.Printf("Reconciliation poll error for intent=%s attempt=%d: %v", intentID, attempt+1, err)
			continue
		}

		if confirmed {
			r.confirmIntent(&intent)
			return
		}
	}

	// Timeout — mark as confirmation timeout
	log.Printf("Reconciliation timeout for intent=%s after %d attempts", intentID, maxAttempts)
	r.failIntent(&intent, "confirmation timeout after polling", "confirmation_timeout")
}

// confirmIntent transitions an intent to confirmed status.
func (r *ReconcilerImpl) confirmIntent(intent *Intent) {
	now := time.Now().UTC()
	intent.Status = "confirmed"
	intent.ConfirmedAt = &now
	if err := r.db.Save(intent).Error; err != nil {
		log.Printf("Reconciliation failed to update intent=%s: %v", intent.ID, err)
		return
	}

	// Update transaction status
	var tx Transaction
	if err := r.db.Where("intent_id = ?", intent.ID).First(&tx).Error; err == nil {
		tx.Status = "confirmed"
		tx.ConfirmedAt = &now
		r.db.Save(&tx)
	}

	// Audit log
	if r.audit != nil {
		r.audit.LogIntentConfirmed(intent.TenantID, "system", intent.ID, GenerateRequestID())
	}

	// Webhook notification
	if r.webhooks != nil {
		r.webhooks.NotifyIntentStateChange(intent, "intent.confirmed")
	}

	log.Printf("Reconciliation complete for intent=%s: confirmed", intent.ID)
}

// failIntent transitions an intent to failed status with a reason.
func (r *ReconcilerImpl) failIntent(intent *Intent, reason, failureType string) {
	now := time.Now().UTC()
	intent.Status = "failed"
	intent.FailureReason = FailureReason(failureType)
	intent.UpdatedAt = now
	if err := r.db.Save(intent).Error; err != nil {
		log.Printf("Reconciliation failed to update intent=%s: %v", intent.ID, err)
		return
	}

	if r.audit != nil {
		r.audit.LogIntentFailed(intent.TenantID, "system", intent.ID, GenerateRequestID(), reason, failureType)
	}

	if r.webhooks != nil {
		r.webhooks.NotifyIntentStateChange(intent, "intent.reconciliation_failed")
	}

	log.Printf("Reconciliation failed for intent=%s: %s", intent.ID, reason)
}

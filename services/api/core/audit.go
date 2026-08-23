package core

import (
	"encoding/json"
	"log"
	"time"

	"gorm.io/gorm"
)

// AuditLogger creates audit events for every state transition.
type AuditLogger struct {
	db *gorm.DB
}

func NewAuditLogger(db *gorm.DB) *AuditLogger {
	return &AuditLogger{db: db}
}

// Log records an audit event.
func (a *AuditLogger) Log(tenantID, actor, action, resource, intentID, requestID, result string, metadata interface{}) {
	var metaJSON json.RawMessage
	if metadata != nil {
		metaJSON, _ = json.Marshal(metadata)
	}

	event := AuditEvent{
		ID:        GenerateUUID(),
		TenantID:  tenantID,
		Actor:     actor,
		Action:    action,
		Resource:  resource,
		IntentID:  intentID,
		RequestID: requestID,
		Result:    result,
		Metadata:  metaJSON,
		CreatedAt: time.Now().UTC(),
	}

	if err := a.db.Create(&event).Error; err != nil {
		log.Printf("AUDIT LOG FAILED: tenant=%s action=%s intent=%s err=%v", tenantID, action, intentID, err)
	}
}

// LogIntentCreated records intent creation.
func (a *AuditLogger) LogIntentCreated(tenantID, actor, intentID, requestID string) {
	a.Log(tenantID, actor, "created", "intent", intentID, requestID, "success", nil)
}

// LogIntentSubmitted records intent submission to pending.
func (a *AuditLogger) LogIntentSubmitted(tenantID, actor, intentID, requestID string) {
	a.Log(tenantID, actor, "submitted", "intent", intentID, requestID, "success", nil)
}

// LogIntentApproved records intent approval.
func (a *AuditLogger) LogIntentApproved(tenantID, actor, intentID, requestID string, policyResult *PolicyResult) {
	a.Log(tenantID, actor, "approved", "intent", intentID, requestID, "success", policyResult)
}

// LogIntentPolicyDenied records policy denial.
func (a *AuditLogger) LogIntentPolicyDenied(tenantID, actor, intentID, requestID, reason string) {
	a.Log(tenantID, actor, "policy_denied", "intent", intentID, requestID, "denied", map[string]string{"reason": reason})
}

// LogIntentZKDenied records ZK verification failure.
func (a *AuditLogger) LogIntentZKDenied(tenantID, actor, intentID, requestID string) {
	a.Log(tenantID, actor, "zk_denied", "intent", intentID, requestID, "denied", nil)
}

// LogIntentExecuted records intent execution start.
func (a *AuditLogger) LogIntentExecuted(tenantID, actor, intentID, requestID string) {
	a.Log(tenantID, actor, "executing", "intent", intentID, requestID, "success", nil)
}

// LogIntentSimulated records transaction simulation result.
func (a *AuditLogger) LogIntentSimulated(tenantID, actor, intentID, requestID string, success bool) {
	result := "success"
	if !success {
		result = "failed"
	}
	a.Log(tenantID, actor, "simulated", "intent", intentID, requestID, result, nil)
}

// LogIntentSigned records MPC signing result.
func (a *AuditLogger) LogIntentSigned(tenantID, actor, intentID, requestID string, participants []uint32) {
	a.Log(tenantID, actor, "signed", "intent", intentID, requestID, "success", map[string]interface{}{
		"participants": participants,
	})
}

// LogIntentSubmittedOnChain records submission to Solana.
func (a *AuditLogger) LogIntentSubmittedOnChain(tenantID, actor, intentID, requestID, txSignature string) {
	a.Log(tenantID, actor, "submitted_on_chain", "intent", intentID, requestID, "success", map[string]string{
		"tx_signature": txSignature,
	})
}

// LogIntentConfirmed records chain confirmation.
func (a *AuditLogger) LogIntentConfirmed(tenantID, actor, intentID, requestID string) {
	a.Log(tenantID, actor, "confirmed", "intent", intentID, requestID, "success", nil)
}

// LogIntentFailed records intent failure.
func (a *AuditLogger) LogIntentFailed(tenantID, actor, intentID, requestID, reason, failureType string) {
	a.Log(tenantID, actor, "failed", "intent", intentID, requestID, "failure", map[string]string{
		"failure_type": failureType,
		"reason":       reason,
	})
}

// LogIntentExpired records intent expiration.
func (a *AuditLogger) LogIntentExpired(tenantID, actor, intentID, requestID string) {
	a.Log(tenantID, actor, "expired", "intent", intentID, requestID, "success", nil)
}

// LogIntentRejected records intent rejection.
func (a *AuditLogger) LogIntentRejected(tenantID, actor, intentID, requestID string) {
	a.Log(tenantID, actor, "rejected", "intent", intentID, requestID, "success", nil)
}

package core

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"gorm.io/gorm"
)

// WebhookEvent represents an event to send via webhook.
type WebhookEvent struct {
	EventType string          `json:"event_type"`
	IntentID  string          `json:"intent_id"`
	TenantID  string          `json:"tenant_id"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// WebhookNotifier sends webhook notifications on state changes.
type WebhookNotifier struct {
	db         *gorm.DB
	httpClient *http.Client
}

func NewWebhookNotifier(db *gorm.DB) *WebhookNotifier {
	return &WebhookNotifier{
		db: db,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// WebhookEndpoint represents a registered webhook endpoint.
type WebhookEndpoint struct {
	ID        string `json:"id" gorm:"primaryKey"`
	TenantID  string `json:"tenant_id" gorm:"index"`
	URL       string `json:"url"`
	EventType string `json:"event_type"`
	IsActive  bool   `json:"is_active" gorm:"default:true"`
	Secret    string `json:"secret"`
	CreatedAt time.Time `json:"created_at"`
}

func (WebhookEndpoint) TableName() string { return "webhook_endpoints" }

// Notify sends webhook notifications for an event to all matching endpoints.
func (w *WebhookNotifier) Notify(event WebhookEvent) {
	var endpoints []WebhookEndpoint
	if err := w.db.Where("tenant_id = ? AND is_active = ? AND (event_type = ? OR event_type = ?)",
		event.TenantID, true, event.EventType, "*").Find(&endpoints).Error; err != nil {
		log.Printf("Failed to load webhook endpoints: %v", err)
		return
	}

	for _, ep := range endpoints {
		go w.deliver(ep, event)
	}
}

// deliver sends a single webhook delivery with retry.
func (w *WebhookNotifier) deliver(endpoint WebhookEndpoint, event WebhookEvent) {
	body, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal webhook event: %v", err)
		return
	}

	maxRetries := 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second // 1s, 2s, 4s, 8s
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			time.Sleep(backoff)
		}

		req, err := http.NewRequest("POST", endpoint.URL, bytes.NewReader(body))
		if err != nil {
			log.Printf("Failed to create webhook request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-VaultForge-Event", event.EventType)
		req.Header.Set("X-VaultForge-Delivery", event.IntentID)

		resp, err := w.httpClient.Do(req)
		if err != nil {
			log.Printf("Webhook delivery failed (attempt %d/%d) endpoint=%s: %v", attempt+1, maxRetries, endpoint.URL, err)
			continue
		}

		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Printf("Webhook delivered: endpoint=%s event=%s (attempt %d)", endpoint.URL, event.EventType, attempt+1)
			return
		}

		log.Printf("Webhook rejected (attempt %d/%d) endpoint=%s status=%d", attempt+1, maxRetries, endpoint.URL, resp.StatusCode)
	}

	log.Printf("Webhook delivery failed after %d attempts: endpoint=%s event=%s", maxRetries, endpoint.URL, event.EventType)
}

// NotifyIntentStateChange sends a webhook for intent state transitions.
func (w *WebhookNotifier) NotifyIntentStateChange(intent *Intent, eventType string) {
	payload, _ := json.Marshal(map[string]interface{}{
		"intent_id": intent.ID,
		"status":    intent.Status,
		"wallet_id": intent.WalletID,
		"amount":    intent.Amount,
		"token":     intent.Token,
	})

	w.Notify(WebhookEvent{
		EventType: eventType,
		IntentID:  intent.ID,
		TenantID:  intent.TenantID,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	})
}

// NotifyTransactionConfirmed sends a webhook for transaction confirmation.
func (w *WebhookNotifier) NotifyTransactionConfirmed(tx *Transaction) {
	payload, _ := json.Marshal(map[string]interface{}{
		"transaction_id":      tx.ID,
		"intent_id":          tx.IntentID,
		"confirmation_signature": tx.ConfirmSignature,
	})

	w.Notify(WebhookEvent{
		EventType: "transaction.confirmed",
		IntentID:  tx.IntentID,
		TenantID:  tx.TenantID,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	})
}

// RegisterEndpoint registers a new webhook endpoint.
func (w *WebhookNotifier) RegisterEndpoint(tenantID, url, eventType, secret string) error {
	ep := WebhookEndpoint{
		ID:        GenerateUUID(),
		TenantID:  tenantID,
		URL:       url,
		EventType: eventType,
		IsActive:  true,
		Secret:    secret,
		CreatedAt: time.Now().UTC(),
	}
	return w.db.Create(&ep).Error
}

// ListEndpoints returns all active webhook endpoints for a tenant.
func (w *WebhookNotifier) ListEndpoints(tenantID string) ([]WebhookEndpoint, error) {
	var endpoints []WebhookEndpoint
	err := w.db.Where("tenant_id = ? AND is_active = ?", tenantID, true).Find(&endpoints).Error
	return endpoints, err
}

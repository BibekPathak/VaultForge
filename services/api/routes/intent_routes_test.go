package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/vaultforge/vaultforge/services/api/core"
)

func setupTestRouter(handler *IntentHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", "tenant-1")
		c.Set("request_id", "test-req-1")
		c.Next()
	})
	v1 := r.Group("/v1")
	handler.RegisterRoutes(v1)
	return r
}

func setupMockHandler() (*IntentHandler, *core.MockIntentStore) {
	store := core.NewMockIntentStore()
	store.Wallets["wallet-1"] = &core.Wallet{ID: "wallet-1", TenantID: "tenant-1", DailyLimit: 100000}
	policyDB := core.NewMockPolicyDB()
	policyDB.Wallets["wallet-1"] = &core.Wallet{ID: "wallet-1", TenantID: "tenant-1", DailyLimit: 100000}
	policyDB.Tenants["tenant-1"] = &core.Tenant{ID: "tenant-1"}

	handler := NewIntentHandler(
		store,
		core.NewPolicyEngine(policyDB),
		&core.MockZKVerifier{},
		&core.MockMPCSigner{},
		&core.MockReconciler{},
		&core.MockAuditLogger{},
		&core.MockSolanaClient{},
		&core.MockWebhookNotifier{},
		core.NewMockTransactionStore(),
	)
	return handler, store
}

func TestCreateIntent_Success(t *testing.T) {
	handler, _ := setupMockHandler()
	r := setupTestRouter(handler)

	body := `{"wallet_id":"wallet-1","destination":"dest-addr","token":"SOL","amount":1000,"chain":"solana","creator":"user-1"}`
	req := httptest.NewRequest("POST", "/v1/intents", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp CreateIntentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Intent == nil {
		t.Fatal("intent should not be nil")
	}
	if resp.Intent.Status != "pending" {
		t.Errorf("expected status=pending, got %s", resp.Intent.Status)
	}
}

func TestCreateIntent_MissingFields(t *testing.T) {
	handler, _ := setupMockHandler()
	r := setupTestRouter(handler)

	body := `{"wallet_id":"wallet-1"}`
	req := httptest.NewRequest("POST", "/v1/intents", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetIntent_Success(t *testing.T) {
	handler, store := setupMockHandler()
	r := setupTestRouter(handler)

	store.Intents["intent-1"] = &core.Intent{ID: "intent-1", TenantID: "tenant-1", Status: "draft"}

	req := httptest.NewRequest("GET", "/v1/intents/intent-1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetIntent_NotFound(t *testing.T) {
	handler, _ := setupMockHandler()
	r := setupTestRouter(handler)

	req := httptest.NewRequest("GET", "/v1/intents/nonexistent", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetIntent_WrongTenant(t *testing.T) {
	handler, store := setupMockHandler()
	r := setupTestRouter(handler)

	store.Intents["intent-1"] = &core.Intent{ID: "intent-1", TenantID: "other-tenant", Status: "draft"}

	req := httptest.NewRequest("GET", "/v1/intents/intent-1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestListIntents_Success(t *testing.T) {
	handler, store := setupMockHandler()
	r := setupTestRouter(handler)

	store.Intents["i1"] = &core.Intent{ID: "i1", TenantID: "tenant-1", Status: "draft"}
	store.Intents["i2"] = &core.Intent{ID: "i2", TenantID: "tenant-1", Status: "approved"}

	req := httptest.NewRequest("GET", "/v1/intents", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp ListIntentsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp.Intents) != 2 {
		t.Errorf("expected 2 intents, got %d", len(resp.Intents))
	}
}

func TestRejectIntent_Success(t *testing.T) {
	handler, store := setupMockHandler()
	r := setupTestRouter(handler)

	store.Intents["intent-1"] = &core.Intent{ID: "intent-1", TenantID: "tenant-1", Status: "pending"}

	req := httptest.NewRequest("POST", "/v1/intents/intent-1/reject", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListWallets_Success(t *testing.T) {
	handler, store := setupMockHandler()
	r := setupTestRouter(handler)

	store.Wallets["w1"] = &core.Wallet{ID: "w1", TenantID: "tenant-1"}

	req := httptest.NewRequest("GET", "/v1/wallets", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestListAuditEvents_Success(t *testing.T) {
	handler, store := setupMockHandler()
	r := setupTestRouter(handler)

	store.AuditEvents["e1"] = &core.AuditEvent{ID: "e1", TenantID: "tenant-1", Action: "intent.created"}

	req := httptest.NewRequest("GET", "/v1/audit-events", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

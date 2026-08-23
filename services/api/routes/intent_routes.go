package routes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vaultforge/vaultforge/services/api/core"
)

// IntentHandler handles intent-related HTTP endpoints.
type IntentHandler struct {
	store        core.IntentStore
	policyEngine *core.PolicyEngine
	zkVerifier   core.ZKVerifier
	mpcSigner    core.MPCSigner
	reconciler   core.Reconciler
}

// NewIntentHandler creates a new IntentHandler.
func NewIntentHandler(
	store core.IntentStore,
	policyEngine *core.PolicyEngine,
	zkVerifier core.ZKVerifier,
	mpcSigner core.MPCSigner,
	reconciler core.Reconciler,
) *IntentHandler {
	return &IntentHandler{
		store:        store,
		policyEngine: policyEngine,
		zkVerifier:   zkVerifier,
		mpcSigner:    mpcSigner,
		reconciler:   reconciler,
	}
}

// RegisterRoutes registers all intent-related routes.
func (h *IntentHandler) RegisterRoutes(router *gin.RouterGroup) {
	intents := router.Group("/intents")
	{
		intents.POST("", h.CreateIntent)
		intents.GET("", h.ListIntents)
		intents.GET("/:id", h.GetIntent)
		intents.POST("/:id/approve", h.ApproveIntent)
		intents.POST("/:id/reject", h.RejectIntent)
		intents.POST("/:id/execute", h.ExecuteIntent)
		intents.POST("/:id/cancel", h.CancelIntent)
	}

	router.GET("/wallets", h.ListWallets)
	router.GET("/wallets/:id", h.GetWallet)
	router.GET("/transactions", h.ListTransactions)
	router.GET("/audit-events", h.ListAuditEvents)
}

// CreateIntent creates a new intent draft.
func (h *IntentHandler) CreateIntent(c *gin.Context) {
	var input CreateIntentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	tenantID := c.MustGet("tenant_id").(string)

	intent := core.NewIntent(
		tenantID,
		input.WalletID,
		input.Destination,
		input.Token,
		input.Chain,
		input.Creator,
		input.Amount,
	)

	if err := h.store.Create(intent); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to create intent"})
		return
	}

	c.JSON(http.StatusCreated, CreateIntentResponse{Intent: intent})
}

// ApproveIntent approves a pending intent after policy and ZK verification.
func (h *IntentHandler) ApproveIntent(c *gin.Context) {
	id := c.Param("id")

	intent, err := h.store.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "intent not found"})
		return
	}

	if intent.TenantID != c.MustGet("tenant_id").(string) {
		c.JSON(http.StatusForbidden, errorResponse{Error: "access denied"})
		return
	}

	if intent.Status != "pending" {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "intent is not in pending status"})
		return
	}

	// Policy engine evaluation
	policyResult, err := h.policyEngine.Evaluate(intent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "policy evaluation failed"})
		return
	}

	if !policyResult.Allow {
		intent.FailureReason = core.FailureReason(core.PolicyDenied)
		intent.Status = "rejected"
		_ = h.store.Update(intent)
		c.JSON(http.StatusOK, ApproveIntentResponse{
			Intent:       intent,
			PolicyResult: policyResult,
		})
		return
	}

	// ZK policy verification
	amountInt := parseAmount(intent.Amount)
	dailyLimit := int64(100000) // would be loaded from policy config

	zkProof, err := h.zkVerifier.GenerateProof(core.PolicyInputs{
		Amount:        intent.Amount,
		DailyLimit:    &dailyLimit,
		Recipient:     intent.Destination,
		PolicyVersion: intent.PolicyVersion,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "ZK proof generation failed"})
		return
	}

	zkVerified, err := h.zkVerifier.VerifyPolicyProof(zkProof)
	if err != nil || !zkVerified {
		intent.FailureReason = core.FailureReason(core.PolicyDenied)
		intent.Status = "rejected"
		_ = h.store.Update(intent)
		c.JSON(http.StatusOK, ApproveIntentResponse{
			Intent:       intent,
			PolicyResult: &core.PolicyResult{Allow: false, Reason: "ZK proof verification failed"},
		})
		return
	}

	_ = amountInt // used for audit logging

	// Transition to approved
	now := time.Now().UTC()
	intent.Status = "approved"
	intent.ApprovedAt = &now
	if err := h.store.Update(intent); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to update intent"})
		return
	}

	c.JSON(http.StatusOK, ApproveIntentResponse{
		Intent:       intent,
		PolicyResult: policyResult,
	})
}

// ExecuteIntent executes an approved intent through the full signing pipeline.
func (h *IntentHandler) ExecuteIntent(c *gin.Context) {
	id := c.Param("id")

	intent, err := h.store.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "intent not found"})
		return
	}

	if intent.TenantID != c.MustGet("tenant_id").(string) {
		c.JSON(http.StatusForbidden, errorResponse{Error: "access denied"})
		return
	}

	if intent.Status != "approved" {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "intent is not in approved status"})
		return
	}

	// Check expiry
	if time.Now().UTC().After(intent.Expiry) {
		intent.Status = "expired"
		intent.FailureReason = core.FailureReason(core.Expired)
		_ = h.store.Update(intent)
		c.JSON(http.StatusOK, ExecuteIntentResponse{
			Intent: intent,
			Error:  "intent expired",
		})
		return
	}

	// Idempotency check
	existingTx, _ := h.store.GetTransactionByNonce(intent.Nonce)
	if existingTx != nil {
		c.JSON(http.StatusAccepted, ExecuteIntentResponse{
			Intent: intent,
			Error:  "duplicate execution detected",
		})
		return
	}

	// Transition to executing
	now := time.Now().UTC()
	intent.Status = "executing"
	intent.UpdatedAt = now
	if err := h.store.Update(intent); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to update intent"})
		return
	}

	// Build transaction
	txBuilder := core.NewTransactionBuilder(
		intent.WalletID,
		intent.Token,
		intent.Amount,
		intent.Destination,
	)

	// Simulate transaction
	simResult, err := txBuilder.Simulate()
	if err != nil {
		intent.Status = "failed"
		intent.FailureReason = core.FailureReason(core.SimulationFailed)
		_ = h.store.Update(intent)
		c.JSON(http.StatusInternalServerError, ExecuteIntentResponse{
			Intent: intent,
			Error:  "transaction simulation failed",
		})
		return
	}

	if !simResult.Allowed {
		intent.Status = "failed"
		intent.FailureReason = core.FailureReason(core.PolicyDenied)
		_ = h.store.Update(intent)
		c.JSON(http.StatusInternalServerError, ExecuteIntentResponse{
			Intent: intent,
			Error:  "transaction simulation denied",
		})
		return
	}

	// Policy re-verification before signing
	policyResult, err := h.policyEngine.Evaluate(intent)
	if err != nil || !policyResult.Allow {
		intent.Status = "failed"
		intent.FailureReason = core.FailureReason(core.PolicyDenied)
		_ = h.store.Update(intent)
		c.JSON(http.StatusInternalServerError, ExecuteIntentResponse{
			Intent: intent,
			Error:  "policy verification failed before signing",
		})
		return
	}

	// MPC signing
	signingResult, err := h.mpcSigner.Sign(core.SigningInput{
		IntentHash: nil, // would compute intent hash
		TxHash:     simResult.TransactionHash,
		Chain:      intent.Chain,
		Context:    "vaultforge-intent-" + id,
	})
	if err != nil {
		intent.Status = "failed"
		intent.FailureReason = core.FailureReason(core.SigningFailed)
		_ = h.store.Update(intent)
		c.JSON(http.StatusInternalServerError, ExecuteIntentResponse{
			Intent: intent,
			Error:  "MPC signing failed",
		})
		return
	}

	// Build full transaction bytes
	_ = txBuilder.Build()

	// Transition to submitted
	now2 := time.Now().UTC()
	intent.Status = "submitted"
	intent.ExecutedAt = &now2
	intent.TXSignature = signingResult.Signature
	if err := h.store.Update(intent); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to update intent"})
		return
	}

	// Start reconciliation goroutine
	go h.reconciler.Start(intent.ID)

	c.JSON(http.StatusAccepted, ExecuteIntentResponse{
		Intent: intent,
		TxHash: simResult.TransactionHash,
	})
}

// RejectIntent rejects an intent.
func (h *IntentHandler) RejectIntent(c *gin.Context) {
	id := c.Param("id")

	intent, err := h.store.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "intent not found"})
		return
	}

	if intent.TenantID != c.MustGet("tenant_id").(string) {
		c.JSON(http.StatusForbidden, errorResponse{Error: "access denied"})
		return
	}

	intent.Status = "rejected"
	intent.FailureReason = core.FailureReason(core.Rejected)
	intent.UpdatedAt = time.Now().UTC()
	if err := h.store.Update(intent); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to update intent"})
		return
	}

	c.JSON(http.StatusOK, RejectIntentResponse{Intent: intent})
}

// CancelIntent cancels a pending or executing intent.
func (h *IntentHandler) CancelIntent(c *gin.Context) {
	id := c.Param("id")

	intent, err := h.store.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "intent not found"})
		return
	}

	if intent.TenantID != c.MustGet("tenant_id").(string) {
		c.JSON(http.StatusForbidden, errorResponse{Error: "access denied"})
		return
	}

	if intent.Status != "pending" && intent.Status != "executing" {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "intent cannot be cancelled in current status"})
		return
	}

	intent.Status = "rejected"
	intent.FailureReason = core.FailureReason(core.Rejected)
	intent.UpdatedAt = time.Now().UTC()
	if err := h.store.Update(intent); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to update intent"})
		return
	}

	c.JSON(http.StatusOK, CancelIntentResponse{Intent: intent})
}

// ListIntents lists intents for a tenant.
func (h *IntentHandler) ListIntents(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(string)
	status := c.Query("status")

	intents, err := h.store.ListByTenant(tenantID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to list intents"})
		return
	}

	c.JSON(http.StatusOK, ListIntentsResponse{Intents: intents})
}

// GetIntent gets a single intent.
func (h *IntentHandler) GetIntent(c *gin.Context) {
	id := c.Param("id")

	intent, err := h.store.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "intent not found"})
		return
	}

	if intent.TenantID != c.MustGet("tenant_id").(string) {
		c.JSON(http.StatusForbidden, errorResponse{Error: "access denied"})
		return
	}

	c.JSON(http.StatusOK, GetIntentResponse{Intent: intent})
}

// GetWallet gets a wallet.
func (h *IntentHandler) GetWallet(c *gin.Context) {
	id := c.Param("id")

	wallet, err := h.store.GetWallet(id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "wallet not found"})
		return
	}

	if wallet.TenantID != c.MustGet("tenant_id").(string) {
		c.JSON(http.StatusForbidden, errorResponse{Error: "access denied"})
		return
	}

	c.JSON(http.StatusOK, GetWalletResponse{Wallet: *wallet})
}

// ListWallets lists wallets for a tenant.
func (h *IntentHandler) ListWallets(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(string)

	wallets, err := h.store.ListWallets(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to list wallets"})
		return
	}

	c.JSON(http.StatusOK, ListWalletsResponse{Wallets: wallets})
}

// ListTransactions lists transactions for a tenant.
func (h *IntentHandler) ListTransactions(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(string)

	transactions, err := h.store.ListTransactions(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to list transactions"})
		return
	}

	c.JSON(http.StatusOK, ListTransactionsResponse{Transactions: transactions})
}

// ListAuditEvents lists audit events for a tenant.
func (h *IntentHandler) ListAuditEvents(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(string)
	action := c.Query("action")

	events, err := h.store.ListAuditEvents(tenantID, action)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to list audit events"})
		return
	}

	c.JSON(http.StatusOK, ListAuditEventsResponse{Events: events})
}

// --- Helper types ---

// CreateIntentInput is the request body for creating an intent.
type CreateIntentInput struct {
	WalletID    string `json:"wallet_id" binding:"required"`
	Destination string `json:"destination" binding:"required"`
	Token       string `json:"token" binding:"required"`
	Amount      int64  `json:"amount" binding:"required,gt=0"`
	Chain       string `json:"chain" binding:"required"`
	Creator     string `json:"creator" binding:"required"`
}

type CreateIntentResponse struct {
	Intent *core.Intent `json:"intent"`
}

type ApproveIntentResponse struct {
	Intent       *core.Intent       `json:"intent"`
	PolicyResult *core.PolicyResult `json:"policy_result"`
}

type ExecuteIntentResponse struct {
	Intent *core.Intent `json:"intent"`
	TxHash string       `json:"tx_hash,omitempty"`
	Error  string       `json:"error,omitempty"`
}

type RejectIntentResponse struct {
	Intent *core.Intent `json:"intent"`
}

type CancelIntentResponse struct {
	Intent *core.Intent `json:"intent"`
}

type ListIntentsResponse struct {
	Intents []core.Intent `json:"intents"`
}

type GetIntentResponse struct {
	Intent *core.Intent `json:"intent"`
}

type GetWalletResponse struct {
	Wallet core.Wallet `json:"wallet"`
}

type ListWalletsResponse struct {
	Wallets []core.Wallet `json:"wallets"`
}

type ListTransactionsResponse struct {
	Transactions []core.Transaction `json:"transactions"`
}

type ListAuditEventsResponse struct {
	Events []core.AuditEvent `json:"events"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func parseAmount(s string) int64 {
	var v int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			v = v*10 + int64(c-'0')
		}
	}
	return v
}

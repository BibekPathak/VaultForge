package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vaultforge/vaultforge/services/api/core"
)

// IntentHandler handles intent-related HTTP endpoints
type IntentHandler struct {
	store       core.IntentStore
	policyEngine *core.PolicyEngine
	zkVerifier  core.ZKVerifier
	mpcSigner   core.MPCSigner
	reconciler  core.Reconciler
}

// NewIntentHandler creates a new IntentHandler
func NewIntentHandler(
	store core.IntentStore,
	policyEngine *core.PolicyEngine,
	zkVerifier core.ZKVerifier,
	mpcSigner core.MPCSigner,
	reconciler core.Reconciler,
) *IntentHandler {
	return &IntentHandler{
		store:       store,
		policyEngine: policyEngine,
		zkVerifier:  zkVerifier,
		mpcSigner:   mpcSigner,
		reconciler:  reconciler,
	}
}

// RegisterRoutes registers the intent routes
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

// CreateIntent creates a new intent
// @Summary Create a new intent
// @Description Creates a new intent draft for policy evaluation and approval
// @Accept json
// @Produce json
// @Param input body CreateIntentInput true "Intent creation input"
// @Success 201 {object} CreateIntentResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Router /v1/intents [post]
func (h *IntentHandler) CreateIntent(c *gin.Context) {
	var input CreateIntentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	// Validate tenant from authenticated context (not from input)
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

	// Store the intent
	if err := h.store.Create(intent); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to create intent"})
		return
	}

	c.JSON(http.StatusCreated, CreateIntentResponse{
		Intent: intent,
	})
}

// ApproveIntent approves a pending intent
// @Summary Approve an intent
// @Description Approves a pending intent, triggering policy evaluation
// @Accept json
// @Produce json
// @Param id path string true "Intent ID"
// @Success 200 {object} ApproveIntentResponse
// @Failure 400 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /v1/intents/{id}/approve [post]
func (h *IntentHandler) ApproveIntent(c *gin.Context) {
	id := c.Param("id")
	
	intent, err := h.store.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "intent not found"})
		return
	}

	// Check tenant isolation
	if intent.TenantID != c.MustGet("tenant_id").(string) {
		c.JSON(http.StatusForbidden, errorResponse{Error: "access denied to different tenant"})
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
		if err := h.store.Update(intent); err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to update intent"})
			return
		}
		c.JSON(http.StatusOK, ApproveIntentResponse{
			Intent:     intent,
			PolicyResult: policyResult,
		})
		return
	}

	// ZK policy verification
	zkProof, err := h.zkVerifier.GenerateProof(core.PolicyInputs{
		Amount:        string(intent.Amount),
		DailyLimit:    nil, // would be loaded from policy config
		Recipient:     intent.Destination,
		PolicyVersion: string(intent.PolicyVersion),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "ZK proof generation failed"})
		return
	}

	zkVerified, err := h.zkVerifier.VerifyPolicyProof(zkProof)
	if err != nil || !zkVerified {
		intent.FailureReason = core.FailureReason(core.PolicyDenied)
		intent.Status = "rejected"
		if err := h.store.Update(intent); err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to update intent"})
			return
		}
		c.JSON(http.StatusOK, ApproveIntentResponse{
			Intent:     intent,
			PolicyResult: PolicyResult{Allow: false, Reason: "ZK proof verification failed"},
		})
		return
	}

	// Transition to approved
	intent.Status = "approved"
	intent.ApprovedAt = &time.Now()
	if err := h.store.Update(intent); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to update intent"})
		return
	}

	c.JSON(http.StatusOK, ApproveIntentResponse{
		Intent:     intent,
		PolicyResult: policyResult,
	})
}

// ExecuteIntent executes an approved intent
// @Summary Execute an intent
// @Description Executes an approved intent through the full signing pipeline
// @Accept json
// @Produce json
// @Param id path string true "Intent ID"
// @Success 202 {object} ExecuteIntentResponse
// @Failure 400 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /v1/intents/{id}/execute [post]
func (h *IntentHandler) ExecuteIntent(c *gin.Context) {
	id := c.Param("id")
	
	intent, err := h.store.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "intent not found"})
		return
	}

	// Check tenant isolation
	if intent.TenantID != c.MustGet("tenant_id").(string) {
		c.JSON(http.StatusForbidden, errorResponse{Error: "access denied to different tenant"})
		return
	}

	// Check status
	if intent.Status != "approved" {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "intent is not in approved status"})
		return
	}

	// Check expiry
	if time.Now().UTC().After(intent.Expiry) {
		intent.Status = "expired"
		intent.FailureReason = core.FailureReason(core.Expired)
		if err := h.store.Update(intent); err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to update intent"})
			return
		}
		c.JSON(http.StatusOK, ExecuteIntentResponse{
			Intent: intent,
			Error: "intent expired",
		})
		return
	}

	// Idempotency check
	existingTx, _ := h.store.GetTransactionByNonce(string(intent.Nonce))
	if existingTx != nil {
		c.JSON(http.StatusAccepted, ExecuteIntentResponse{
			Intent: intent,
			Error: "duplicate execution detected",
		})
		return
	}

	// Transition to executing
	intent.Status = "executing"
	intent.UpdatedAt = time.Now()
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
		h.store.Update(intent)
		c.JSON(http.StatusInternalServerError, ExecuteIntentResponse{
			Intent: intent,
			Error: "transaction simulation failed",
		})
		return
	}

	if !simResult.Allowed {
		intent.Status = "failed"
		intent.FailureReason = core.FailureReason(core.PolicyDenied)
		h.store.Update(intent)
		c.JSON(http.StatusInternalServerError, ExecuteIntentResponse{
			Intent: intent,
			Error: "transaction simulation denied",
		})
		return
	}

	// Policy re-verification before signing (already done during approval)
	// But we re-check to be sure
	policyResult, err := h.policyEngine.Evaluate(intent)
	if err != nil || !policyResult.Allow {
		intent.Status = "failed"
		intent.FailureReason = core.FailureReason(core.PolicyDenied)
		h.store.Update(intent)
		c.JSON(http.StatusInternalServerError, ExecuteIntentResponse{
			Intent: intent,
			Error: "policy verification failed before signing",
		})
		return
	}

	// MPC signing
	signingResult, err := h.mpcSigner.Sign(
		core.SigningInput{
			IntentHash:  nil, // would compute intent hash
			TxHash:      simResult.TransactionHash,
			Chain:       intent.Chain,
			Context:     "vaultforge-intent-" + id,
		},
	)
	if err != nil {
		intent.Status = "failed"
		intent.FailureReason = core.FailureReason(core.SigningFailed)
		h.store.Update(intent)
		c.JSON(http.StatusInternalServerError, ExecuteIntentResponse{
			Intent: intent,
			Error: "MPC signing failed",
		})
		return
	}

	// Build full transaction bytes
	txBytes := txBuilder.Build()
	
	// Submit to Solana
	// ... (RPC submission)

	// Transition to submitted
	intent.Status = "submitted"
	intent.ExecutedAt = &time.Now()
	intent.TransactionSignature = signingResult.Signature
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

// RejectIntent rejects an intent
// @Summary Reject an intent
// @Description Rejects an intent, moving it to failed/rejected state
// @Accept json
// @Produce json
// @Param id path string true "Intent ID"
// @Success 200 {object} RejectIntentResponse
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /v1/intents/{id}/reject [post]
func (h *IntentHandler) RejectIntent(c *gin.Context) {
	id := c.Param("id")
	
	intent, err := h.store.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "intent not found"})
		return
	}

	if intent.TenantID != c.MustGet("tenant_id").(string) {
		c.JSON(http.StatusForbidden, errorResponse{Error: "access denied to different tenant"})
		return
	}

	intent.Status = "rejected"
	intent.FailureReason = core.FailureReason(core.Rejected)
	intent.UpdatedAt = time.Now()
	if err := h.store.Update(intent); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to update intent"})
		return
	}

	c.JSON(http.StatusOK, RejectIntentResponse{Intent: intent})
}

// CancelIntent cancels an intent
// @Summary Cancel an intent
// @Description Cancels a pending or executing intent
// @Accept json
// @Produce json
// @Param id path string true "Intent ID"
// @Success 200 {object} CancelIntentResponse
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /v1/intents/{id}/cancel [post]
func (h *IntentHandler) CancelIntent(c *gin.Context) {
	id := c.Param("id")
	
	intent, err := h.store.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "intent not found"})
		return
	}

	if intent.TenantID != c.MustGet("tenant_id").(string) {
		c.JSON(http.StatusForbidden, errorResponse{Error: "access denied to different tenant"})
		return
	}

	// Only pending and executing intents can be cancelled
	if intent.Status != "pending" && intent.Status != "executing" {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "intent cannot be cancelled in current status"})
		return
	}

	intent.Status = "rejected"
	intent.FailureReason = core.FailureReason(core.Rejected)
	intent.UpdatedAt = time.Now()
	if err := h.store.Update(intent); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to update intent"})
		return
	}

	c.JSON(http.StatusOK, CancelIntentResponse{Intent: intent})
}

// ListIntents lists intents for a tenant
// @Summary List intents
// @Description Lists intents for the authenticated tenant
// @Produce json
// @Param status query string false "Filter by status"
// @Success 200 {object} ListIntentsResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Router /v1/intents [get]
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

// GetIntent gets a single intent
// @Summary Get an intent
// @Description Gets an intent by ID
// @Produce json
// @Param id path string true "Intent ID"
// @Success 200 {object} GetIntentResponse
// @Failure 404 {object} errorResponse
// @Router /v1/intents/:id [get]
func (h *IntentHandler) GetIntent(c *gin.Context) {
	id := c.Param("id")
	
	intent, err := h.store.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "intent not found"})
		return
	}

	if intent.TenantID != c.MustGet("tenant_id").(string) {
		c.JSON(http.StatusForbidden, errorResponse{Error: "access denied to different tenant"})
		return
	}

	c.JSON(http.StatusOK, GetIntentResponse{Intent: intent})
}

// GetWallet gets a wallet
// @Summary Get a wallet
// @Description Gets a wallet by ID
// @Produce json
// @Param id path string true "Wallet ID"
// @Success 200 {object} GetWalletResponse
// @Failure 404 {object} errorResponse
// @Router /v1/wallets/:id [get]
func (h *IntentHandler) GetWallet(c *gin.Context) {
	id := c.Param("id")
	
	wallet, err := h.store.GetWallet(id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "wallet not found"})
		return
	}

	if wallet.TenantID != c.MustGet("tenant_id").(string) {
		c.JSON(http.StatusForbidden, errorResponse{Error: "access denied to different tenant"})
		return
	}

	c.JSON(http.StatusOK, GetWalletResponse{Wallet: wallet})
}

// ListWallets lists wallets for a tenant
// @Summary List wallets
// @Description Lists wallets for the authenticated tenant
// @Produce json
// @Success 200 {object} ListWalletsResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Router /v1/wallets [get]
func (h *IntentHandler) ListWallets(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(string)
	
	wallets, err := h.store.ListWallets(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to list wallets"})
		return
	}

	c.JSON(http.StatusOK, ListWalletsResponse{Wallets: wallets})
}

// ListTransactions lists transactions
// @Summary List transactions
// @Description Lists transactions for a tenant
// @Produce json
// @Success 200 {object} ListTransactionsResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Router /v1/transactions [get]
func (h *IntentHandler) ListTransactions(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(string)
	
	transactions, err := h.store.ListTransactions(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to list transactions"})
		return
	}

	c.JSON(http.StatusOK, ListTransactionsResponse{Transactions: transactions})
}

// ListAuditEvents lists audit events
// @Summary List audit events
// @Description Lists audit events for a tenant
// @Produce json
// @Param action query string false "Filter by action"
// @Success 200 {object} ListAuditEventsResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Router /v1/audit-events [get]
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

// Helper types
type CreateIntentInput struct {
	WalletID      string `json:"wallet_id" validate:"required"`
	Destination   string `json:"destination" validate:"required"`
	Token         string `json:"token" validate:"required"`
	Amount        string `json:"amount" validate:"required,numeric"`
	Chain         string `json:"chain" validate:"required"`
	Creator       string `json:"creator" validate:"required"`
}

type CreateIntentResponse struct {
	Intent core.Intent `json:"intent"`
}

type ApproveIntentResponse struct {
	Intent         core.Intent `json:"intent"`
	PolicyResult   core.PolicyResult `json:"policy_result"`
}

type ExecuteIntentResponse struct {
	Intent  core.Intent `json:"intent"`
	TxHash  string `json:"tx_hash"`
	Error   string `json:"error,omitempty"`
}

type RejectIntentResponse struct {
	Intent core.Intent `json:"intent"`
}

type CancelIntentResponse struct {
	Intent core.Intent `json:"intent"`
}

type ListIntentsResponse struct {
	Intents []core.Intent `json:"intents"`
}

type GetIntentResponse struct {
	Intent core.Intent `json:"intent"`
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
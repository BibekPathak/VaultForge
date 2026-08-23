package routes

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vaultforge/vaultforge/services/api/core"
)

type IntentHandler struct {
	store        core.IntentStore
	policyEngine *core.PolicyEngine
	zkVerifier   core.ZKVerifier
	mpcSigner    core.MPCSigner
	reconciler   core.Reconciler
	audit        core.IntentAuditor
	solana       core.SolanaSubmitter
	webhooks     core.StateNotifier
	txStore      core.TransactionStore
}

func NewIntentHandler(
	store core.IntentStore,
	policyEngine *core.PolicyEngine,
	zkVerifier core.ZKVerifier,
	mpcSigner core.MPCSigner,
	reconciler core.Reconciler,
	audit core.IntentAuditor,
	solana core.SolanaSubmitter,
	webhooks core.StateNotifier,
	txStore core.TransactionStore,
) *IntentHandler {
	return &IntentHandler{
		store:        store,
		policyEngine: policyEngine,
		zkVerifier:   zkVerifier,
		mpcSigner:    mpcSigner,
		reconciler:   reconciler,
		audit:        audit,
		solana:       solana,
		webhooks:     webhooks,
		txStore:      txStore,
	}
}

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

func (h *IntentHandler) CreateIntent(c *gin.Context) {
	var input CreateIntentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	tenantID := c.MustGet("tenant_id").(string)
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = core.GenerateRequestID()
	}
	intent := core.NewIntent(tenantID, input.WalletID, input.Destination, input.Token, input.Chain, input.Creator, input.Amount)
	if err := h.store.Create(intent); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to create intent"})
		return
	}
	h.audit.LogIntentCreated(tenantID, input.Creator, intent.ID, requestID)
	h.webhooks.NotifyIntentStateChange(intent, "intent.created")
	c.JSON(http.StatusCreated, CreateIntentResponse{Intent: intent})
}

func (h *IntentHandler) ApproveIntent(c *gin.Context) {
	id := c.Param("id")
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = core.GenerateRequestID()
	}
	intent, err := h.store.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "intent not found"})
		return
	}
	tenantID := c.MustGet("tenant_id").(string)
	actor := c.GetHeader("X-Actor")
	if actor == "" {
		actor = "unknown"
	}
	if intent.TenantID != tenantID {
		c.JSON(http.StatusForbidden, errorResponse{Error: "access denied"})
		return
	}
	if intent.Status != "pending" {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "intent is not in pending status"})
		return
	}
	policyResult, err := h.policyEngine.Evaluate(intent)
	if err != nil {
		h.audit.LogIntentFailed(tenantID, actor, id, requestID, err.Error(), "policy_error")
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "policy evaluation failed"})
		return
	}
	if !policyResult.Allow {
		intent.FailureReason = core.FailureReason(core.PolicyDenied)
		intent.Status = "rejected"
		_ = h.store.Update(intent)
		h.audit.LogIntentPolicyDenied(tenantID, actor, id, requestID, policyResult.Reason)
		h.webhooks.NotifyIntentStateChange(intent, "intent.policy_denied")
		c.JSON(http.StatusOK, ApproveIntentResponse{Intent: intent, PolicyResult: policyResult})
		return
	}
	dailyLimit := int64(100000)
	zkProof, err := h.zkVerifier.GenerateProof(core.PolicyInputs{
		Amount: intent.Amount, DailyLimit: &dailyLimit, Recipient: intent.Destination,
		PolicyVersion: intent.PolicyVersion, IntentID: intent.ID, WalletID: intent.WalletID,
	})
	if err != nil {
		h.audit.LogIntentFailed(tenantID, actor, id, requestID, err.Error(), "zk_generation")
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "ZK proof generation failed"})
		return
	}
	zkVerified, err := h.zkVerifier.VerifyPolicyProof(zkProof)
	if err != nil || !zkVerified {
		intent.FailureReason = core.FailureReason(core.PolicyDenied)
		intent.Status = "rejected"
		_ = h.store.Update(intent)
		h.audit.LogIntentZKDenied(tenantID, actor, id, requestID)
		h.webhooks.NotifyIntentStateChange(intent, "intent.zk_denied")
		c.JSON(http.StatusOK, ApproveIntentResponse{
			Intent: intent, PolicyResult: &core.PolicyResult{Allow: false, Reason: "ZK proof verification failed"},
		})
		return
	}
	now := time.Now().UTC()
	intent.Status = "approved"
	intent.ApprovedAt = &now
	if err := h.store.Update(intent); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to update intent"})
		return
	}
	h.audit.LogIntentApproved(tenantID, actor, id, requestID, policyResult)
	h.webhooks.NotifyIntentStateChange(intent, "intent.approved")
	c.JSON(http.StatusOK, ApproveIntentResponse{Intent: intent, PolicyResult: policyResult})
}

func (h *IntentHandler) ExecuteIntent(c *gin.Context) {
	id := c.Param("id")
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = core.GenerateRequestID()
	}
	intent, err := h.store.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "intent not found"})
		return
	}
	tenantID := c.MustGet("tenant_id").(string)
	if intent.TenantID != tenantID {
		c.JSON(http.StatusForbidden, errorResponse{Error: "access denied"})
		return
	}
	if intent.Status != "approved" {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "intent is not in approved status"})
		return
	}
	if time.Now().UTC().After(intent.Expiry) {
		intent.Status = "expired"
		intent.FailureReason = core.FailureReason(core.Expired)
		_ = h.store.Update(intent)
		h.audit.LogIntentExpired(tenantID, "system", id, requestID)
		h.webhooks.NotifyIntentStateChange(intent, "intent.expired")
		c.JSON(http.StatusOK, ExecuteIntentResponse{Intent: intent, Error: "intent expired"})
		return
	}
	existingTx, _ := h.store.GetTransactionByNonce(intent.Nonce)
	if existingTx != nil {
		c.JSON(http.StatusAccepted, ExecuteIntentResponse{Intent: intent, Error: "duplicate execution detected"})
		return
	}
	now := time.Now().UTC()
	intent.Status = "executing"
	intent.UpdatedAt = now
	if err := h.store.Update(intent); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to update intent"})
		return
	}
	h.audit.LogIntentExecuted(tenantID, "user", id, requestID)
	h.webhooks.NotifyIntentStateChange(intent, "intent.executing")

	txBuilder := core.NewTransactionBuilder(intent.WalletID, intent.Token, intent.Amount, intent.Destination)
	blockhash, err := h.solana.GetRecentBlockhash()
	if err != nil {
		log.Printf("Warning: failed to get recent blockhash: %v (using placeholder)", err)
		blockhash = "placeholder-blockhash"
	}
	txBuilder.SetBlockhash(blockhash)

	simResult, err := txBuilder.Simulate()
	if err != nil {
		intent.Status = "failed"
		intent.FailureReason = core.FailureReason(core.SimulationFailed)
		_ = h.store.Update(intent)
		h.audit.LogIntentFailed(tenantID, "system", id, requestID, err.Error(), "simulation")
		h.webhooks.NotifyIntentStateChange(intent, "intent.simulation_failed")
		c.JSON(http.StatusInternalServerError, ExecuteIntentResponse{Intent: intent, Error: "transaction simulation failed"})
		return
	}
	if !simResult.Allowed {
		intent.Status = "failed"
		intent.FailureReason = core.FailureReason(core.SimulationFailed)
		_ = h.store.Update(intent)
		h.audit.LogIntentSimulated(tenantID, "system", id, requestID, false)
		h.webhooks.NotifyIntentStateChange(intent, "intent.simulation_denied")
		c.JSON(http.StatusInternalServerError, ExecuteIntentResponse{Intent: intent, Error: "transaction simulation denied"})
		return
	}
	h.audit.LogIntentSimulated(tenantID, "system", id, requestID, true)

	policyResult, err := h.policyEngine.Evaluate(intent)
	if err != nil || !policyResult.Allow {
		intent.Status = "failed"
		intent.FailureReason = core.FailureReason(core.PolicyDenied)
		_ = h.store.Update(intent)
		reason := "policy verification failed"
		if err != nil {
			reason = err.Error()
		}
		h.audit.LogIntentPolicyDenied(tenantID, "system", id, requestID, reason)
		h.webhooks.NotifyIntentStateChange(intent, "intent.policy_denied")
		c.JSON(http.StatusInternalServerError, ExecuteIntentResponse{Intent: intent, Error: "policy verification failed before signing"})
		return
	}

	intentHash := core.ComputeIntentHash(intent)
	signingResult, err := h.mpcSigner.Sign(core.SigningInput{
		IntentHash: intentHash, TxHash: simResult.TransactionHash, Chain: intent.Chain, Context: "vaultforge-intent-" + id,
	})
	if err != nil {
		intent.Status = "failed"
		intent.FailureReason = core.FailureReason(core.SigningFailed)
		_ = h.store.Update(intent)
		h.audit.LogIntentFailed(tenantID, "system", id, requestID, err.Error(), "signing")
		h.webhooks.NotifyIntentStateChange(intent, "intent.signing_failed")
		c.JSON(http.StatusInternalServerError, ExecuteIntentResponse{Intent: intent, Error: "MPC signing failed"})
		return
	}
	h.audit.LogIntentSigned(tenantID, "system", id, requestID, signingResult.Participants)

	txBytes := txBuilder.Build()
	submitResult, err := h.solana.SubmitTransaction(txBytes)
	if err != nil {
		intent.Status = "failed"
		intent.FailureReason = core.FailureReason(core.SubmissionFailed)
		_ = h.store.Update(intent)
		h.audit.LogIntentFailed(tenantID, "system", id, requestID, err.Error(), "submission")
		h.webhooks.NotifyIntentStateChange(intent, "intent.submission_failed")
		c.JSON(http.StatusInternalServerError, ExecuteIntentResponse{Intent: intent, Error: "Solana submission failed"})
		return
	}
	if !submitResult.Success {
		intent.Status = "failed"
		intent.FailureReason = core.FailureReason(core.SubmissionFailed)
		_ = h.store.Update(intent)
		h.audit.LogIntentFailed(tenantID, "system", id, requestID, submitResult.Error, "submission")
		h.webhooks.NotifyIntentStateChange(intent, "intent.submission_failed")
		c.JSON(http.StatusInternalServerError, ExecuteIntentResponse{Intent: intent, Error: "Solana submission rejected: " + submitResult.Error})
		return
	}

	now2 := time.Now().UTC()
	intent.Status = "submitted"
	intent.ExecutedAt = &now2
	intent.TXSignature = []byte(submitResult.Signature)
	if err := h.store.Update(intent); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to update intent"})
		return
	}

	txRecord := &core.Transaction{
		ID: core.GenerateUUID(), IntentID: intent.ID, TenantID: tenantID, WalletID: intent.WalletID,
		TransactionBytes: txBytes, Blockhash: blockhash, Status: "submitted", SubmittedAt: &now2,
		ConfirmSignature: submitResult.Signature, CreatedAt: now2, UpdatedAt: now2,
	}
	_ = h.txStore.Create(txRecord)

	h.audit.LogIntentSubmittedOnChain(tenantID, "system", id, requestID, submitResult.Signature)
	h.webhooks.NotifyIntentStateChange(intent, "intent.submitted")
	go h.reconciler.Start(intent.ID)

	c.JSON(http.StatusAccepted, ExecuteIntentResponse{Intent: intent, TxHash: submitResult.Signature})
}

func (h *IntentHandler) RejectIntent(c *gin.Context) {
	id := c.Param("id")
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = core.GenerateRequestID()
	}
	intent, err := h.store.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "intent not found"})
		return
	}
	tenantID := c.MustGet("tenant_id").(string)
	actor := c.GetHeader("X-Actor")
	if actor == "" {
		actor = "unknown"
	}
	if intent.TenantID != tenantID {
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
	h.audit.LogIntentRejected(tenantID, actor, id, requestID)
	h.webhooks.NotifyIntentStateChange(intent, "intent.rejected")
	c.JSON(http.StatusOK, RejectIntentResponse{Intent: intent})
}

func (h *IntentHandler) CancelIntent(c *gin.Context) {
	id := c.Param("id")
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = core.GenerateRequestID()
	}
	intent, err := h.store.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "intent not found"})
		return
	}
	tenantID := c.MustGet("tenant_id").(string)
	actor := c.GetHeader("X-Actor")
	if actor == "" {
		actor = "unknown"
	}
	if intent.TenantID != tenantID {
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
	h.audit.LogIntentRejected(tenantID, actor, id, requestID)
	h.webhooks.NotifyIntentStateChange(intent, "intent.cancelled")
	c.JSON(http.StatusOK, CancelIntentResponse{Intent: intent})
}

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

func (h *IntentHandler) ListWallets(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(string)
	wallets, err := h.store.ListWallets(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to list wallets"})
		return
	}
	c.JSON(http.StatusOK, ListWalletsResponse{Wallets: wallets})
}

func (h *IntentHandler) ListTransactions(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(string)
	transactions, err := h.store.ListTransactions(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to list transactions"})
		return
	}
	c.JSON(http.StatusOK, ListTransactionsResponse{Transactions: transactions})
}

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

type CreateIntentInput struct {
	WalletID    string `json:"wallet_id" binding:"required"`
	Destination string `json:"destination" binding:"required"`
	Token       string `json:"token" binding:"required"`
	Amount      int64  `json:"amount" binding:"required,gt=0"`
	Chain       string `json:"chain" binding:"required"`
	Creator     string `json:"creator" binding:"required"`
}
type CreateIntentResponse struct { Intent *core.Intent `json:"intent"` }
type ApproveIntentResponse struct { Intent *core.Intent `json:"intent"`; PolicyResult *core.PolicyResult `json:"policy_result"` }
type ExecuteIntentResponse struct { Intent *core.Intent `json:"intent"`; TxHash string `json:"tx_hash,omitempty"`; Error string `json:"error,omitempty"` }
type RejectIntentResponse struct { Intent *core.Intent `json:"intent"` }
type CancelIntentResponse struct { Intent *core.Intent `json:"intent"` }
type ListIntentsResponse struct { Intents []core.Intent `json:"intents"` }
type GetIntentResponse struct { Intent *core.Intent `json:"intent"` }
type GetWalletResponse struct { Wallet core.Wallet `json:"wallet"` }
type ListWalletsResponse struct { Wallets []core.Wallet `json:"wallets"` }
type ListTransactionsResponse struct { Transactions []core.Transaction `json:"transactions"` }
type ListAuditEventsResponse struct { Events []core.AuditEvent `json:"events"` }
type errorResponse struct { Error string `json:"error"` }

package core

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// SolanaClient submits transactions to Solana and polls for confirmation.
type SolanaClient struct {
	rpcURL       string
	httpClient   *http.Client
	commitment   string // "confirmed", "finalized"
	maxRetries   int
	pollInterval time.Duration
}

// NewSolanaClient creates a new Solana RPC client.
func NewSolanaClient(rpcURL string) *SolanaClient {
	if rpcURL == "" {
		rpcURL = "https://api.devnet.solana.com"
	}
	return &SolanaClient{
		rpcURL: rpcURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		commitment:   "confirmed",
		maxRetries:   3,
		pollInterval: 2 * time.Second,
	}
}

// SubmitResult is the result of submitting a transaction to Solana.
type SubmitResult struct {
	Signature string
	Success   bool
	Error     string
}

// rpcRequest is a JSON-RPC 2.0 request.
type rpcRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

// rpcResponse is a JSON-RPC 2.0 response.
type rpcResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// submitTransactionResponse is the response from sendTransaction.
type submitTransactionResponse struct {
	Signature string `json:"signature"`
}

// signatureStatusResponse is the response from getSignatureStatuses.
type signatureStatusResponse struct {
	Value []signatureStatus `json:"value"`
}

type signatureStatus struct {
	Signature          string      `json:"signature"`
	ConfirmationStatus string      `json:"confirmationStatus"`
	Err                interface{} `json:"err"`
}

// SubmitTransaction submits a signed transaction to Solana with exponential backoff retry.
func (c *SolanaClient) SubmitTransaction(txBytes []byte) (*SubmitResult, error) {
	// Solana's sendTransaction expects a base64-encoded serialized transaction.
	txB64 := base64.StdEncoding.EncodeToString(txBytes)

	req := rpcRequest{
		Jsonrpc: "2.0",
		ID:      1,
		Method:  "sendTransaction",
		Params: []interface{}{
			txB64,
			map[string]string{
				"encoding":            "base64",
				"commitment":          c.commitment,
				"preflightCommitment": c.commitment,
				"maxRetries":          "5",
			},
		},
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second // 1s, 2s, 4s
			if backoff > 10*time.Second {
				backoff = 10 * time.Second
			}
			log.Printf("Retrying Solana submit (attempt %d/%d) after %v", attempt+1, c.maxRetries+1, backoff)
			time.Sleep(backoff)
		}

		resp, err := c.sendRPC(req)
		if err != nil {
			lastErr = fmt.Errorf("RPC request failed: %w", err)
			continue
		}

		if resp.Error != nil {
			// RPC-level errors (e.g., blockhash not found) are retryable
			if attempt < c.maxRetries {
				lastErr = fmt.Errorf("RPC error: %s", resp.Error.Message)
				continue
			}
			return &SubmitResult{
				Success: false,
				Error:   resp.Error.Message,
			}, nil
		}

		var sigResp submitTransactionResponse
		if err := json.Unmarshal(resp.Result, &sigResp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		log.Printf("Transaction submitted to Solana: signature=%s (attempt %d)", sigResp.Signature, attempt+1)

		return &SubmitResult{
			Signature: sigResp.Signature,
			Success:   true,
		}, nil
	}

	return nil, fmt.Errorf("submit failed after %d retries: %w", c.maxRetries, lastErr)
}

// WaitForConfirmation polls Solana for transaction confirmation.
func (c *SolanaClient) WaitForConfirmation(signature string) (bool, error) {
	for attempt := 0; attempt < c.maxRetries*10; attempt++ {
		status, err := c.GetSignatureStatus(signature)
		if err != nil {
			log.Printf("Confirmation poll error (attempt %d): %v", attempt+1, err)
			time.Sleep(c.pollInterval)
			continue
		}

		if status == nil {
			time.Sleep(c.pollInterval)
			continue
		}

		if status.Err != nil {
			return false, fmt.Errorf("transaction failed on chain: %v", status.Err)
		}

		if status.ConfirmationStatus == "confirmed" || status.ConfirmationStatus == "finalized" {
			log.Printf("Transaction confirmed: signature=%s status=%s", signature, status.ConfirmationStatus)
			return true, nil
		}

		time.Sleep(c.pollInterval)
	}

	return false, fmt.Errorf("confirmation timeout after %d attempts", c.maxRetries*10)
}

// GetSignatureStatus returns the status of a transaction signature.
func (c *SolanaClient) GetSignatureStatus(signature string) (*signatureStatus, error) {
	req := rpcRequest{
		Jsonrpc: "2.0",
		ID:      1,
		Method:  "getSignatureStatuses",
		Params: []interface{}{
			[]string{signature},
			map[string]bool{
				"searchTransactionHistory": true,
			},
		},
	}

	resp, err := c.sendRPC(req)
	if err != nil {
		return nil, err
	}

	var statusResp signatureStatusResponse
	if err := json.Unmarshal(resp.Result, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to parse status: %w", err)
	}

	if len(statusResp.Value) == 0 {
		return nil, nil
	}

	return &statusResp.Value[0], nil
}

// GetRecentBlockhash fetches a recent blockhash from Solana.
func (c *SolanaClient) GetRecentBlockhash() (string, error) {
	req := rpcRequest{
		Jsonrpc: "2.0",
		ID:      1,
		Method:  "getLatestBlockhash",
		Params: []interface{}{
			map[string]string{
				"commitment": c.commitment,
			},
		},
	}

	resp, err := c.sendRPC(req)
	if err != nil {
		return "", fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	var result struct {
		Value struct {
			Blockhash string `json:"blockhash"`
		} `json:"value"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", fmt.Errorf("failed to parse blockhash: %w", err)
	}

	return result.Value.Blockhash, nil
}

// sendRPC sends a JSON-RPC request to Solana.
func (c *SolanaClient) sendRPC(req rpcRequest) (*rpcResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.rpcURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &rpcResp, nil
}

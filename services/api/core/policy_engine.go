package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// PolicyEngine evaluates intents against policy rules before signing.
type PolicyEngine struct {
	db DBQuerier
}

// DBQuerier defines the minimal database operations needed by the policy engine.
type DBQuerier interface {
	GetWallet(id string) (*Wallet, error)
	GetTenant(id string) (*Tenant, error)
	ListPolicies(tenantID string) ([]*Policy, error)
}

// Policy represents a policy rule that can be evaluated.
type Policy struct {
	ID        string
	TenantID  string
	Name      string
	RuleType  string // "daily_limit", "single_tx_limit", "allowed_recipients", etc.
	Config    json.RawMessage
	IsActive  bool
}

// Evaluate evaluates all applicable policies for the given intent.
// Returns ALLOW only if all policies pass; otherwise DENY with a structured reason.
func (pe *PolicyEngine) Evaluate(intent *Intent) (*PolicyResult, error) {
	// Look up the wallet to get tenant and policy configuration
	wallet, err := pe.db.GetWallet(intent.WalletID)
	if err != nil {
		return nil, fmt.Errorf("failed to look up wallet %s: %w", intent.WalletID, err)
	}

	// Look up tenant
	tenant, err := pe.db.GetTenant(wallet.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to look up tenant %s: %w", wallet.TenantID, err)
	}

	// Retrieve active policies for the tenant
	policies, err := pe.db.ListPolicies(tenant.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies for tenant %s: %w", tenant.ID, err)
	}

	// Evaluate each policy
	var allPassed bool = true
	var deniedReasons []string

	for _, p := range policies {
		if !p.IsActive {
			continue
		}

		switch p.RuleType {
		case "daily_limit":
			if !pe.evaluateDailyLimit(p, intent, wallet) {
				allPassed = false
				deniedReasons = append(deniedReasons, fmt.Sprintf("daily limit policy: %s", p.Name))
			}
		case "single_tx_limit":
			if !pe.evaluateSingleTxLimit(p, intent, wallet) {
				allPassed = false
				deniedReasons = append(deniedReasons, fmt.Sprintf("single transaction limit policy: %s", p.Name))
			}
		case "allowed_recipients":
			if !pe.evaluateAllowedRecipients(p, intent) {
				allPassed = false
				deniedReasons = append(deniedReasons, fmt.Sprintf("allowed recipients policy: %s", p.Name))
			}
		case "blocked_recipients":
			if !pe.evaluateBlockedRecipients(p, intent) {
				allPassed = false
				deniedReasons = append(deniedReasons, fmt.Sprintf("blocked recipients policy: %s", p.Name))
			}
		case "allowed_tokens":
			if !pe.evaluateAllowedTokens(p, intent) {
				allPassed = false
				deniedReasons = append(deniedReasons, fmt.Sprintf("allowed tokens policy: %s", p.Name))
			}
		case "required_signatures":
			if !pe.evaluateRequiredSigs(p, intent) {
				allPassed = false
				deniedReasons = append(deniedReasons, fmt.Sprintf("required signatures policy: %s", p.Name))
			}
		case "time_restriction":
			if !pe.evaluateTimeRestriction(p, intent) {
				allPassed = false
				deniedReasons = append(deniedReasons, fmt.Sprintf("time restriction policy: %s", p.Name))
			}
		default:
			// Unknown policy type - skip or log
		}
	}

	if allPassed {
		return &PolicyResult{Allow: true}, nil
	}

	return &PolicyResult{
		Allow:   false,
		Reason:  strings.Join(deniedReasons, "; "),
	}, nil
}

// evaluateDailyLimit checks if the intent amount is within the daily limit for the wallet.
func (pe *PolicyEngine) evaluateDailyLimit(p *Policy, intent *Intent, wallet *Wallet) bool {
	// Parse the daily limit from policy config
	dailyLimit := pe.parseAmount(p.Config, wallet.DailyLimit)
	if dailyLimit <= 0 {
		// No daily limit configured, allow
		return true
	}

	// Parse the intent amount
	amount := pe.parseAmount(intent.Amount, 0)

	// Check if amount is within daily limit
	if amount > dailyLimit {
		return false
	}

	// Additional check: single transaction limit within daily
	singleLimit := pe.parseAmount(p.Config, 0)
	if singleLimit > 0 && amount > singleLimit {
		return false
	}

	return true
}

// evaluateSingleTxLimit checks if the intent amount is within the single transaction limit.
func (pe *PolicyEngine) evaluateSingleTxLimit(p *Policy, intent *Intent, wallet *Wallet) bool {
	// Parse the single transaction limit from policy config
	singleLimit := pe.parseAmount(p.Config, 0)
	if singleLimit <= 0 {
		// No single transaction limit configured, allow
		return true
	}

	// Parse the intent amount
	amount := pe.parseAmount(intent.Amount, 0)

	if amount > singleLimit {
		return false
	}

	return true
}

// evaluateAllowedRecipients checks if the destination recipient is allowed.
func (pe *PolicyEngine) evaluateAllowedRecipients(p *Policy, intent *Intent) bool {
	// Parse allowed recipients from policy config
	var allowed []string
	if err := pe.parseRecipients(p.Config, &allowed); err != nil {
		// If we can't parse, default to allowing
		return true
	}

	// Check if the destination is in the allowed list
	for _, allowedRecipient := range allowed {
		if strings.EqualFold(allowedRecipient, intent.Destination) {
			return true
		}
	}

	// Destination not in allowed list
	return false
}

// evaluateBlockedRecipients checks if the destination recipient is blocked.
func (pe *PolicyEngine) evaluateBlockedRecipients(p *Policy, intent *Intent) bool {
	// Parse blocked recipients from policy config
	var blocked []string
	if err := pe.parseRecipients(p.Config, &blocked); err != nil {
		return true // default to allow if can't parse
	}

	// Check if the destination is in the blocked list
	for _, blockedRecipient := range blocked {
		if strings.EqualFold(blockedRecipient, intent.Destination) {
			return false
		}
	}

	// Destination not in blocked list
	return true
}

// evaluateAllowedTokens checks if the token is allowed.
func (pe *PolicyEngine) evaluateAllowedTokens(p *Policy, intent *Intent) bool {
	// Parse allowed tokens from policy config
	var allowed []string
	if err := pe.parseTokens(p.Config, &allowed); err != nil {
		return true // default to allow if can't parse
	}

	// Check if the token is in the allowed list
	for _, allowedToken := range allowed {
		if strings.EqualFold(allowedToken, intent.Token) {
			return true
		}
	}

	// Token not in allowed list
	return false
}

// evaluateRequiredSigs checks if the intent has the required number of signatures.
func (pe *PolicyEngine) evaluateRequiredSigs(p *Policy, intent *Intent) bool {
	// Parse required signatures from policy config
	required := pe.parseRequiredSigs(p.Config)
	if required <= 0 {
		// Default: require 2-of-3
		required = 2
	}

	// Check if the intent has enough approvers
	if len(intent.Approvers) >= required {
		return true
	}

	return false
}

// evaluateTimeRestriction checks if the intent is within the allowed time window.
func (pe *PolicyEngine) evaluateTimeRestriction(p *Policy, intent *Intent) bool {
	// Parse time restriction from policy config
	startStr, endStr := pe.parseTimeWindow(p.Config)

	// If no time window configured, allow
	if startStr == "" && endStr == "" {
		return true
	}

	now := time.Now().UTC()

	// Parse start time
	var startTime time.Time
	if startStr != "" {
		var err error
		startTime, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			// If we can't parse, default to allowing
			return true
		}
	}

	// Parse end time
	var endTime time.Time
	if endStr != "" {
		var err error
		endTime, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			return true
		}
	}

	// Check if current time is within the restriction window
	if !startTime.IsZero() && now.Before(startTime) {
		return false // before start time
	}

	if !endTime.IsZero() && now.After(endTime) {
		return false // after end time
	}

	return true
}

// parseAmount parses a string amount value, returning the int64 value.
func (pe *PolicyEngine) parseAmount(s string, fallback int64) int64 {
	if s == "" {
		return fallback
	}
	var v int64
	fmt.Sscanf(s, "%d", &v)
	return v
}

// parseRecipients parses the policy config to extract allowed/blocked recipients.
func (pe *PolicyEngine) parseRecipients(config json.RawMessage, recipients *[]string) error {
	if len(config) == 0 {
		return nil
	}
	return json.Unmarshal(config, recipients)
}

// parseTokens parses the policy config to extract allowed tokens.
func (pe *PolicyEngine) parseTokens(config json.RawMessage, tokens *[]string) error {
	if len(config) == 0 {
		return nil
	}
	return json.Unmarshal(config, tokens)
}

// parseRequiredSigs parses the required signature count from policy config.
func (pe *PolicyEngine) parseRequiredSigs(config json.RawMessage) int {
	var v int
	if len(config) == 0 {
		return v
	}
	// Try to parse as integer
	s := string(config)
	// Remove any brackets or quotes
	s = strings.Trim(s, "[]\"")
	_, err := fmt.Sscanf(s, "%d", &v)
	if err != nil {
		return 0 // default later
	}
	return v
}

// parseTimeWindow parses the time restriction start and end times from policy config.
func (pe *PolicyEngine) parseTimeWindow(config json.RawMessage) (string, string) {
	var structConfig struct {
		Start string `json:"start"`
		End   string `json:"end"`
	}
	if len(config) == 0 {
		return "", ""
	}
	json.Unmarshal(config, &structConfig)
	return structConfig.Start, structConfig.End
}

// NewPolicyEngine creates a new PolicyEngine with the given database querier.
func NewPolicyEngine(db DBQuerier) *PolicyEngine {
	return &PolicyEngine{db: db}
}
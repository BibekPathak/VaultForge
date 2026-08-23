package core

import (
	"encoding/json"
	"fmt"
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

// Evaluate evaluates all applicable policies for the given intent.
func (pe *PolicyEngine) Evaluate(intent *Intent) (*PolicyResult, error) {
	wallet, err := pe.db.GetWallet(intent.WalletID)
	if err != nil {
		return nil, fmt.Errorf("failed to look up wallet %s: %w", intent.WalletID, err)
	}

	tenant, err := pe.db.GetTenant(wallet.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to look up tenant %s: %w", wallet.TenantID, err)
	}

	policies, err := pe.db.ListPolicies(tenant.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies for tenant %s: %w", tenant.ID, err)
	}

	var allPassed = true
	var deniedReasons []string

	for _, p := range policies {
		if !p.IsActive {
			continue
		}

		switch p.RuleType {
		case "daily_limit":
			if !pe.evaluateDailyLimit(p, intent, wallet) {
				allPassed = false
				deniedReasons = append(deniedReasons, fmt.Sprintf("daily limit: %s", p.Name))
			}
		case "single_tx_limit":
			if !pe.evaluateSingleTxLimit(p, intent) {
				allPassed = false
				deniedReasons = append(deniedReasons, fmt.Sprintf("single tx limit: %s", p.Name))
			}
		case "allowed_recipients":
			if !pe.evaluateAllowedRecipients(p, intent) {
				allPassed = false
				deniedReasons = append(deniedReasons, fmt.Sprintf("allowed recipients: %s", p.Name))
			}
		case "blocked_recipients":
			if !pe.evaluateBlockedRecipients(p, intent) {
				allPassed = false
				deniedReasons = append(deniedReasons, fmt.Sprintf("blocked recipients: %s", p.Name))
			}
		case "allowed_tokens":
			if !pe.evaluateAllowedTokens(p, intent) {
				allPassed = false
				deniedReasons = append(deniedReasons, fmt.Sprintf("allowed tokens: %s", p.Name))
			}
		case "required_signatures":
			if !pe.evaluateRequiredSigs(p, intent) {
				allPassed = false
				deniedReasons = append(deniedReasons, fmt.Sprintf("required sigs: %s", p.Name))
			}
		case "time_restriction":
			if !pe.evaluateTimeRestriction(p, intent) {
				allPassed = false
				deniedReasons = append(deniedReasons, fmt.Sprintf("time restriction: %s", p.Name))
			}
		}
	}

	if allPassed {
		return &PolicyResult{Allow: true}, nil
	}

	return &PolicyResult{
		Allow:  false,
		Reason: strings.Join(deniedReasons, "; "),
	}, nil
}

func (pe *PolicyEngine) evaluateDailyLimit(p *Policy, intent *Intent, wallet *Wallet) bool {
	dailyLimit := pe.parseConfigInt64(p.Config, "limit", wallet.DailyLimit)
	if dailyLimit <= 0 {
		return true
	}
	amount := parseIntentAmount(intent.Amount)
	return amount <= dailyLimit
}

func (pe *PolicyEngine) evaluateSingleTxLimit(p *Policy, intent *Intent) bool {
	singleLimit := pe.parseConfigInt64(p.Config, "limit", 0)
	if singleLimit <= 0 {
		return true
	}
	amount := parseIntentAmount(intent.Amount)
	return amount <= singleLimit
}

func (pe *PolicyEngine) evaluateAllowedRecipients(p *Policy, intent *Intent) bool {
	var allowed []string
	if err := json.Unmarshal(p.Config, &allowed); err != nil {
		return true
	}
	for _, r := range allowed {
		if strings.EqualFold(r, intent.Destination) {
			return true
		}
	}
	return false
}

func (pe *PolicyEngine) evaluateBlockedRecipients(p *Policy, intent *Intent) bool {
	var blocked []string
	if err := json.Unmarshal(p.Config, &blocked); err != nil {
		return true
	}
	for _, r := range blocked {
		if strings.EqualFold(r, intent.Destination) {
			return false
		}
	}
	return true
}

func (pe *PolicyEngine) evaluateAllowedTokens(p *Policy, intent *Intent) bool {
	var allowed []string
	if err := json.Unmarshal(p.Config, &allowed); err != nil {
		return true
	}
	for _, t := range allowed {
		if strings.EqualFold(t, intent.Token) {
			return true
		}
	}
	return false
}

func (pe *PolicyEngine) evaluateRequiredSigs(p *Policy, intent *Intent) bool {
	required := pe.parseConfigInt(p.Config, "required", 2)
	return len(intent.Approvers) >= required
}

func (pe *PolicyEngine) evaluateTimeRestriction(p *Policy, intent *Intent) bool {
	var window struct {
		Start string `json:"start"`
		End   string `json:"end"`
	}
	if err := json.Unmarshal(p.Config, &window); err != nil {
		return true
	}
	if window.Start == "" && window.End == "" {
		return true
	}

	now := time.Now().UTC()
	if window.Start != "" {
		start, err := time.Parse(time.RFC3339, window.Start)
		if err == nil && now.Before(start) {
			return false
		}
	}
	if window.End != "" {
		end, err := time.Parse(time.RFC3339, window.End)
		if err == nil && now.After(end) {
			return false
		}
	}
	return true
}

func (pe *PolicyEngine) parseConfigInt64(config json.RawMessage, key string, fallback int64) int64 {
	if len(config) == 0 {
		return fallback
	}
	var m map[string]interface{}
	if err := json.Unmarshal(config, &m); err != nil {
		return fallback
	}
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return int64(val)
		case json.Number:
			n, _ := val.Int64()
			return n
		}
	}
	return fallback
}

func (pe *PolicyEngine) parseConfigInt(config json.RawMessage, key string, fallback int) int {
	if len(config) == 0 {
		return fallback
	}
	var m map[string]interface{}
	if err := json.Unmarshal(config, &m); err != nil {
		return fallback
	}
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return int(f)
		}
	}
	return fallback
}

func parseIntentAmount(s string) int64 {
	var v int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			v = v*10 + int64(c-'0')
		}
	}
	return v
}

// NewPolicyEngine creates a new PolicyEngine.
func NewPolicyEngine(db DBQuerier) *PolicyEngine {
	return &PolicyEngine{db: db}
}

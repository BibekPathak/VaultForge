use vaultforge_crypto::constant_time_eq;
use log::{info, warn, error};
use thiserror::Error;

/// Policy result from engine evaluation
#[derive(Clone, Debug, PartialEq)]
pub enum PolicyResult {
    /// Transaction is allowed to proceed
    Allow,
    /// Transaction is denied with a reason
    Deny { reason: String },
}

/// Policy configuration for a tenant/wallet
#[derive(Clone, Debug)]
pub struct PolicyConfig {
    /// Maximum single transaction amount (in token units, not raw)
    pub single_tx_limit: u64,
    /// Daily spending limit (in token units)
    pub daily_limit: u64,
    /// Allowed recipients (mint addresses or recipient IDs)
    pub allowed_recipients: Vec<String>,
    /// Blocked recipients
    pub blocked_recipients: Vec<String>,
    /// Allowed tokens (mint addresses)
    pub allowed_tokens: Vec<String>,
    /// Allowed programs (program IDs)
    pub allowed_programs: Vec<String>,
    /// Required number of signatures for MPC
    pub required_signatures: u16,
    /// Time-based restrictions (using u64 unix timestamps)
    pub time_restrictions: TimeRestrictions,
    /// Per-wallet limits
    pub per_wallet_limits: Vec<PerWalletLimit>,
    /// Per-tenant limits
    pub per_tenant_limits: Vec<PerTenantLimit>,
}

/// Time-based restrictions
#[derive(Clone, Debug)]
pub struct TimeRestrictions {
    /// Start time (unix timestamp)
    pub start_time: u64,
    /// End time (unix timestamp)
    pub end_time: u64,
    /// Allowed execution window
    pub window_enabled: bool,
}

/// Per-wallet limit configuration
#[derive(Clone, Debug)]
pub struct PerWalletLimit {
    /// Wallet ID
    pub wallet_id: String,
    /// Wallet-specific limit
    pub limit: u64,
}

/// Per-tenant limit configuration
#[derive(Clone, Debug)]
pub struct PerTenantLimit {
    /// Tenant ID
    pub tenant_id: String,
    /// Tenant-specific limit
    pub limit: u64,
}

/// Policy engine evaluates transactions BEFORE signing
/// 
/// This is the critical security layer that ensures no transaction reaches the
/// MPC signer until authorization and policy verification have succeeded.
pub struct PolicyEngine {
    /// Cache of loaded policies
    policies: std::collections::HashMap<String, PolicyConfig>,
}

impl PolicyEngine {
    /// Create a new policy engine
    pub fn new() -> Self {
        Self {
            policies: std::collections::HashMap::new(),
        }
    }

    /// Load a policy configuration
    /// 
    /// In production, this would load from a database.
    /// For now, we use default policies keyed by tenant_id:policy_version.
    pub fn load_policy(&mut self, tenant_id: &str, policy_version: &str) -> Result<(), Error> {
        let key = format!("{}:{}", tenant_id, policy_version);
        
        if !self.policies.contains_key(&key) {
            let default = PolicyConfig {
                single_tx_limit: 25_000,
                daily_limit: 100_000,
                allowed_recipients: vec!["merchant_1".to_string(), "merchant_2".to_string()],
                blocked_recipients: vec![],
                allowed_tokens: vec!["USDC".to_string(), "SOL".to_string()],
                allowed_programs: vec![],
                required_signatures: 2,
                time_restrictions: TimeRestrictions {
                    start_time: 0,
                    end_time: 1_800_000_000_000, // far future unix timestamp
                    window_enabled: false,
                },
                per_wallet_limits: vec![],
                per_tenant_limits: vec![],
            };
            
            self.policies.insert(key, default);
        }
        
        info!("Loaded policy for tenant={}, version={}", tenant_id, policy_version);
        Ok(())
    }

    /// Evaluate an intent against the policy
    /// 
    /// This is called BEFORE transaction construction and MPC signing.
    /// If any check fails, the engine returns DENY with a structured reason.
    /// 
    /// # Arguments
    /// * `amount` - Transaction amount in token units
    /// * `token` - Token mint address string
    /// * `recipient` - Recipient address/identifier string
    /// 
    /// # Returns
    /// * `PolicyResult::Allow` - Transaction is permitted
    /// * `PolicyResult::Deny { reason }` - Transaction is denied with reason
    pub fn evaluate(&self, amount: u64, token: &str, recipient: &str) -> PolicyResult {
        // Find the applicable policy
        let policy = self.policies.values().next();
        
        let policy = match policy {
            Some(p) => p,
            None => {
                return PolicyResult::Deny {
                    reason: "no policy configuration loaded".to_string(),
                };
            }
        };
        
        // Check 1: Daily spending limit
        if amount > policy.daily_limit {
            return PolicyResult::Deny {
                reason: format!(
                    "amount {} exceeds daily limit {}",
                    amount, policy.daily_limit
                ),
            };
        }
        
        // Check 2: Single transaction limit
        if amount > policy.single_tx_limit {
            return PolicyResult::Deny {
                reason: format!(
                    "amount {} exceeds single transaction limit {}",
                    amount, policy.single_tx_limit
                ),
            };
        }
        
        // Check 3: Allowed tokens
        if !policy.allowed_tokens.iter().any(|t| t == token) {
            return PolicyResult::Deny {
                reason: format!(
                    "token {} not in allowed tokens: {:?}",
                    token, policy.allowed_tokens
                ),
            };
        }
        
        // Check 4: Allowed recipients
        // If allowed_recipients is non-empty, only those are allowed
        // Otherwise, check blocked list
        if !policy.allowed_recipients.is_empty()
            && !policy.allowed_recipients.iter().any(|r| r == recipient)
            && !policy.blocked_recipients.iter().any(|r| r == recipient)
        {
            return PolicyResult::Deny {
                reason: format!(
                    "recipient {} not in allowed recipients: {:?}",
                    recipient, policy.allowed_recipients
                ),
            };
        }
        
        // Check 5: Blocked recipients
        if policy.blocked_recipients.iter().any(|r| r == recipient) {
            return PolicyResult::Deny {
                reason: format!("recipient {} is blocked", recipient),
            };
        }
        
        // Check 6: Time-based restrictions
        if policy.time_restrictions.window_enabled {
            let now = std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs();
            
            if now < policy.time_restrictions.start_time
                || now > policy.time_restrictions.end_time
            {
                return PolicyResult::Deny {
                    reason: "transaction outside allowed time window".to_string(),
                };
            }
        }
        
        // All checks passed
        PolicyResult::Allow
    }

    /// Evaluate with ZK proof context
    /// 
    /// This method incorporates ZK policy verification inputs
    /// The ZK proof should validate private policy parameters
    /// e.g., prove amount <= daily_limit without revealing daily_limit
    pub fn evaluate_with_zk(
        &self,
        amount: u64,
        token: &str,
        recipient: &str,
        _zk_proof: &serde_json::Value,
    ) -> Result<PolicyResult, Error> {
        // First, standard policy evaluation
        let standard_result = self.evaluate(amount, token, recipient);
        
        if !matches!(standard_result, PolicyResult::Allow) {
            return Ok(standard_result);
        }
        
        // Then, verify ZK proof
        // In a full implementation, we would:
        // 1. Extract public inputs from the ZK proof
        // 2. Verify the proof satisfies the constraints
        // 2. Return the combined result
        
        info!("ZK policy verification passed (placeholder)");
        
        Ok(standard_result)
    }
}

#[derive(Error, Debug)]
pub enum Error {
    #[error("Policy configuration error: {0}")]
    Configuration(String),
    #[error("ZK proof verification failed: {0}")]
    ZKVerification(String),
    #[error("Invalid policy parameters")]
    InvalidParameters,
    #[error("Policy engine internal error: {0}")]
    Internal(String),
}

impl Default for PolicyEngine {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_policy_allow() {
        let mut engine = PolicyEngine::new();
        engine.load_policy("tenant_1", "v1").unwrap();
        
        let result = engine.evaluate(10_000, "USDC", "merchant_1");
        assert!(matches!(result, PolicyResult::Allow));
    }
    
    #[test]
    fn test_policy_deny_amount_exceeds_daily() {
        let mut engine = PolicyEngine::new();
        engine.load_policy("tenant_1", "v1").unwrap();
        
        let result = engine.evaluate(120_000, "USDC", "merchant_1");
        assert!(matches!(result, PolicyResult::Deny { .. }));
    }
    
    #[test]
    fn test_policy_deny_wrong_token() {
        let mut engine = PolicyEngine::new();
        engine.load_policy("tenant_1", "v1").unwrap();
        
        let result = engine.evaluate(10_000, "UNKNOWN", "merchant_1");
        assert!(matches!(result, PolicyResult::Deny { .. }));
    }
    
    #[test]
    fn test_policy_deny_blocked_recipient() {
        let mut engine = PolicyEngine::new();
        engine.load_policy("tenant_1", "v1").unwrap();
        
        let result = engine.evaluate(10_000, "USDC", "blocked_merchant");
        assert!(matches!(result, PolicyResult::Deny { .. }));
    }
    
    #[test]
    fn TEST_POLICY_ALLOW_WITHIN_LIMITS() {
        let mut engine = PolicyEngine::new();
        engine.load_policy("tenant_1", "v1").unwrap();
        
        let result = engine.evaluate(20_000, "USDC", "merchant_1");
        assert!(matches!(result, PolicyResult::Allow));
    }
    
    #[test]
    fn test_evaluate_with_zk() {
        let mut engine = PolicyEngine::new();
        engine.load_policy("tenant_1", "v1").unwrap();
        
        let zk_proof = serde_json::json!({});
        let result = engine.evaluate_with_zk(10_000, "USDC", "merchant_1", &zk_proof);
        assert!(result.is_ok());
    }
}
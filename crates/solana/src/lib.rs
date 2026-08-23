use vaultforge_crypto::Sha256;
use thiserror::Error;
use log::info;

// Core Solana transaction infrastructure
// This crate provides the transaction building, simulation, and replay protection
// that bridges intent authorization and MPC signing.

/// Transaction builder state
#[derive(Clone)]
pub struct TransactionBuilder {
    pub wallet_id: String,
    pub destination: String,
    pub token: String,
    pub amount: u64,
    pub blockhash: Option<String>,
    pub priority_fee: u64,
    pub is_sol_transfer: bool,
}

impl TransactionBuilder {
    /// Create new SOL transfer builder
    pub fn new_sol_transfer(wallet_id: String, destination: String, amount: u64) -> Self {
        Self {
            wallet_id,
            destination,
            token: "SOL".to_string(),
            amount,
            blockhash: None,
            priority_fee: 0,
            is_sol_transfer: true,
        }
    }

    /// Create new Token transfer builder
    pub fn new_token_transfer(wallet_id: String, token: String, destination: String, amount: u64) -> Self {
        Self {
            wallet_id,
            destination,
            token,
            amount,
            blockhash: None,
            priority_fee: 0,
            is_sol_transfer: false,
        }
    }

    /// Set recent blockhash
    pub fn set_blockhash(&mut self, blockhash: String) {
        self.blockhash = Some(blockhash);
    }

    /// Set priority fee
    pub fn set_priority_fee(&mut self, fee: u64) {
        self.priority_fee = fee;
    }

    /// Build canonical transaction bytes bound to intent
    /// 
    /// The exact bytes being signed must be bound to the approved intent.
    /// This is the critical representation that MPC signers will sign.
    pub fn build(&self) -> Vec<u8> {
        let mut tx_bytes = Vec::new();
        
        // 1. Intent ID binding (32 bytes at start)
        // These bytes are filled in by the MPC signing layer with the actual intent hash
        tx_bytes.extend_from_slice(&[0u8; 32]);
        
        // 2. Transaction type: 0=SOL, 1=Token (determined by is_sol_transfer flag)
        let tx_type: u8 = if self.is_sol_transfer { 0 } else { 1 };
        tx_bytes.push(tx_type);
        
        // 3. Amount in token units (8 bytes, u64 big-endian)
        tx_bytes.extend_from_slice(&self.amount.to_be_bytes());
        
        // 4. Destination pubkey (32 bytes) - pad or truncate
        let mut dest_padded = [0u8; 32];
        let dest_bytes = self.destination.as_bytes();
        let len = std::cmp::min(dest_bytes.len(), 32);
        dest_padded[..len].copy_from_slice(&dest_bytes[..len]);
        tx_bytes.extend_from_slice(&dest_padded);
        
        // 5. Mint/token identifier (32 bytes) - pad or truncate
        let mut token_padded = [0u8; 32];
        let token_bytes = self.token.as_bytes();
        let tlen = std::cmp::min(token_bytes.len(), 32);
        token_padded[..tlen].copy_from_slice(&token_bytes[..tlen]);
        tx_bytes.extend_from_slice(&token_padded);
        
        // 6. Blockhash (32 bytes - hash of blockhash string or placeholder)
        if let Some(bh) = &self.blockhash {
            let bh_hash = Sha256::hash(bh.as_bytes());
            tx_bytes.extend_from_slice(&bh_hash);
        } else {
            tx_bytes.extend_from_slice(&[0u8; 32]);
        }
        
        // 7. Priority fee (8 bytes)
        tx_bytes.extend_from_slice(&self.priority_fee.to_be_bytes());
        
        // 8. Timestamp (8 bytes unix epoch)
        tx_bytes.extend_from_slice(&std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_secs().to_be_bytes());
        
        info!("Built canonical transaction bytes: {} bytes", tx_bytes.len());
        tx_bytes
    }

    /// Compute transaction hash for signing
    pub fn compute_transaction_hash(&self) -> Vec<u8> {
        let tx_bytes = self.build();
        Sha256::hash(&tx_bytes).to_vec()
    }

    /// Compute idempotency key from intent parameters
    pub fn compute_idempotency_key(&self) -> String {
        format!(
            "{}::{}::{}:{}",
            self.wallet_id, self.destination, self.token, self.amount
        )
    }
}

/// Transaction builder errors
#[derive(Error, Debug)]
pub enum BuilderError {
    #[error("Builder not properly configured")]
    NotConfigured,
    #[error("Invalid amount")]
    InvalidAmount,
    #[error("Blockhash missing")]
    MissingBlockhash,
}

/// Transaction simulation result
#[derive(Clone, Debug)]
pub struct SimulationResult {
    pub allowed: bool,
    pub transaction_hash: Option<Vec<u8>>,
    pub message: String,
}

/// Simulation errors
#[derive(Error, Debug)]
pub enum SimulationError {
    #[error("Insufficient funds")]
    InsufficientFunds,
    #[error("Blockhash expired")]
    BlockhashExpired,
    #[error("Simulation failed: {0}")]
    Failed(String),
}

/// Replay protection key
pub struct ReplayKey {
    pub intent_id: String,
    pub chain: String,
    pub version: u64,
    pub used: bool,
}

impl ReplayKey {
    /// Check if this key can be used (not already used)
    pub fn is_available(&self) -> bool {
        !self.used
    }

    /// Mark this key as used
    pub fn mark_used(&mut self) {
        self.used = true;
    }
}

/// Create SOL transfer builder
pub fn new_sol_transfer_builder(
    wallet_id: String,
    destination: String,
    amount: u64,
) -> TransactionBuilder {
    TransactionBuilder::new_sol_transfer(wallet_id, destination, amount)
}

/// Create Token transfer builder
pub fn new_token_transfer_builder(
    wallet_id: String,
    token: String,
    destination: String,
    amount: u64,
) -> TransactionBuilder {
    TransactionBuilder::new_token_transfer(wallet_id, token, destination, amount)
}

/// Constant-time comparison for cryptographic comparisons
pub fn constant_time_eq(a: &[u8], b: &[u8]) -> bool {
    if a.len() != b.len() {
        return false;
    }
    let mut result: u8 = 0;
    for (x, y) in a.iter().zip(b.iter()) {
        result |= x ^ y;
    }
    result == 0
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_build_sol_transfer() {
        let builder = TransactionBuilder::new_sol_transfer("wallet_1".to_string(), 
            "merchant_1".to_string(), 1000);
        let tx_bytes = builder.build();
        assert!(!tx_bytes.is_empty());
        assert_eq!(tx_bytes[32], 0u8); // SOL transfer type at byte 32 (after 32-byte intent ID)
    }
    
    #[test]
    fn test_build_token_transfer() {
        let builder = TransactionBuilder::new_token_transfer(
            "wallet_1".to_string(), "USDC".to_string(), 
            "merchant_1".to_string(), 1000);
        let tx_bytes = builder.build();
        assert!(!tx_bytes.is_empty());
        assert_eq!(tx_bytes[32], 1u8); // Token transfer type at byte 32 (after 32-byte intent ID)
    }
    
    #[test]
    fn test_constant_time_eq() {
        use vaultforge_crypto::constant_time_eq;
        assert!(constant_time_eq(b"abc", b"abc"));
        assert!(!constant_time_eq(b"abc", b"abd"));
    }
    
    #[test]
    fn test_replay_key() {
        let mut key = ReplayKey {
            intent_id: "intent-123".to_string(),
            chain: "solana".to_string(),
            version: 1,
            used: false,
        };
        assert!(key.is_available());
        key.mark_used();
        assert!(!key.is_available());
    }
}

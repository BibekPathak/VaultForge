use vaultforge_crypto::Sha256;
use log::info;
use thiserror::Error;

// Transaction builder state - no solana_sdk dependencies
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

    pub fn set_blockhash(&mut self, blockhash: String) {
        self.blockhash = Some(blockhash);
    }

    pub fn set_priority_fee(&mut self, fee: u64) {
        self.priority_fee = fee;
    }

    /// Build canonical transaction bytes bound to intent
    pub fn build(&self) -> Vec<u8> {
        let mut tx_bytes = Vec::new();
        
        // Intent ID binding (first 32 bytes - placeholder)
        let intent_id_bytes = &[0u8; 32];
        tx_bytes.extend_from_slice(intent_id_bytes);
        
        // Transaction type: 0 = SOL, 1 = Token
        tx_bytes.push(if self.is_sol_transfer { 0u8 } else { 1u8 });
        
        // Amount in token units
        tx_bytes.extend_from_slice(&self.amount.to_be_bytes());
        
        // Destination
        let dest_bytes = self.destination.as_bytes();
        tx_bytes.extend_from_slice(dest_bytes);
        
        // Token mint
        let token_bytes = self.token.as_bytes();
        tx_bytes.extend_from_slice(token_bytes);
        
        // Blockhash (or placeholder)
        if let Some(bh) = &self.blockhash {
            tx_bytes.extend_from_slice(bh.as_bytes());
        } else {
            tx_bytes.extend_from_slice(&[0u8; 32]);
        }
        
        // Priority fee
        tx_bytes.extend_from_slice(&self.priority_fee.to_be_bytes());
        
        // Timestamp
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
}

#[derive(Error, Debug)]
pub enum TransactionBuilderError {
    #[error("Builder not properly configured")]
    NotConfigured,
    #[error("Invalid amount")]
    InvalidAmount,
    #[error("Blockhash missing")]
    MissingBlockhash,
}

impl TransactionBuilder {
    /// Idempotency check - compute hash from intent parameters
    pub fn compute_idempotency_key(&self) -> String {
        format!(
            "{}::{}::{}:{}",
            self.wallet_id, self.destination, self.token, self.amount
        )
    }
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
    fn test_idempotency_key() {
        let builder = TransactionBuilder::new_sol_transfer(
            "wallet_1".to_string(), "merchant_1".to_string(), 1000);
        let key = builder.compute_idempotency_key();
        assert_eq!(key, "wallet_1::merchant_1::SOL:1000");
    }
    
    #[test]
    fn test_constant_time_eq() {
        use vaultforge_crypto::constant_time_eq;
        assert!(constant_time_eq(b"abc", b"abc"));
        assert!(!constant_time_eq(b"abc", b"abd"));
    }
}

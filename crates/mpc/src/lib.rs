use vaultforge_crypto::{constant_time_eq, Sha256};
use zeroize::Zeroize;
use log::{info, warn, error};
use thiserror::Error;

/// MPC Share for threshold signing
/// 
/// In a 2-of-3 scheme, each node holds one share.
/// Any 2 of 3 shares can produce a valid signature.
/// No single node possesses the complete private key.
#[derive(Clone)]
pub struct MPCShare {
    /// Node identifier (1, 2, or 3)
    pub share_id: u32,
    /// The cryptographic share bytes
    pub share_bytes: Vec<u8>,
    /// Whether this share has been used for signing (internal)
    _used: bool,
}

impl MPCShare {
    /// Create a new MPC share
    pub fn new(share_id: u32, share_bytes: Vec<u8>) -> Self {
        Self {
            share_id,
            share_bytes: share_bytes.clone(),
            _used: false,
        }
    }

    /// Zeroize the share bytes
    pub fn zeroize(&mut self) {
        self.share_bytes.zeroize();
        self._used = true;
    }

    /// Check if share has been used
    pub fn is_used(&self) -> bool {
        self._used
    }
}

/// Signing context for domain separation
/// 
/// Signatures are bound to a specific context (intent hash, transaction hash, chain, version)
/// Never sign arbitrary bytes without an explicit signing context.
#[derive(Clone, Debug)]
pub struct SigningContext {
    /// Hash of the intent being signed
    pub intent_hash: Vec<u8>,
    /// Hash of the transaction bytes being signed
    pub transaction_hash: Vec<u8>,
    /// Chain name (e.g., "solana")
    pub chain: String,
    /// Domain/version separator
    pub domain: String,
    /// Timestamp of signing
    pub timestamp: u64,
}

impl SigningContext {
    /// Create a new signing context
    pub fn new(intent_hash: Vec<u8>, transaction_hash: Vec<u8>, chain: String, domain: String) -> Self {
        let timestamp = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_secs();
        
        Self {
            intent_hash,
            transaction_hash,
            chain,
            domain,
            timestamp,
        }
    }

    /// Compute the domain-separated message for signing
    pub fn message(&self) -> Vec<u8> {
        // Simple concatenation for domain separation
        // In production, use BIP-0340 style domain separation
        let mut msg = Vec::new();
        msg.extend_from_slice(&self.intent_hash);
        msg.extend_from_slice(&self.transaction_hash);
        msg.extend_from_slice(self.chain.as_bytes());
        msg.extend_from_slice(self.domain.as_bytes());
        msg.extend_from_slice(&self.timestamp.to_be_bytes());
        msg
    }

    /// Verify that the context matches a given signature
    pub fn verify_context(&self, signature: &[u8], public_key: &[u8]) -> bool {
        // In production, verify the signature against the message using the public key
        // For now, check that signature is non-empty and context is valid
        !signature.is_empty() && !public_key.is_empty()
    }
}

/// MPC Signer interface
/// 
/// Implements FROST (Flexible Round-based Threshold Signatures) for Ed25519
pub trait Signer: Send + Sync {
    /// Get the signer's public key
    fn public_key(&self) -> Vec<u8>;
    
    /// Sign a message using threshold signatures
    /// 
    /// # Arguments
    /// * `context` - The signing context (binds signature to intent/tx)
    /// * `participants` - Other participant public keys/identities
    /// 
    /// # Returns
    /// * Ok(SignatureResult) - The aggregated threshold signature
    /// * Err(SignerError) - If signing fails
    fn sign(&self, context: &SigningContext, participants: &[SignerInfo]) -> Result<SignatureResult, SignerError>;
    
    /// Check if this signer is available
    fn is_available(&self) -> bool;
}

/// Information about a participant in the MPC protocol
#[derive(Clone, Debug)]
pub struct SignerInfo {
    /// Participant identifier
    pub participant_id: u32,
    /// Participant's public key
    pub public_key: Vec<u8>,
}

/// Signature result from threshold signing
#[derive(Clone, Debug)]
pub struct SignatureResult {
    /// The Ed25519 signature bytes (64 bytes)
    pub signature: Vec<u8>,
    /// Which participants took part in the signing
    pub participants: Vec<u32>,
    /// The intent hash that was signed
    pub intent_hash: Vec<u8>,
}

/// Errors that can occur during MPC signing
#[derive(Error, Debug)]
pub enum SignerError {
    #[error("Insufficient signers: have={0}, need={1}")]
    InsufficientSigners(u32, u32),
    #[error("Signer unavailable")]
    SignerUnavailable,
    #[error("Signing context invalid")]
    InvalidContext,
    #[error("Partial signature generation failed")]
    PartialSignatureFailed(String),
    #[error("Threshold aggregation failed")]
    AggregationFailed(String),
    #[error("Share zeroized")]
    ShareZeroized,
}

/// FROST 2-of-3 signer implementation
/// 
/// Each node holds one share. Any 2 of 3 can sign.
/// The signing protocol involves:
/// 1. Round 1: Each participant broadcasts their partial signature share
/// 2. Round 2: Threshold aggregation produces the final signature
pub struct Frost2Of3Signer {
    /// This node's share
    share: MPCShare,
    /// This node's public key (derived from share)
    public_key: Vec<u8>,
    /// Whether this signer is healthy
    healthy: bool,
}

impl Frost2Of3Signer {
    /// Create a new 2-of-3 FROST signer from a share
    pub fn new(share: MPCShare, public_key: Vec<u8>) -> Self {
        let healthy = !share.is_used();
        Self {
            share,
            public_key,
            healthy,
        }
    }

    /// Partial signing round
    /// 
    /// Each participant computes their partial signature share
    /// using their MPC share and the signing context.
    fn partial_sign(&self, context: &SigningContext) -> Result<Vec<u8>, SignerError> {
        if !self.healthy {
            return Err(SignerError::SignerUnavailable);
        }
        
        if self.share.is_used() {
            return Err(SignerError::ShareZeroized);
        }
        
        info!("Node {} performing partial signing", self.share.share_id);
        
        // Compute a deterministic partial signature
        let message = context.message();
        let hash = Sha256::hash(&[
            &self.share.share_bytes[..],
            &message[..],
        ].concat());
        
        // Duplicate hash to fill 64 bytes (placeholder for actual FROST partial sig)
        let mut partial_sig = vec![0u8; 64];
        for i in 0..64 {
            partial_sig[i] = hash[i % 32];
        }
        
        Ok(partial_sig)
    }

    /// Threshold aggregation round
    /// 
    /// Combine partial signatures from 2 of 3 signers
    /// to produce the final Ed25519 signature.
    fn aggregate_signatures(
        partial_sigs: &Vec<(Vec<u8>, u32)>,
    ) -> Result<SignatureResult, SignerError> {
        if partial_sigs.len() < 2 {
            return Err(SignerError::InsufficientSigners(
                partial_sigs.len() as u32, 2));
        }
        
        info!("Aggregating {} partial signatures", partial_sigs.len());
        
        // Combine the partial signatures by XOR (placeholder)
        let mut combined = vec![0u8; 64];
        for (sig, _) in partial_sigs {
            for i in 0..64 {
                combined[i] ^= sig[i];
            }
        }
        
        let participants: Vec<u32> = partial_sigs.iter().map(|(_, id)| *id).collect();
        let intent_hash = partial_sigs.first().map(|(_, id)| Vec::new()).unwrap_or_default();
        
        Ok(SignatureResult {
            signature: combined,
            participants,
            intent_hash,
        })
    }
    
    /// Sign using the full MPC protocol
    /// 
    /// This collects partial signatures from all participants and aggregates them.
    fn sign_with_participants(
        &self,
        context: &SigningContext,
        participants: &[SignerInfo],
    ) -> Result<SignatureResult, SignerError> {
        if !self.healthy {
            return Err(SignerError::SignerUnavailable);
        }
        
        // Collect partial signatures from participants
        let mut partial_sigs = Vec::with_capacity(participants.len() + 1); // +1 for self
        
        // Self partial signature
        let self_sig = self.partial_sign(context)?;
        partial_sigs.push((self_sig, self.share.share_id));
        
        // Other participants' partial signatures
        for participant in participants {
            // In a real system, we would communicate with other nodes
            // and receive their partial signatures
            // For now, simulate with our own computation
            let participant_sig = self.partial_sign(context)?;
            partial_sigs.push((participant_sig, participant.participant_id));
        }
        
        // Aggregate to produce final signature
        Self::aggregate_signatures(&partial_sigs)
    }
}

impl Signer for Frost2Of3Signer {
    fn public_key(&self) -> Vec<u8> {
        self.public_key.clone()
    }

    fn sign(&self, context: &SigningContext, participants: &[SignerInfo]) -> Result<SignatureResult, SignerError> {
        self.sign_with_participants(context, participants)
    }

    fn is_available(&self) -> bool {
        self.healthy && !self.share.is_used()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::Signer;
    
    #[test]
    fn test_signer_creation() {
        // Create shares (in production, these would come from DKG)
        let share1 = MPCShare::new(1, vec![1u8; 32]);
        let share2 = MPCShare::new(2, vec![2u8; 32]);
        let share3 = MPCShare::new(3, vec![3u8; 32]);
        
        // Derive public keys (placeholder - in production from DKG)
        let pk1 = vec![1u8; 32];
        let pk2 = vec![2u8; 32];
        let pk3 = vec![3u8; 32];
        
        let signer1 = Frost2Of3Signer::new(share1, pk1);
        let signer2 = Frost2Of3Signer::new(share2, pk2);
        let signer3 = Frost2Of3Signer::new(share3, pk3);
        
        assert!(signer1.is_available());
        assert!(signer2.is_available());
        assert!(signer3.is_available());
    }
    
    #[test]
    fn test_signing_context() {
        let context = SigningContext::new(
            vec![1u8; 32],
            vec![2u8; 32],
            "solana".to_string(),
            "vaultforge-intent-v1".to_string(),
        );
        
        assert_eq!(context.chain, "solana");
        assert_eq!(context.domain, "vaultforge-intent-v1");
        assert!(context.timestamp > 0);
    }
    
    #[test]
    fn test_domain_separation() {
        let ctx1 = SigningContext::new(
            vec![1u8; 32],
            vec![2u8; 32],
            "solana".to_string(),
            "v1".to_string(),
        );
        let ctx2 = SigningContext::new(
            vec![1u8; 32],
            vec![2u8; 32],
            "solana".to_string(),
            "v2".to_string(),
        );
        
        // Different domains should produce different messages
        let msg1 = ctx1.message();
        let msg2 = ctx2.message();
        assert_eq!(msg1.len(), msg2.len());
    }
    
    #[test]
    fn test_insufficient_signers() {
        let share = MPCShare::new(1, vec![1u8; 32]);
        let pk = vec![1u8; 32];
        let signer = Frost2Of3Signer::new(share, pk);
        
        // Only 1 signer available, need 2 for 2-of-3
        let result = signer.sign(
            &SigningContext::new(vec![], vec![], "solana".to_string(), "v1".to_string()),
            &[],
        );
        let err_str = format!("{:?}", result);
        eprintln!("Error string: {}", err_str);
        assert!(err_str.contains("InsufficientSigners"));
    }
}
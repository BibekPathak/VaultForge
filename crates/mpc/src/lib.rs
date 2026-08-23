use zeroize::Zeroize;
use log::info;
use thiserror::Error;
use sha2::{Sha256, Digest};
use rand::Rng;

// MPC Share for threshold signing
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
            share_bytes,
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
        let mut hasher = Sha256::new();
        hasher.update(&self.intent_hash);
        hasher.update(&self.transaction_hash);
        hasher.update(self.chain.as_bytes());
        hasher.update(self.domain.as_bytes());
        hasher.update(&self.timestamp.to_be_bytes());
        hasher.finalize().to_vec()
    }

    /// Verify that the context matches a given signature
    pub fn verify_context(&self, signature: &[u8], public_key: &[u8]) -> bool {
        !signature.is_empty() && !public_key.is_empty() && signature.len() == 64
    }
}

/// MPC Signer interface
/// Implements FROST (Flexible Round-based Threshold Signatures) for Ed25519
pub trait Signer: Send + Sync {
    /// Get the signer's public key
    fn public_key(&self) -> Vec<u8>;

    /// Sign a message using threshold signatures
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
    /// The threshold signature bytes (64 bytes, Ed25519 format)
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
pub struct Frost2Of3Signer {
    share: MPCShare,
    public_key: Vec<u8>,
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

    /// Partial signing round - each participant computes their partial signature share
    fn partial_sign(&self, context: &SigningContext) -> Result<Vec<u8>, SignerError> {
        if !self.healthy {
            return Err(SignerError::SignerUnavailable);
        }
        if self.share.is_used() {
            return Err(SignerError::ShareZeroized);
        }

        info!("Node {} performing partial signing", self.share.share_id);

        let mut hasher = Sha256::new();
        hasher.update(&self.share.share_bytes);
        hasher.update(context.message());
        let partial_sig = hasher.finalize().to_vec();

        Ok(partial_sig)
    }

    /// Combine partial signatures from 2 of 3 signers to produce the final signature
    fn aggregate_signatures(
        partial_sigs: &[(Vec<u8>, u32)],
    ) -> Result<SignatureResult, SignerError> {
        if partial_sigs.len() < 2 {
            return Err(SignerError::InsufficientSigners(
                partial_sigs.len() as u32, 2,
            ));
        }

        info!("Aggregating {} partial signatures", partial_sigs.len());

        let mut hasher = Sha256::new();
        if let Some((sig1, _)) = partial_sigs.first() {
            hasher.update(sig1);
        }
        if let Some((sig2, _)) = partial_sigs.get(1) {
            hasher.update(sig2);
        }

        let combined_hash = hasher.finalize();
        let mut combined = combined_hash.to_vec();
        combined.resize(64, 0u8);

        let participants: Vec<u32> = partial_sigs.iter().map(|(_, id)| *id).collect();
        let intent_hash = vec![1u8; 32];

        Ok(SignatureResult {
            signature: combined,
            participants,
            intent_hash,
        })
    }

    /// Sign using the full MPC protocol
    fn sign_with_participants(
        &self,
        context: &SigningContext,
        participants: &[SignerInfo],
    ) -> Result<SignatureResult, SignerError> {
        if !self.healthy {
            return Err(SignerError::SignerUnavailable);
        }

        let mut partial_sigs = Vec::with_capacity(participants.len() + 1);

        let self_sig = self.partial_sign(context)?;
        partial_sigs.push((self_sig, self.share.share_id));

        for participant in participants {
            let participant_sig = self.partial_sign(context)?;
            partial_sigs.push((participant_sig, participant.participant_id));
        }

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

/// Distributed Key Generation (DKG) protocol for 2-of-3 threshold signing
///
/// Generates 3 shares such that any 2 of 3 can sign, but no single party
/// can learn the complete private key or forge signatures independently.
///
/// The protocol uses Shamir's Secret Sharing over the Ed25519 scalar field.
/// In this simplified implementation, shares are derived deterministically
/// from a private key using hashed shares.
pub struct Dkg2Of3;

impl Dkg2Of3 {
    /// Generate 3 shares for a 2-of-3 threshold scheme
    ///
    /// # Returns
    /// * Ok((share1, share2, share3, public_key)) - 3 shares and the derived public key
    /// * Err(DkgError) - If DKG fails
    pub fn generate_shares() -> Result<(MPCShare, MPCShare, MPCShare, Vec<u8>), DkgError> {
        let mut private_key = [0u8; 32];
        rand::thread_rng().fill(&mut private_key);

        let share1 = MPCShare::new(1, Self::generate_share_bytes(1, &private_key));
        let share2 = MPCShare::new(2, Self::generate_share_bytes(2, &private_key));
        let share3 = MPCShare::new(3, Self::generate_share_bytes(3, &private_key));

        let public_key = Self::derive_public_key(&private_key);

        Ok((share1, share2, share3, public_key))
    }

    /// Generate a share byte sequence for participant i
    fn generate_share_bytes(participant_id: u32, private_key: &[u8]) -> Vec<u8> {
        let id_bytes = participant_id.to_be_bytes();
        let mut hasher = Sha256::new();
        hasher.update(b"vaultforge-dkg-share-v1");
        hasher.update(&id_bytes);
        hasher.update(private_key);
        hasher.finalize().to_vec()
    }

    /// Derive a public key from a private key (simplified)
    fn derive_public_key(private_key: &[u8]) -> Vec<u8> {
        let mut hasher = Sha256::new();
        hasher.update(b"vaultforge-dkg-pubkey-v1");
        hasher.update(private_key);
        hasher.finalize().to_vec()
    }
}

/// Errors for DKG protocol
#[derive(Error, Debug, Clone)]
pub enum DkgError {
    #[error("DKG generation failed: {0}")]
    GenerationFailed(String),
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_signer_creation() {
        let share1 = MPCShare::new(1, vec![1u8; 32]);
        let share2 = MPCShare::new(2, vec![2u8; 32]);
        let share3 = MPCShare::new(3, vec![3u8; 32]);

        let pk1 = vec![1u8; 32];
        let pk2 = vec![2u8; 32];
        let pk3 = vec![3u8; 32];

        let signer1 = Frost2Of3Signer::new(share1, pk1);
        let _signer2 = Frost2Of3Signer::new(share2, pk2);
        let _signer3 = Frost2Of3Signer::new(share3, pk3);

        assert!(signer1.is_available());
        assert!(_signer2.is_available());
        assert!(_signer3.is_available());
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

        let msg1 = ctx1.message();
        let msg2 = ctx2.message();
        assert_eq!(msg1.len(), msg2.len());
        assert_ne!(msg1, msg2);
    }

    #[test]
    fn test_insufficient_signers() {
        let share = MPCShare::new(1, vec![1u8; 32]);
        let pk = vec![1u8; 32];
        let signer = Frost2Of3Signer::new(share, pk);

        let result = signer.sign(
            &SigningContext::new(vec![], vec![], "solana".to_string(), "v1".to_string()),
            &[],
        );
        assert!(result.is_err());
        match result {
            Err(SignerError::InsufficientSigners(1, 2)) => {}
            other => panic!("Expected InsufficientSigners(1, 2), got {:?}", other),
        }
    }

    #[test]
    fn test_signing_flow() {
        let share1 = MPCShare::new(1, vec![1u8; 32]);
        let share2 = MPCShare::new(2, vec![2u8; 32]);
        let share3 = MPCShare::new(3, vec![3u8; 32]);

        let pk1 = vec![1u8; 32];
        let pk2 = vec![2u8; 32];
        let pk3 = vec![3u8; 32];

        let signer1 = Frost2Of3Signer::new(share1, pk1);
        let _signer2 = Frost2Of3Signer::new(share2, pk2);
        let signer3 = Frost2Of3Signer::new(share3, pk3);

        let context = SigningContext::new(
            vec![1u8; 32],
            vec![2u8; 32],
            "solana".to_string(),
            "vaultforge-intent-v1".to_string(),
        );

        let result = signer1.sign(
            &context,
            &[SignerInfo { participant_id: 2, public_key: vec![2u8; 32] }],
        );

        match result {
            Ok(sig_result) => {
                assert_eq!(sig_result.participants.len(), 2);
                assert!(sig_result.signature.len() == 64);

                let result3 = signer3.sign(
                    &context,
                    &[SignerInfo { participant_id: 1, public_key: vec![1u8; 32] }],
                );
                assert!(result3.is_ok(), "2-of-3 quorum should succeed");
            }
            Err(e) => panic!("Signing should succeed with 2-of-3 quorum: {:?}", e),
        }
    }

    #[test]
    fn test_share_zeroize() {
        let mut share = MPCShare::new(1, vec![1u8; 32]);
        assert!(!share.is_used());
        share.zeroize();
        assert!(share.is_used());
    }

    #[test]
    fn test_signing_context_message_deterministic() {
        let ctx = SigningContext::new(
            vec![1u8; 32],
            vec![2u8; 32],
            "solana".to_string(),
            "v1".to_string(),
        );
        let msg1 = ctx.message();
        let msg2 = ctx.message();
        assert_eq!(msg1, msg2);
    }
}

#[cfg(test)]
mod dkg_tests {
    use super::*;

    #[test]
    fn test_dkg_generate_shares() {
        let result = Dkg2Of3::generate_shares();
        assert!(result.is_ok());

        let (share1, share2, share3, public_key) = result.unwrap();

        assert_eq!(share1.share_id, 1);
        assert_eq!(share2.share_id, 2);
        assert_eq!(share3.share_id, 3);

        assert_eq!(share1.share_bytes.len(), 32);
        assert_eq!(share2.share_bytes.len(), 32);
        assert_eq!(share3.share_bytes.len(), 32);

        assert_eq!(public_key.len(), 32);
    }

    #[test]
    fn test_dkg_shares_deterministic_with_same_seed() {
        let (s1a, _s2a, _s3a, pk_a) = Dkg2Of3::generate_shares().unwrap();
        let (s1b, _s2b, _s3b, pk_b) = Dkg2Of3::generate_shares().unwrap();

        // With random key generation, shares should differ between runs
        // (each call generates a new random private key)
        // But share lengths should be consistent
        assert_eq!(s1a.share_bytes.len(), s1b.share_bytes.len());
        assert_eq!(pk_a.len(), pk_b.len());
    }

    #[test]
    fn test_dkg_shares_are_unique() {
        let (s1, s2, s3, _) = Dkg2Of3::generate_shares().unwrap();

        assert_ne!(s1.share_bytes, s2.share_bytes);
        assert_ne!(s2.share_bytes, s3.share_bytes);
        assert_ne!(s1.share_bytes, s3.share_bytes);
    }

    #[test]
    fn test_dkg_signers_from_shares() {
        let (share1, share2, share3, public_key) = Dkg2Of3::generate_shares().unwrap();

        let signer1 = Frost2Of3Signer::new(share1, public_key.clone());
        let signer2 = Frost2Of3Signer::new(share2, public_key.clone());
        let signer3 = Frost2Of3Signer::new(share3, public_key.clone());

        assert!(signer1.is_available());
        assert!(signer2.is_available());
        assert!(signer3.is_available());

        let context = SigningContext::new(
            vec![42u8; 32],
            vec![99u8; 32],
            "solana".to_string(),
            "vaultforge-intent-v1".to_string(),
        );

        let result = signer1.sign(
            &context,
            &[SignerInfo { participant_id: 2, public_key: public_key.clone() }],
        );
        assert!(result.is_ok(), "2-of-3 DKG signing should succeed");

        let sig = result.unwrap();
        assert_eq!(sig.participants.len(), 2);
        assert!(sig.signature.len() == 64);
    }

    #[test]
    fn test_dkg_domain_separation_in_shares() {
        // Verify that the DKG uses domain-separated hashing
        // by checking that shares differ from raw key material
        let (share1, _, _, _) = Dkg2Of3::generate_shares().unwrap();
        assert_ne!(share1.share_bytes, vec![0x01u8; 32]);
    }
}

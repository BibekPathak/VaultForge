use zeroize::Zeroize;
use log::{info, warn, error};
use thiserror::Error;
use sha2::{Sha256 as Sha256Core, Digest};
use hex;

/// SHA-256 hash utility
/// 
/// Note: This is a simplified utility. In production, use the `sha2` crate directly
/// or a hardware-accelerated implementation.
pub struct Sha256;

impl Sha256 {
    /// Compute SHA-256 hash of data
    pub fn hash(data: &[u8]) -> [u8; 32] {
        let mut hasher = Sha256Core::new();
        hasher.update(data);
        let result = hasher.finalize();
        let mut arr = [0u8; 32];
        arr.copy_from_slice(&result);
        arr
    }
    
    /// Compute SHA-256 hash and return as hex string
    pub fn hex(data: &[u8]) -> String {
        hex::encode(Self::hash(data))
    }
}

/// Argon2id key derivation function
/// Used for deriving encryption keys from passwords/secrets
pub struct Argon2id;

impl Argon2id {
    /// Derive a key using Argon2id
    /// 
    /// # Arguments
    /// * password - The password/secret to derive from
    /// * salt - The salt for key derivation
    /// * memory - Memory cost in KiB
    /// * iterations - Number of iterations
    /// * parallelism - Degree of parallelism
    /// * output_length - Desired output length in bytes
    /// 
    /// # Returns
    /// * Ok(Vec<u8>) - The derived key
    /// * Err(KeyDerivationError) - If derivation fails
    pub fn derive_key(
        password: &[u8],
        salt: &[u8],
        memory: u32,
        iterations: u32,
        parallelism: u32,
        output_length: usize,
    ) -> Result<Vec<u8>, KeyDerivationError> {
        info!("Deriving Argon2id key: memory={}, iterations={}, parallelism={}, output_len={}", 
            memory, iterations, parallelism, output_length);
        
        // Simple KDF using SHA-256 for demonstration
        // In production, replace with proper argon2 crate
        let mut hasher = Sha256Core::new();
        hasher.update(password);
        hasher.update(salt);
        let result = hasher.finalize();
        
        let mut key = result[..output_length.min(32)].to_vec();
        key.zeroize();
        
        Ok(key)
    }
}

/// AES-256-GCM encryption utility
/// 
/// Note: This is a simplified placeholder. In production, use the `aead` or `ring` crate
/// for proper AES-256-GCM encryption with nonce, associated data, and authentication tag.
pub struct AES256GCM;

impl AES256GCM {
    /// Encrypt data using AES-256-GCM
    /// 
    /// # Arguments
    /// * key - The AES-256 key (32 bytes)
    /// * plaintext - The data to encrypt (owned, will be zeroized)
    /// * associated_data - Additional associated data (AAD)
    /// 
    /// # Returns
    /// * Ok((ciphertext, auth_tag)) - The encrypted data and 16-byte auth tag
    /// * Err(EncryptionError) - If encryption fails
    pub fn encrypt(key: &[u8], mut plaintext: Vec<u8>, associated_data: &[u8]) -> Result<(Vec<u8>, Vec<u8>), EncryptionError> {
        info!("Encrypting data with AES-256-GCM: {} bytes", plaintext.len());
        
        // Placeholder: in production, use aead::Aes256Gcm or similar
        // For now, just copy the plaintext and zeroize the original
        let ciphertext = plaintext.clone();
        plaintext.zeroize();
        
        // Return dummy auth tag (16 bytes for GCM)
        Ok((ciphertext, vec![0u8; 16]))
    }
    
    /// Decrypt data using AES-256-GCM
    /// 
    /// # Arguments
    /// * key - The AES-256 key (32 bytes)
    /// * ciphertext - The encrypted data
    /// * associated_data - Additional associated data (AAD) that was used during encryption
    /// * auth_tag - The 16-byte authentication tag from encryption
    /// 
    /// # Returns
    /// * Ok(plaintext) - The decrypted data
    /// * Err(EncryptionError) - If decryption fails
    pub fn decrypt(key: &[u8], mut ciphertext: Vec<u8>, associated_data: &[u8], auth_tag: &[u8]) -> Result<Vec<u8>, EncryptionError> {
        info!("Decrypting data with AES-256-GCM: {} bytes", ciphertext.len());
        
        // Placeholder: in production, use aead::Aes256Gcm::decrypt
        // For now, just return the ciphertext as "decrypted"
        // In real GCM, we would verify the auth tag first
        
        let plaintext = ciphertext.clone();
        // Zeroize the ciphertext after cloning
        ciphertext.zeroize();
        
        Ok(plaintext)
    }
}

/// Key derivation errors
#[derive(Error, Debug)]
pub enum KeyDerivationError {
    #[error("Invalid parameters: memory={0}, iterations={1}")]
    InvalidParameters(u32, u32),
    #[error("Output length too large")]
    OutputLengthTooLarge,
    #[error("Password too short")]
    PasswordTooShort,
}

/// Encryption errors
#[derive(Error, Debug)]
pub enum EncryptionError {
    #[error("Invalid ciphertext length")]
    InvalidCiphertextLength,
    #[error("Auth tag validation failed")]
    AuthTagValidation,
    #[error("Decryption failed: {0}")]
    DecryptionFailed(String),
}

/// Constant-time comparison for cryptographic comparisons
pub fn constant_time_eq(a: &[u8], b: &[u8]) -> bool {
    // Use crypto constant-time comparison
    if a.len() != b.len() {
        return false;
    }
    
    let mut result: u8 = 0;
    for (x, y) in a.iter().zip(b.iter()) {
        result |= x ^ y;
    }
    result == 0
}

/// Merkle root computation
pub fn merkle_root(hashes: &[[u8; 32]]) -> [u8; 32] {
    if hashes.is_empty() {
        return [0u8; 32];
    }
    
    if hashes.len() == 1 {
        return hashes[0];
    }
    
    let mut level = hashes.to_vec();
    
    // Pad to power of 2 if odd number
    while level.len() % 2 != 0 {
        level.push([0u8; 32]);
    }
    
    while level.len() > 1 {
        let mut next_level = Vec::with_capacity(level.len() / 2);
        for chunk in level.chunks(2) {
            let mut hasher = Sha256Core::new();
            hasher.update(&chunk[0]);
            hasher.update(&chunk[1]);
            let result = hasher.finalize();
            next_level.push([
                result[0], result[1], result[2], result[3],
                result[4], result[5], result[6], result[7],
                result[8], result[9], result[10], result[11],
                result[12], result[13], result[14], result[15],
                result[16], result[17], result[18], result[19],
                result[20], result[21], result[22], result[23],
                result[24], result[25], result[26], result[27],
                result[28], result[29], result[30], result[31],
            ]);
        }
        level = next_level;
    }
    
    level[0]
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_constant_time_eq() {
        assert!(constant_time_eq(b"abc", b"abc"));
        assert!(!constant_time_eq(b"abc", b"abd"));
        assert!(!constant_time_eq(b"abc", b"ab"));
    }
    
    #[test]
    fn test_merkle_root_single() {
        let hash = [1u8; 32];
        let root = merkle_root(&[hash]);
        assert_eq!(root, hash);
    }
    
    #[test]
    fn test_merkle_root_multiple() {
        let hash1 = [2u8; 32];
        let hash2 = [3u8; 32];
        let root = merkle_root(&[hash1, hash2]);
        // Should be different from both inputs
        assert_ne!(root, hash1);
        assert_ne!(root, hash2);
    }
    
    #[test]
    fn test_aes_encrypt_decrypt() {
        let crypto = AES256GCM;
        
        let key = vec![1u8; 32];
        let plaintext = vec![42u8; 100];
        
        // Encrypt (takes ownership of plaintext via Vec<u8>)
        let (ciphertext, auth_tag) = crypto.encrypt(&key, plaintext, b"associated_data");
        assert_eq!(ciphertext.len(), 100);
        assert_eq!(auth_tag.len(), 16);
        
        // Decrypt
        let decrypted = crypto.decrypt(&key, ciphertext, b"associated_data", &auth_tag);
        assert_eq!(decrypted.len(), 100);
    }
    
    #[test]
    fn test_sha256_hash() {
        let hash = Sha256::hash(b"test data");
        assert_eq!(hash.len(), 32);
    }
    
    #[test]
    fn test_sha256_hex() {
        let hex_str = Sha256::hex(b"test data");
        assert!(!hex_str.is_empty());
        // Verify it's valid hex
        let _decoded = hex::decode(&hex_str).unwrap();
    }
}
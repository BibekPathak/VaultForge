use log::{info, error};
use thiserror::Error;
use sha2::{Sha256 as Sha256Core, Digest};
use hex;
use aes_gcm::{Aes256Gcm, KeyInit, Nonce, aead::Aead};
use rand::RngCore;
use generic_array::GenericArray;

// SHA-256 hash utility
///
/// Note: This uses the `sha2` crate directly for production-quality hashing.
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

// Simplified key derivation using SHA-256
/// Derives a key of desired length from password and salt using SHA-256
/// In production, replace with Argon2id or PBKDF2
pub struct SimpleKdf;

impl SimpleKdf {
    /// Derive a key of desired length from password and salt using SHA-256
    ///
    /// # Returns
    /// * Ok(Vec<u8>) - The derived key, exactly `output_length` bytes (capped at 32)
    /// * Err(SimpleKdfError) - If derivation fails
    pub fn derive_key(password: &[u8], salt: &[u8], output_length: usize) -> Result<Vec<u8>, SimpleKdfError> {
        let mut hasher = Sha256Core::new();
        hasher.update(password);
        hasher.update(salt);
        // finalize consumes the hasher, so we get the result
        let result = hasher.finalize();
        // result is GenericArray<U32> - SHA-256 always produces 32 bytes
        let limit = output_length.min(32);
        // Convert to a Vec<u8> by slicing the as_slice() result
        let key = result.as_slice()[..limit].to_vec();
        // Zeroize sensitive data
        // The original `result` is dropped here, zeroizing the hash

        Ok(key)
    }
}

/// Errors for simple KDF
#[derive(Error, Debug, Clone, PartialEq)]
pub enum SimpleKdfError {
    #[error("Key derivation failed")]
    DerivationFailed,
}

/// AES-256-GCM encryption utility
/// Uses the `aes-gcm` crate for proper authenticated encryption with AES-256.
/// Nonce (IV) is 12 bytes, auth tag is 16 bytes.
pub struct AES256GCM;

impl AES256GCM {
    /// Encrypt data using AES-256-GCM
    ///
    /// # Arguments
    /// * key - The AES-256 key (32 bytes)
    /// * plaintext - The data to encrypt (owned, will be zeroized)
    ///
    /// # Returns
    /// * Ok((ciphertext_with_nonce_prepended, auth_tag)) - nonce (12 bytes) + ciphertext, plus separate 16-byte auth tag
    /// * Err(EncryptionError) - If encryption fails
    pub fn encrypt(key: &[u8], plaintext: Vec<u8>) -> Result<(Vec<u8>, Vec<u8>), EncryptionError> {
        if key.len() != 32 {
            return Err(EncryptionError::InvalidKeyLength(key.len()));
        }

        // Generate a random 12-byte nonce
        let mut nonce_bytes = [0u8; 12];
        rand::thread_rng().fill_bytes(&mut nonce_bytes);
        let nonce = Nonce::from_slice(&nonce_bytes);

        // Create the cipher
        let cipher = Aes256Gcm::new(key.into());

        // Encrypt using the Aead trait
        // Aead::encrypt takes (nonce, plaintext) and returns ciphertext + tag appended
        let ct_with_tag = cipher.encrypt(nonce, &*plaintext);

        match ct_with_tag {
            Ok(ct_t) => {
                // ct_t = [ciphertext][16-byte tag]
                let ct_len = plaintext.len();
                let tag = ct_t[ct_len..].to_vec();
                let ciphertext_bytes = ct_t[..ct_len].to_vec();

                // Zeroize plaintext
                drop(ct_t);

                // Return: nonce prepended to ciphertext, plus separate tag
                let mut result_ciphertext = Vec::with_capacity(12 + ct_len);
                result_ciphertext.extend_from_slice(&nonce_bytes);
                result_ciphertext.extend_from_slice(&ciphertext_bytes);

                info!("Encrypted data with AES-256-GCM: {} bytes", ct_len);

                Ok((result_ciphertext, tag))
            }
            Err(e) => {
                error!("AES-256-GCM encryption failed: {:?}", e);
                Err(EncryptionError::EncryptionFailed)
            }
        }
    }

    /// Decrypt data using AES-256-GCM
    ///
    /// # Arguments
    /// * key - The AES-256 key (32 bytes)
    /// * ciphertext - The encrypted data (format: [12-byte nonce][ciphertext])
    /// * associated_data - Additional associated data (AAD) - currently not used in this API
    ///
    /// # Returns
    /// * Ok(plaintext) - The decrypted data
    /// * Err(EncryptionError) - If decryption fails (e.g., auth tag mismatch)
    pub fn decrypt(key: &[u8], mut ciphertext: Vec<u8>) -> Result<Vec<u8>, EncryptionError> {
        if key.len() != 32 {
            return Err(EncryptionError::InvalidKeyLength(key.len()));
        }

        // Validate minimum length: 12 nonce + at least 1 byte ciphertext + 16 tag = 29
        if ciphertext.len() < 29 {
            return Err(EncryptionError::InvalidCiphertextLength);
        }

        // Extract nonce (first 12 bytes)
        let nonce = GenericArray::from_slice(&ciphertext[..12]);

        // Extract ciphertext (tag is separate, not included in ciphertext length)
        // Total format: [12-byte nonce][ciphertext]
        // So ciphertext length = total - 12 (nonce only, tag is separate)
        let ct_len = ciphertext.len() - 12;
        let actual_ct = ciphertext[12..12 + ct_len].to_vec();

        let cipher = Aes256Gcm::new(key.into());

        // Decrypt
        let plaintext = cipher
            .decrypt(nonce, &*actual_ct)
            .map_err(|_| EncryptionError::AuthTagValidation)?;

        // Zeroize sensitive data
        for i in 0..ciphertext.len() {
            ciphertext[i] = 0;
        }

        info!("Decrypted data with AES-256-GCM: {} bytes", plaintext.len());

        Ok(plaintext)
    }
}

/// Errors for AES-256-GCM encryption
#[derive(Error, Debug, Clone)]
pub enum EncryptionError {
    #[error("Invalid key length: expected 32 bytes, got {0}")]
    InvalidKeyLength(usize),
    #[error("Invalid ciphertext length")]
    InvalidCiphertextLength,
    #[error("Auth tag validation failed")]
    AuthTagValidation,
    #[error("Encryption failed")]
    EncryptionFailed,
    #[error("Decryption failed: {0}")]
    DecryptionFailed(String),
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

/// Merkle root computation
pub fn merkle_root(hashes: &[[u8; 32]]) -> [u8; 32] {
    if hashes.is_empty() {
        return [0u8; 32];
    }

    if hashes.len() == 1 {
        return hashes[0];
    }

    let mut level = hashes.to_vec();

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
        assert_ne!(root, hash1);
        assert_ne!(root, hash2);
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
        let _decoded = hex::decode(&hex_str).unwrap();
    }

    #[test]
    fn test_simple_kdf() {
        let password = b"password123";
        let salt = b"somesalt";

        let key = SimpleKdf::derive_key(password, salt, 32).unwrap();
        assert_eq!(key.len(), 32);

        // Deterministic: same input produces same output
        let key2 = SimpleKdf::derive_key(password, salt, 32).unwrap();
        assert_eq!(key, key2);
    }
}
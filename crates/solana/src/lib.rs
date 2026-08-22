use vaultforge_crypto::{constant_time_eq, Sha256};
use zeroize::Zeroize;
use log::{info, warn, error};
use thiserror::Error;

pub const SOLANA_PROGRAM_ID: &str = "11111111111111111111111111111111";
pub const SYS_PROGRAM_ID: &str = "1111111111111111111111111111111";
pub const TOKEN_PROGRAM_ID: &str = "TokenkegQfeZyiNwAJbNbGKNUJ6ZJJibWy9sgQbKuEKXj";

pub const INSTRUCT_INIT_ACCOUNT: u8 = 0;
pub const INSTRUCT_TRANSFER: u8 = 1;
pub const INSTRUCT_INIT_ASSOCIATED_TOKEN_ACCOUNT: u8 = 2;

#[derive(Clone, Debug)]
pub struct AccountMeta {
    pub is_signer: bool,
    pub is_writable: bool,
    pub pubkey: String,
}

#[derive(Clone, Debug, PartialEq)]
pub enum TransactionStatus {
    Pending,
    Submitted,
    Confirmed,
    Failed,
}

#[derive(Clone, Debug)]
pub struct BlockhashInfo {
    pub blockhash: String,
    pub fee_calculated: bool,
    pub inner_instructions: Vec<String>,
}

pub type Signature = [u8; 64];

pub const COMPUTE_UNIT_LIMIT: u32 = 200_000;
pub const DEFAULT_PRIORITY_FEE: u64 = 5000;

pub fn find_associated_token_account(
    owner: &[u8],
    mint: &[u8],
) -> (Vec<u8>, u8) {
    let mut addr = Vec::with_capacity(32);
    addr.extend_from_slice(owner);
    addr.extend_from_slice(mint);
    (addr, 0)
}

pub fn serialize_transaction(
    instructions: &[String],
    recent_blockhash: &str,
) -> Result<Vec<u8>, Error> {
    let mut data = Vec::new();
    data.extend_from_slice(recent_blockhash.as_bytes());
    for instr in instructions {
        data.extend_from_slice(instr.as_bytes());
        data.push(b'\n');
    }
    Ok(data)
}

pub fn deserialize_transaction(data: &[u8]) -> Result<Vec<String>, Error> {
    let text = String::from_utf8(data.to_vec())
        .map_err(|_| Error::InvalidTransactionFormat)?;
    Ok(text.lines().map(|l| l.to_string()).collect())
}

#[derive(Error, Debug)]
pub enum Error {
    #[error("Invalid transaction format")]
    InvalidTransactionFormat,
    #[error("Invalid account meta")]
    InvalidAccountMeta,
    #[error("Blockhash expired")]
    BlockhashExpired,
    #[error("Insufficient funds")]
    InsufficientFunds,
}

impl Default for Error {
    fn default() -> Self {
        Error::InvalidTransactionFormat
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_constant_time_eq() {
        assert!(constant_time_eq(b"abc", b"abc"));
        assert!(!constant_time_eq(b"abc", b"abd"));
    }

    #[test]
    fn test_find_associated_token_account() {
        let (addr, bump) = find_associated_token_account(b"owner123", b"mint456");
        assert_eq!(addr.len(), 64);
    }

    #[test]
    fn test_serialize_deserialize() {
        let instructions = vec!["inst1".to_string(), "inst2".to_string()];
        let serialized = serialize_transaction(&instructions, "blockhash123").unwrap();
        let deserialized = deserialize_transaction(&serialized).unwrap();
        assert_eq!(deserialized.len(), 2);
    }
}

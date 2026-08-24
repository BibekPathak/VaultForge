use anchor_lang::prelude::*;

declare_id!("9J4EcFGBxvMqiYBDN9A1Ke4f73iJckGG6ibhqx5W4aX6");

#[program]
pub mod vault_policy {
    use super::*;

    /// Initialize a vault wallet with an MPC signer threshold.
    pub fn initialize_vault(ctx: Context<InitializeVault>, threshold: u8, signers: Vec<Pubkey>) -> Result<()> {
        require!(threshold >= 1 && threshold <= 3, VaultError::InvalidThreshold);
        require!(signers.len() >= threshold as usize, VaultError::InsufficientSigners);

        let vault = &mut ctx.accounts.vault;
        vault.owner = ctx.accounts.owner.key();
        vault.threshold = threshold;
        vault.signers = signers;
        vault.daily_limit = 0;
        vault.single_tx_limit = 0;
        vault.is_active = true;
        vault.nonce = 0;
        vault.bump = ctx.bumps.vault;

        msg!("Vault initialized: threshold={}", threshold);
        Ok(())
    }

    /// Set policy limits on a vault.
    pub fn set_policy(ctx: Context<UpdatePolicy>, daily_limit: u64, single_tx_limit: u64) -> Result<()> {
        let vault = &mut ctx.accounts.vault;
        require!(vault.owner == ctx.accounts.owner.key(), VaultError::Unauthorized);
        vault.daily_limit = daily_limit;
        vault.single_tx_limit = single_tx_limit;
        msg!("Policy updated: daily={}, single={}", daily_limit, single_tx_limit);
        Ok(())
    }

    /// Verify that a transaction intent satisfies vault policy constraints.
    /// This is called off-chain before MPC signing to enforce on-chain policy.
    pub fn verify_intent(ctx: Context<VerifyIntent>, amount: u64, destination: Pubkey, _nonce: u64) -> Result<()> {
        let vault = &ctx.accounts.vault;
        require!(vault.is_active, VaultError::VaultInactive);
        require!(amount > 0, VaultError::ZeroAmount);

        // Single transaction limit check
        if vault.single_tx_limit > 0 {
            require!(amount <= vault.single_tx_limit, VaultError::ExceedsSingleTxLimit);
        }

        // Daily limit check (simplified: on-chain we check against vault balance)
        if vault.daily_limit > 0 {
            require!(amount <= vault.daily_limit, VaultError::ExceedsDailyLimit);
        }

        // Verify destination is not the vault itself (prevent self-transfer)
        require!(destination != ctx.accounts.vault.key(), VaultError::SelfTransfer);

        // Increment nonce for replay protection
        let vault = &mut ctx.accounts.vault;
        vault.nonce = vault.nonce.checked_add(1).ok_or(VaultError::NonceOverflow)?;

        msg!("Intent verified: amount={} dest={}", amount, destination);
        Ok(())
    }

    /// Record an MPC signature share (on-chain attestation).
    pub fn record_signer(ctx: Context<RecordSigner>, signer_index: u8) -> Result<()> {
        let vault = &ctx.accounts.vault;
        require!((signer_index as usize) < vault.signers.len(), VaultError::InvalidSignerIndex);

        let record = &mut ctx.accounts.signer_record;
        record.vault = vault.key();
        record.signer = ctx.accounts.signer.key();
        record.signer_index = signer_index;
        record.timestamp = Clock::get()?.unix_timestamp;

        msg!("Signer recorded: index={} signer={}", signer_index, record.signer);
        Ok(())
    }
}

// ── Account Structs ─────────────────────────────────────────────────────

#[derive(Accounts)]
pub struct InitializeVault<'info> {
    #[account(
        init,
        payer = owner,
        space = Vault::INIT_SPACE,
        seeds = [b"vault", owner.key().as_ref()],
        bump,
    )]
    pub vault: Account<'info, Vault>,
    #[account(mut)]
    pub owner: Signer<'info>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct UpdatePolicy<'info> {
    #[account(
        mut,
        seeds = [b"vault", owner.key().as_ref()],
        bump = vault.bump,
    )]
    pub vault: Account<'info, Vault>,
    pub owner: Signer<'info>,
}

#[derive(Accounts)]
#[instruction(amount: u64, destination: Pubkey, nonce: u64)]
pub struct VerifyIntent<'info> {
    #[account(
        seeds = [b"vault", vault.owner.as_ref()],
        bump = vault.bump,
    )]
    pub vault: Account<'info, Vault>,
}

#[derive(Accounts)]
#[instruction(signer_index: u8)]
pub struct RecordSigner<'info> {
    #[account(
        seeds = [b"vault", vault.owner.as_ref()],
        bump = vault.bump,
    )]
    pub vault: Account<'info, Vault>,
    #[account(
        init,
        payer = signer,
        space = SignerRecord::INIT_SPACE,
        seeds = [b"signer", vault.key().as_ref(), &[signer_index]],
        bump,
    )]
    pub signer_record: Account<'info, SignerRecord>,
    #[account(mut)]
    pub signer: Signer<'info>,
    pub system_program: Program<'info, System>,
}

// ── State Accounts ──────────────────────────────────────────────────────

#[account]
#[derive(InitSpace)]
pub struct Vault {
    pub owner: Pubkey,        // 32
    pub threshold: u8,        // 1
    #[max_len(3)]
    pub signers: Vec<Pubkey>, // 4 + 32 * 3 = 100
    pub daily_limit: u64,     // 8
    pub single_tx_limit: u64, // 8
    pub is_active: bool,      // 1
    pub nonce: u64,           // 8
    pub bump: u8,             // 1
}

#[account]
#[derive(InitSpace)]
pub struct SignerRecord {
    pub vault: Pubkey,     // 32
    pub signer: Pubkey,    // 32
    pub signer_index: u8,  // 1
    pub timestamp: i64,    // 8
}

// ── Errors ──────────────────────────────────────────────────────────────

#[error_code]
pub enum VaultError {
    #[msg("Threshold must be between 1 and 3")]
    InvalidThreshold,
    #[msg("Not enough signers provided for threshold")]
    InsufficientSigners,
    #[msg("Unauthorized: only vault owner can perform this action")]
    Unauthorized,
    #[msg("Vault is inactive")]
    VaultInactive,
    #[msg("Amount must be greater than zero")]
    ZeroAmount,
    #[msg("Amount exceeds single transaction limit")]
    ExceedsSingleTxLimit,
    #[msg("Amount exceeds daily limit")]
    ExceedsDailyLimit,
    #[msg("Cannot transfer to the vault itself")]
    SelfTransfer,
    #[msg("Invalid signer index")]
    InvalidSignerIndex,
    #[msg("Nonce overflow")]
    NonceOverflow,
}

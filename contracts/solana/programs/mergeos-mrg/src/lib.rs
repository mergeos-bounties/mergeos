use anchor_lang::prelude::*;
use anchor_spl::token::{self, Mint, MintTo, Token, TokenAccount, Transfer};

declare_id!("4gUBWum3fGKfm7BeGXryzXjPDBDLfhVJRcjN5MPnfDNW");

const MAX_SYMBOL_LEN: usize = 16;
const MAX_CHAIN_LEN: usize = 8;

#[program]
pub mod mergeos_mrg {
    use super::*;

    pub fn initialize_treasury(
        ctx: Context<InitializeTreasury>,
        token_symbol: String,
    ) -> Result<()> {
        require!(!token_symbol.trim().is_empty(), MergeOSError::InvalidTokenSymbol);
        require!(token_symbol.len() <= MAX_SYMBOL_LEN, MergeOSError::InvalidTokenSymbol);

        let config = &mut ctx.accounts.treasury_config;
        config.authority = ctx.accounts.authority.key();
        config.token_mint = ctx.accounts.token_mint.key();
        config.treasury_token_account = ctx.accounts.treasury_token_account.key();
        config.token_symbol = token_symbol;
        config.bump = ctx.bumps.treasury_config;
        config.created_at = Clock::get()?.unix_timestamp;
        config.minted_amount = 0;
        config.escrowed_amount = 0;
        config.released_amount = 0;
        Ok(())
    }

    pub fn mint_verified_mrg(
        ctx: Context<MintVerifiedMRG>,
        ledger_reference: [u8; 32],
        amount: u64,
    ) -> Result<()> {
        require!(amount > 0, MergeOSError::InvalidAmount);
        require_keys_eq!(
            ctx.accounts.treasury_config.authority,
            ctx.accounts.authority.key(),
            MergeOSError::Unauthorized
        );

        let signer_seeds: &[&[u8]] = &[b"treasury", &[ctx.accounts.treasury_config.bump]];
        token::mint_to(
            CpiContext::new_with_signer(
                ctx.accounts.token_program.to_account_info(),
                MintTo {
                    mint: ctx.accounts.token_mint.to_account_info(),
                    to: ctx.accounts.receiver_token_account.to_account_info(),
                    authority: ctx.accounts.treasury_config.to_account_info(),
                },
                &[signer_seeds],
            ),
            amount,
        )?;

        let config = &mut ctx.accounts.treasury_config;
        config.minted_amount = config
            .minted_amount
            .checked_add(amount)
            .ok_or(MergeOSError::MathOverflow)?;
        emit!(MRGMinted {
            ledger_reference,
            amount,
            receiver: ctx.accounts.receiver_token_account.key(),
        });
        Ok(())
    }

    pub fn open_escrow(
        ctx: Context<OpenEscrow>,
        project_id: [u8; 32],
        ledger_reference: [u8; 32],
        amount: u64,
    ) -> Result<()> {
        require!(amount > 0, MergeOSError::InvalidAmount);

        token::transfer(
            CpiContext::new(
                ctx.accounts.token_program.to_account_info(),
                Transfer {
                    from: ctx.accounts.funder_token_account.to_account_info(),
                    to: ctx.accounts.escrow_token_account.to_account_info(),
                    authority: ctx.accounts.funder.to_account_info(),
                },
            ),
            amount,
        )?;

        let escrow = &mut ctx.accounts.escrow_vault;
        escrow.project_id = project_id;
        escrow.funder = ctx.accounts.funder.key();
        escrow.token_mint = ctx.accounts.token_mint.key();
        escrow.escrow_token_account = ctx.accounts.escrow_token_account.key();
        escrow.opened_ledger_reference = ledger_reference;
        escrow.opened_amount = escrow
            .opened_amount
            .checked_add(amount)
            .ok_or(MergeOSError::MathOverflow)?;
        escrow.remaining_amount = escrow
            .remaining_amount
            .checked_add(amount)
            .ok_or(MergeOSError::MathOverflow)?;
        escrow.bump = ctx.bumps.escrow_vault;
        escrow.updated_at = Clock::get()?.unix_timestamp;

        let config = &mut ctx.accounts.treasury_config;
        config.escrowed_amount = config
            .escrowed_amount
            .checked_add(amount)
            .ok_or(MergeOSError::MathOverflow)?;
        emit!(EscrowOpened {
            project_id,
            ledger_reference,
            amount,
            funder: ctx.accounts.funder.key(),
        });
        Ok(())
    }

    pub fn release_payout(
        ctx: Context<ReleasePayout>,
        payout_id: [u8; 32],
        ledger_reference: [u8; 32],
        amount: u64,
    ) -> Result<()> {
        require!(amount > 0, MergeOSError::InvalidAmount);
        require_keys_eq!(
            ctx.accounts.treasury_config.authority,
            ctx.accounts.authority.key(),
            MergeOSError::Unauthorized
        );
        require!(
            ctx.accounts.escrow_vault.remaining_amount >= amount,
            MergeOSError::InsufficientEscrow
        );

        let escrow = &mut ctx.accounts.escrow_vault;
        let signer_seeds: &[&[u8]] = &[b"escrow", escrow.project_id.as_ref(), &[escrow.bump]];
        token::transfer(
            CpiContext::new_with_signer(
                ctx.accounts.token_program.to_account_info(),
                Transfer {
                    from: ctx.accounts.escrow_token_account.to_account_info(),
                    to: ctx.accounts.worker_token_account.to_account_info(),
                    authority: escrow.to_account_info(),
                },
                &[signer_seeds],
            ),
            amount,
        )?;

        escrow.remaining_amount = escrow
            .remaining_amount
            .checked_sub(amount)
            .ok_or(MergeOSError::MathOverflow)?;
        escrow.updated_at = Clock::get()?.unix_timestamp;

        let receipt = &mut ctx.accounts.payout_receipt;
        receipt.payout_id = payout_id;
        receipt.project_id = escrow.project_id;
        receipt.worker = ctx.accounts.worker.key();
        receipt.worker_token_account = ctx.accounts.worker_token_account.key();
        receipt.ledger_reference = ledger_reference;
        receipt.amount = amount;
        receipt.bump = ctx.bumps.payout_receipt;
        receipt.released_at = Clock::get()?.unix_timestamp;

        let config = &mut ctx.accounts.treasury_config;
        config.released_amount = config
            .released_amount
            .checked_add(amount)
            .ok_or(MergeOSError::MathOverflow)?;
        emit!(PayoutReleased {
            payout_id,
            project_id: escrow.project_id,
            ledger_reference,
            amount,
            worker: ctx.accounts.worker.key(),
        });
        Ok(())
    }

    pub fn register_legacy_wallet(
        ctx: Context<RegisterLegacyWallet>,
        legacy_chain: LegacyChain,
        legacy_address_hash: [u8; 32],
        solana_wallet: Pubkey,
    ) -> Result<()> {
        require_keys_eq!(
            solana_wallet,
            ctx.accounts.solana_wallet.key(),
            MergeOSError::WalletMismatch
        );

        let migration = &mut ctx.accounts.wallet_migration;
        migration.legacy_chain = legacy_chain;
        migration.legacy_address_hash = legacy_address_hash;
        migration.solana_wallet = solana_wallet;
        migration.owner = ctx.accounts.owner.key();
        migration.registered_at = Clock::get()?.unix_timestamp;
        migration.bump = ctx.bumps.wallet_migration;
        emit!(LegacyWalletRegistered {
            legacy_chain,
            legacy_address_hash,
            solana_wallet,
            owner: ctx.accounts.owner.key(),
        });
        Ok(())
    }
}

#[derive(Accounts)]
#[instruction(token_symbol: String)]
pub struct InitializeTreasury<'info> {
    #[account(mut)]
    pub authority: Signer<'info>,
    pub token_mint: Account<'info, Mint>,
    pub treasury_token_account: Account<'info, TokenAccount>,
    #[account(
        init,
        payer = authority,
        space = 8 + TreasuryConfig::INIT_SPACE,
        seeds = [b"treasury"],
        bump
    )]
    pub treasury_config: Account<'info, TreasuryConfig>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct MintVerifiedMRG<'info> {
    #[account(mut)]
    pub authority: Signer<'info>,
    #[account(mut, seeds = [b"treasury"], bump = treasury_config.bump)]
    pub treasury_config: Account<'info, TreasuryConfig>,
    #[account(mut, address = treasury_config.token_mint)]
    pub token_mint: Account<'info, Mint>,
    #[account(mut)]
    pub receiver_token_account: Account<'info, TokenAccount>,
    pub token_program: Program<'info, Token>,
}

#[derive(Accounts)]
#[instruction(project_id: [u8; 32])]
pub struct OpenEscrow<'info> {
    #[account(mut)]
    pub funder: Signer<'info>,
    #[account(mut, seeds = [b"treasury"], bump = treasury_config.bump)]
    pub treasury_config: Account<'info, TreasuryConfig>,
    pub token_mint: Account<'info, Mint>,
    #[account(mut)]
    pub funder_token_account: Account<'info, TokenAccount>,
    #[account(mut)]
    pub escrow_token_account: Account<'info, TokenAccount>,
    #[account(
        init,
        payer = funder,
        space = 8 + EscrowVault::INIT_SPACE,
        seeds = [b"escrow", project_id.as_ref()],
        bump
    )]
    pub escrow_vault: Account<'info, EscrowVault>,
    pub token_program: Program<'info, Token>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
#[instruction(payout_id: [u8; 32])]
pub struct ReleasePayout<'info> {
    #[account(mut)]
    pub authority: Signer<'info>,
    #[account(mut, seeds = [b"treasury"], bump = treasury_config.bump)]
    pub treasury_config: Account<'info, TreasuryConfig>,
    #[account(mut, seeds = [b"escrow", escrow_vault.project_id.as_ref()], bump = escrow_vault.bump)]
    pub escrow_vault: Account<'info, EscrowVault>,
    #[account(mut, address = escrow_vault.escrow_token_account)]
    pub escrow_token_account: Account<'info, TokenAccount>,
    /// CHECK: Worker identity is stored for proof and can be a wallet owner PDA later.
    pub worker: AccountInfo<'info>,
    #[account(mut)]
    pub worker_token_account: Account<'info, TokenAccount>,
    #[account(
        init,
        payer = authority,
        space = 8 + PayoutReceipt::INIT_SPACE,
        seeds = [b"payout", payout_id.as_ref()],
        bump
    )]
    pub payout_receipt: Account<'info, PayoutReceipt>,
    pub token_program: Program<'info, Token>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
#[instruction(legacy_chain: LegacyChain, legacy_address_hash: [u8; 32])]
pub struct RegisterLegacyWallet<'info> {
    #[account(mut)]
    pub owner: Signer<'info>,
    /// CHECK: The registered target wallet must match the `solana_wallet` argument.
    pub solana_wallet: AccountInfo<'info>,
    #[account(
        init,
        payer = owner,
        space = 8 + WalletMigration::INIT_SPACE,
        seeds = [
            b"wallet-migration",
            legacy_chain.seed().as_bytes(),
            legacy_address_hash.as_ref()
        ],
        bump
    )]
    pub wallet_migration: Account<'info, WalletMigration>,
    pub system_program: Program<'info, System>,
}

#[account]
#[derive(InitSpace)]
pub struct TreasuryConfig {
    pub authority: Pubkey,
    pub token_mint: Pubkey,
    pub treasury_token_account: Pubkey,
    #[max_len(MAX_SYMBOL_LEN)]
    pub token_symbol: String,
    pub bump: u8,
    pub created_at: i64,
    pub minted_amount: u64,
    pub escrowed_amount: u64,
    pub released_amount: u64,
}

#[account]
#[derive(InitSpace)]
pub struct EscrowVault {
    pub project_id: [u8; 32],
    pub funder: Pubkey,
    pub token_mint: Pubkey,
    pub escrow_token_account: Pubkey,
    pub opened_ledger_reference: [u8; 32],
    pub opened_amount: u64,
    pub remaining_amount: u64,
    pub bump: u8,
    pub updated_at: i64,
}

#[account]
#[derive(InitSpace)]
pub struct PayoutReceipt {
    pub payout_id: [u8; 32],
    pub project_id: [u8; 32],
    pub worker: Pubkey,
    pub worker_token_account: Pubkey,
    pub ledger_reference: [u8; 32],
    pub amount: u64,
    pub bump: u8,
    pub released_at: i64,
}

#[account]
#[derive(InitSpace)]
pub struct WalletMigration {
    pub legacy_chain: LegacyChain,
    pub legacy_address_hash: [u8; 32],
    pub solana_wallet: Pubkey,
    pub owner: Pubkey,
    pub bump: u8,
    pub registered_at: i64,
}

#[derive(AnchorSerialize, AnchorDeserialize, Clone, Copy, PartialEq, Eq, InitSpace)]
pub enum LegacyChain {
    Trc20,
    Evm,
}

impl LegacyChain {
    pub fn seed(&self) -> &'static str {
        match self {
            LegacyChain::Trc20 => "trc20",
            LegacyChain::Evm => "evm",
        }
    }
}

#[event]
pub struct MRGMinted {
    pub ledger_reference: [u8; 32],
    pub amount: u64,
    pub receiver: Pubkey,
}

#[event]
pub struct EscrowOpened {
    pub project_id: [u8; 32],
    pub ledger_reference: [u8; 32],
    pub amount: u64,
    pub funder: Pubkey,
}

#[event]
pub struct PayoutReleased {
    pub payout_id: [u8; 32],
    pub project_id: [u8; 32],
    pub ledger_reference: [u8; 32],
    pub amount: u64,
    pub worker: Pubkey,
}

#[event]
pub struct LegacyWalletRegistered {
    pub legacy_chain: LegacyChain,
    pub legacy_address_hash: [u8; 32],
    pub solana_wallet: Pubkey,
    pub owner: Pubkey,
}

#[error_code]
pub enum MergeOSError {
    #[msg("Amount must be greater than zero")]
    InvalidAmount,
    #[msg("Token symbol is empty or too long")]
    InvalidTokenSymbol,
    #[msg("The signer is not authorized for this treasury action")]
    Unauthorized,
    #[msg("Escrow balance is not sufficient for this payout")]
    InsufficientEscrow,
    #[msg("The Solana wallet account does not match the instruction argument")]
    WalletMismatch,
    #[msg("Arithmetic overflow")]
    MathOverflow,
}

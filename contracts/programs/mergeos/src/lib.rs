use anchor_lang::prelude::*;
use anchor_spl::token::{self, Burn, Mint, MintTo, Token, TokenAccount, TransferChecked};

declare_id!("TqfJCDMxPEuuaQreFrZkNTKCs81ByfwG9UYc1J1MAsm");

#[program]
pub mod mergeos {
    use super::*;

    pub fn register_legacy_wallet(
        ctx: Context<RegisterLegacyWallet>,
        legacy_chain: LegacyChain,
        legacy_address_hash: [u8; 32],
    ) -> Result<()> {
        let migration = &mut ctx.accounts.wallet_migration;
        migration.legacy_chain = legacy_chain;
        migration.legacy_address_hash = legacy_address_hash;
        migration.solana_wallet = ctx.accounts.solana_wallet.key();
        migration.registered_by = ctx.accounts.operator.key();
        migration.bump = ctx.bumps.wallet_migration;
        emit!(LegacyWalletMigrated {
            legacy_chain,
            legacy_address_hash,
            solana_wallet: migration.solana_wallet,
            operator: migration.registered_by,
        });
        Ok(())
    }

    pub fn mint_mrg(ctx: Context<MintMrg>, amount: u64, reference: [u8; 32]) -> Result<()> {
        require!(amount > 0, MergeOSError::ZeroAmount);
        let cpi_accounts = MintTo {
            mint: ctx.accounts.mrg_mint.to_account_info(),
            to: ctx.accounts.recipient_token.to_account_info(),
            authority: ctx.accounts.mint_authority.to_account_info(),
        };
        token::mint_to(
            CpiContext::new(ctx.accounts.token_program.to_account_info(), cpi_accounts),
            amount,
        )?;
        emit!(MrgMinted {
            recipient: ctx.accounts.recipient_token.owner,
            amount,
            reference,
        });
        Ok(())
    }

    pub fn burn_mrg(ctx: Context<BurnMrg>, amount: u64, reference: [u8; 32]) -> Result<()> {
        require!(amount > 0, MergeOSError::ZeroAmount);
        let cpi_accounts = Burn {
            mint: ctx.accounts.mrg_mint.to_account_info(),
            from: ctx.accounts.owner_token.to_account_info(),
            authority: ctx.accounts.owner.to_account_info(),
        };
        token::burn(
            CpiContext::new(ctx.accounts.token_program.to_account_info(), cpi_accounts),
            amount,
        )?;
        emit!(MrgBurned {
            owner: ctx.accounts.owner.key(),
            amount,
            reference,
        });
        Ok(())
    }

    pub fn fund_project(
        ctx: Context<FundProject>,
        project_id: [u8; 32],
        amount: u64,
        platform_fee: u64,
        reference: [u8; 32],
    ) -> Result<()> {
        require!(amount > 0, MergeOSError::ZeroAmount);
        require!(platform_fee <= amount, MergeOSError::InvalidFee);
        let escrow = &mut ctx.accounts.project_escrow;
        escrow.project_id = project_id;
        escrow.client = ctx.accounts.client.key();
        escrow.vault = ctx.accounts.escrow_vault.key();
        escrow.mrg_mint = ctx.accounts.mrg_mint.key();
        escrow.work_pool = amount.checked_sub(platform_fee).ok_or(MergeOSError::MathOverflow)?;
        escrow.platform_fee = platform_fee;
        escrow.reserved = 0;
        escrow.closed = false;
        escrow.bump = ctx.bumps.project_escrow;

        transfer_checked_from_client(
            &ctx.accounts.client_token,
            &ctx.accounts.escrow_vault,
            &ctx.accounts.client,
            &ctx.accounts.mrg_mint,
            &ctx.accounts.token_program,
            amount,
            ctx.accounts.mrg_mint.decimals,
        )?;
        emit!(ProjectFunded {
            project_id,
            client: escrow.client,
            amount,
            platform_fee,
            work_pool: escrow.work_pool,
            reference,
        });
        Ok(())
    }

    pub fn reserve_task(
        ctx: Context<ReserveTask>,
        task_id: [u8; 32],
        worker: Pubkey,
        amount: u64,
        reference: [u8; 32],
    ) -> Result<()> {
        require!(amount > 0, MergeOSError::ZeroAmount);
        let escrow = &mut ctx.accounts.project_escrow;
        let next_reserved = escrow.reserved.checked_add(amount).ok_or(MergeOSError::MathOverflow)?;
        require!(next_reserved <= escrow.work_pool, MergeOSError::InsufficientEscrow);

        let reserve = &mut ctx.accounts.task_reserve;
        reserve.project = escrow.key();
        reserve.project_id = escrow.project_id;
        reserve.task_id = task_id;
        reserve.worker = worker;
        reserve.amount = amount;
        reserve.status = ReserveStatus::Reserved;
        reserve.bump = ctx.bumps.task_reserve;
        escrow.reserved = next_reserved;

        emit!(TaskReserved {
            project_id: escrow.project_id,
            task_id,
            worker,
            amount,
            reference,
        });
        Ok(())
    }

    pub fn release_task(ctx: Context<ReleaseTask>, reference: [u8; 32]) -> Result<()> {
        require!(ctx.accounts.task_reserve.status == ReserveStatus::Reserved, MergeOSError::ReserveNotOpen);
        let amount = ctx.accounts.task_reserve.amount;
        transfer_checked_from_vault(ctx.accounts.as_vault_transfer(), amount)?;
        ctx.accounts.task_reserve.status = ReserveStatus::Released;
        emit!(TaskPaid {
            project_id: ctx.accounts.task_reserve.project_id,
            task_id: ctx.accounts.task_reserve.task_id,
            worker: ctx.accounts.task_reserve.worker,
            amount,
            reference,
        });
        Ok(())
    }

    pub fn refund_task(ctx: Context<RefundTask>, reference: [u8; 32]) -> Result<()> {
        require!(ctx.accounts.task_reserve.status == ReserveStatus::Reserved, MergeOSError::ReserveNotOpen);
        let amount = ctx.accounts.task_reserve.amount;
        transfer_checked_from_vault(ctx.accounts.as_vault_transfer(), amount)?;
        ctx.accounts.task_reserve.status = ReserveStatus::Refunded;
        emit!(TaskRefunded {
            project_id: ctx.accounts.task_reserve.project_id,
            task_id: ctx.accounts.task_reserve.task_id,
            recipient: ctx.accounts.refund_token.owner,
            amount,
            reference,
        });
        Ok(())
    }

    pub fn close_project(ctx: Context<CloseProject>, reference: [u8; 32]) -> Result<()> {
        let escrow = &mut ctx.accounts.project_escrow;
        require!(!escrow.closed, MergeOSError::ProjectClosed);
        escrow.closed = true;
        emit!(ProjectClosed {
            project_id: escrow.project_id,
            reference,
        });
        Ok(())
    }
}

#[derive(Accounts)]
#[instruction(legacy_chain: LegacyChain, legacy_address_hash: [u8; 32])]
pub struct RegisterLegacyWallet<'info> {
    #[account(mut)]
    pub operator: Signer<'info>,
    /// CHECK: Wallet owner public key recorded for off-chain migration reconciliation.
    pub solana_wallet: UncheckedAccount<'info>,
    #[account(
        init,
        payer = operator,
        space = 8 + WalletMigration::SPACE,
        seeds = [b"wallet-migration", legacy_chain.seed(), legacy_address_hash.as_ref()],
        bump
    )]
    pub wallet_migration: Account<'info, WalletMigration>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct MintMrg<'info> {
    #[account(mut)]
    pub mint_authority: Signer<'info>,
    #[account(mut)]
    pub mrg_mint: Account<'info, Mint>,
    #[account(mut, constraint = recipient_token.mint == mrg_mint.key())]
    pub recipient_token: Account<'info, TokenAccount>,
    pub token_program: Program<'info, Token>,
}

#[derive(Accounts)]
pub struct BurnMrg<'info> {
    #[account(mut)]
    pub owner: Signer<'info>,
    #[account(mut)]
    pub mrg_mint: Account<'info, Mint>,
    #[account(mut, constraint = owner_token.mint == mrg_mint.key(), constraint = owner_token.owner == owner.key())]
    pub owner_token: Account<'info, TokenAccount>,
    pub token_program: Program<'info, Token>,
}

#[derive(Accounts)]
#[instruction(project_id: [u8; 32])]
pub struct FundProject<'info> {
    #[account(mut)]
    pub client: Signer<'info>,
    #[account(mut)]
    pub mrg_mint: Account<'info, Mint>,
    #[account(mut, constraint = client_token.mint == mrg_mint.key(), constraint = client_token.owner == client.key())]
    pub client_token: Account<'info, TokenAccount>,
    #[account(
        init,
        payer = client,
        space = 8 + ProjectEscrow::SPACE,
        seeds = [b"project", project_id.as_ref()],
        bump
    )]
    pub project_escrow: Account<'info, ProjectEscrow>,
    #[account(mut, constraint = escrow_vault.mint == mrg_mint.key())]
    pub escrow_vault: Account<'info, TokenAccount>,
    pub token_program: Program<'info, Token>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
#[instruction(task_id: [u8; 32])]
pub struct ReserveTask<'info> {
    #[account(mut)]
    pub operator: Signer<'info>,
    #[account(mut)]
    pub project_escrow: Account<'info, ProjectEscrow>,
    #[account(
        init,
        payer = operator,
        space = 8 + TaskReserve::SPACE,
        seeds = [b"task", project_escrow.project_id.as_ref(), task_id.as_ref()],
        bump
    )]
    pub task_reserve: Account<'info, TaskReserve>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct ReleaseTask<'info> {
    #[account(mut)]
    pub operator: Signer<'info>,
    #[account(mut)]
    pub project_escrow: Account<'info, ProjectEscrow>,
    #[account(mut, constraint = task_reserve.project == project_escrow.key())]
    pub task_reserve: Account<'info, TaskReserve>,
    #[account(mut, constraint = escrow_vault.key() == project_escrow.vault)]
    pub escrow_vault: Account<'info, TokenAccount>,
    #[account(mut, constraint = worker_token.mint == mrg_mint.key())]
    pub worker_token: Account<'info, TokenAccount>,
    pub mrg_mint: Account<'info, Mint>,
    pub token_program: Program<'info, Token>,
}

#[derive(Accounts)]
pub struct RefundTask<'info> {
    #[account(mut)]
    pub operator: Signer<'info>,
    #[account(mut)]
    pub project_escrow: Account<'info, ProjectEscrow>,
    #[account(mut, constraint = task_reserve.project == project_escrow.key())]
    pub task_reserve: Account<'info, TaskReserve>,
    #[account(mut, constraint = escrow_vault.key() == project_escrow.vault)]
    pub escrow_vault: Account<'info, TokenAccount>,
    #[account(mut, constraint = refund_token.mint == mrg_mint.key())]
    pub refund_token: Account<'info, TokenAccount>,
    pub mrg_mint: Account<'info, Mint>,
    pub token_program: Program<'info, Token>,
}

#[derive(Accounts)]
pub struct CloseProject<'info> {
    #[account(mut)]
    pub operator: Signer<'info>,
    #[account(mut)]
    pub project_escrow: Account<'info, ProjectEscrow>,
}

pub struct VaultTransfer<'a, 'info> {
    pub project_escrow: &'a Account<'info, ProjectEscrow>,
    pub escrow_vault: &'a Account<'info, TokenAccount>,
    pub recipient_token: &'a Account<'info, TokenAccount>,
    pub mrg_mint: &'a Account<'info, Mint>,
    pub token_program: &'a Program<'info, Token>,
}

impl<'info> ReleaseTask<'info> {
    fn as_vault_transfer(&self) -> VaultTransfer<'_, 'info> {
        VaultTransfer {
            project_escrow: &self.project_escrow,
            escrow_vault: &self.escrow_vault,
            recipient_token: &self.worker_token,
            mrg_mint: &self.mrg_mint,
            token_program: &self.token_program,
        }
    }
}

impl<'info> RefundTask<'info> {
    fn as_vault_transfer(&self) -> VaultTransfer<'_, 'info> {
        VaultTransfer {
            project_escrow: &self.project_escrow,
            escrow_vault: &self.escrow_vault,
            recipient_token: &self.refund_token,
            mrg_mint: &self.mrg_mint,
            token_program: &self.token_program,
        }
    }
}

#[account]
pub struct WalletMigration {
    pub legacy_chain: LegacyChain,
    pub legacy_address_hash: [u8; 32],
    pub solana_wallet: Pubkey,
    pub registered_by: Pubkey,
    pub bump: u8,
}

impl WalletMigration {
    pub const SPACE: usize = 1 + 32 + 32 + 32 + 1;
}

#[account]
pub struct ProjectEscrow {
    pub project_id: [u8; 32],
    pub client: Pubkey,
    pub vault: Pubkey,
    pub mrg_mint: Pubkey,
    pub work_pool: u64,
    pub platform_fee: u64,
    pub reserved: u64,
    pub closed: bool,
    pub bump: u8,
}

impl ProjectEscrow {
    pub const SPACE: usize = 32 + 32 + 32 + 32 + 8 + 8 + 8 + 1 + 1;
}

#[account]
pub struct TaskReserve {
    pub project: Pubkey,
    pub project_id: [u8; 32],
    pub task_id: [u8; 32],
    pub worker: Pubkey,
    pub amount: u64,
    pub status: ReserveStatus,
    pub bump: u8,
}

impl TaskReserve {
    pub const SPACE: usize = 32 + 32 + 32 + 32 + 8 + 1 + 1;
}

#[derive(AnchorSerialize, AnchorDeserialize, Clone, Copy, PartialEq, Eq)]
pub enum LegacyChain {
    Trc20,
    Evm,
}

impl LegacyChain {
    pub fn seed(&self) -> &'static [u8] {
        match self {
            LegacyChain::Trc20 => b"trc20",
            LegacyChain::Evm => b"evm",
        }
    }
}

#[derive(AnchorSerialize, AnchorDeserialize, Clone, Copy, PartialEq, Eq)]
pub enum ReserveStatus {
    Reserved,
    Released,
    Refunded,
}

fn transfer_checked_from_client<'info>(
    from: &Account<'info, TokenAccount>,
    to: &Account<'info, TokenAccount>,
    authority: &Signer<'info>,
    mint: &Account<'info, Mint>,
    token_program: &Program<'info, Token>,
    amount: u64,
    decimals: u8,
) -> Result<()> {
    let cpi_accounts = TransferChecked {
        from: from.to_account_info(),
        mint: mint.to_account_info(),
        to: to.to_account_info(),
        authority: authority.to_account_info(),
    };
    token::transfer_checked(
        CpiContext::new(token_program.to_account_info(), cpi_accounts),
        amount,
        decimals,
    )
}

fn transfer_checked_from_vault(ctx: VaultTransfer<'_, '_>, amount: u64) -> Result<()> {
    require!(amount > 0, MergeOSError::ZeroAmount);
    let seeds: &[&[u8]] = &[
        b"project",
        ctx.project_escrow.project_id.as_ref(),
        &[ctx.project_escrow.bump],
    ];
    let signer = &[seeds];
    let cpi_accounts = TransferChecked {
        from: ctx.escrow_vault.to_account_info(),
        mint: ctx.mrg_mint.to_account_info(),
        to: ctx.recipient_token.to_account_info(),
        authority: ctx.project_escrow.to_account_info(),
    };
    token::transfer_checked(
        CpiContext::new_with_signer(ctx.token_program.to_account_info(), cpi_accounts, signer),
        amount,
        ctx.mrg_mint.decimals,
    )
}

#[event]
pub struct LegacyWalletMigrated {
    pub legacy_chain: LegacyChain,
    pub legacy_address_hash: [u8; 32],
    pub solana_wallet: Pubkey,
    pub operator: Pubkey,
}

#[event]
pub struct MrgMinted {
    pub recipient: Pubkey,
    pub amount: u64,
    pub reference: [u8; 32],
}

#[event]
pub struct MrgBurned {
    pub owner: Pubkey,
    pub amount: u64,
    pub reference: [u8; 32],
}

#[event]
pub struct ProjectFunded {
    pub project_id: [u8; 32],
    pub client: Pubkey,
    pub amount: u64,
    pub platform_fee: u64,
    pub work_pool: u64,
    pub reference: [u8; 32],
}

#[event]
pub struct TaskReserved {
    pub project_id: [u8; 32],
    pub task_id: [u8; 32],
    pub worker: Pubkey,
    pub amount: u64,
    pub reference: [u8; 32],
}

#[event]
pub struct TaskPaid {
    pub project_id: [u8; 32],
    pub task_id: [u8; 32],
    pub worker: Pubkey,
    pub amount: u64,
    pub reference: [u8; 32],
}

#[event]
pub struct TaskRefunded {
    pub project_id: [u8; 32],
    pub task_id: [u8; 32],
    pub recipient: Pubkey,
    pub amount: u64,
    pub reference: [u8; 32],
}

#[event]
pub struct ProjectClosed {
    pub project_id: [u8; 32],
    pub reference: [u8; 32],
}

#[error_code]
pub enum MergeOSError {
    #[msg("amount must be greater than zero")]
    ZeroAmount,
    #[msg("platform fee cannot exceed the funded amount")]
    InvalidFee,
    #[msg("arithmetic overflow")]
    MathOverflow,
    #[msg("escrow does not have enough available work pool")]
    InsufficientEscrow,
    #[msg("task reserve is not open")]
    ReserveNotOpen,
    #[msg("project is already closed")]
    ProjectClosed,
}

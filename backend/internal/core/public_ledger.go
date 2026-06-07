package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

const defaultPublicLedgerEventLimit = 80

func (s *Store) PublicLedgerProof() PublicLedgerProofResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projectIDs, taskProjectIDs := s.publicLedgerScopeIndexesLocked()
	response := PublicLedgerProofResponse{
		ProtocolVersion: "mergeos.ledger-proof.v1",
		Kind:            "ledger_proof",
		TokenSymbol:     normalizedTokenSymbol(s.cfg.TokenSymbol),
		Valid:           true,
		RootHash:        strings.Repeat("0", 64),
		PublicRootHash:  strings.Repeat("0", 64),
		GeneratedAt:     time.Now().UTC(),
		Entries:         []PublicLedgerProofRow{},
	}

	previousHash := strings.Repeat("0", 64)
	publicPreviousHash := strings.Repeat("0", 64)
	for index, entry := range s.ledger {
		projectID, taskID := publicLedgerScope(entry, projectIDs, taskProjectIDs)
		publicEntry := publicLedgerEntry(entry, projectID, taskID)
		valid := entry.Sequence == index+1 &&
			entry.PreviousHash == previousHash &&
			entry.EntryHash == ledgerEntryHash(entry)
		if valid {
			response.VerifiedCount++
		} else {
			response.Valid = false
			response.BrokenCount++
		}

		publicHash := publicLedgerProofHash(publicEntry, publicPreviousHash)
		response.Entries = append(response.Entries, PublicLedgerProofRow{
			Sequence:           publicEntry.Sequence,
			Type:               publicEntry.Type,
			AmountCents:        publicEntry.AmountCents,
			Reference:          publicEntry.Reference,
			EntryHash:          publicEntry.EntryHash,
			PublicHash:         publicHash,
			PreviousHash:       publicEntry.PreviousHash,
			PublicPreviousHash: publicPreviousHash,
			Valid:              valid,
			CreatedAt:          publicEntry.CreatedAt,
		})
		previousHash = entry.EntryHash
		publicPreviousHash = publicHash
		response.RootHash = entry.EntryHash
		response.PublicRootHash = publicHash
	}
	response.EntryCount = len(response.Entries)
	response.ContractReference = response.PublicRootHash
	return response
}

func (s *Store) PublicLedgerEvents(limit int) PublicLiveFeedResponse {
	limit = normalizePublicLedgerEventLimit(limit)

	s.mu.RLock()
	defer s.mu.RUnlock()

	projectIDs, taskProjectIDs := s.publicLedgerScopeIndexesLocked()
	response := PublicLiveFeedResponse{
		ProtocolVersion: "mergeos.live-feed.v1",
		Kind:            "live_feed",
		Stats: PublicLiveFeedStats{
			ProjectCount:     len(s.projects),
			LedgerEntryCount: len(s.ledger),
			TokenSymbol:      normalizedTokenSymbol(s.cfg.TokenSymbol),
		},
		Items: []PublicLiveFeedItem{},
	}
	for _, entry := range s.ledger {
		if response.Stats.UpdatedAt == nil || entry.CreatedAt.After(*response.Stats.UpdatedAt) {
			updatedAt := entry.CreatedAt
			response.Stats.UpdatedAt = &updatedAt
		}
		switch entry.Type {
		case "task_payment", "manual_credit":
			response.Stats.AcceptedTaskCount++
		case "token_mint", "platform_fee", "project_reserve", "task_reserve", "payment_verified":
			response.Stats.TotalBudgetCents += entry.AmountCents
		}
		response.Items = append(response.Items, publicLedgerLiveFeedItem(entry, projectIDs, taskProjectIDs, s.projects))
	}
	sort.Slice(response.Items, func(i, j int) bool {
		if response.Items[i].CreatedAt.Equal(response.Items[j].CreatedAt) {
			return response.Items[i].ID > response.Items[j].ID
		}
		return response.Items[i].CreatedAt.After(response.Items[j].CreatedAt)
	})
	if len(response.Items) > limit {
		response.Items = response.Items[:limit]
	}
	return response
}

func (s *Store) PublicTokenEconomy() PublicTokenEconomyResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projectIDs, taskProjectIDs := s.publicLedgerScopeIndexesLocked()
	response := PublicTokenEconomyResponse{
		ProtocolVersion: "mergeos.token-economy.v1",
		Kind:            "token_economy",
		TokenSymbol:     normalizedTokenSymbol(s.cfg.TokenSymbol),
		Balances:        []PublicTokenBalance{},
		Flows:           []PublicTokenFlow{},
		RecentEntries:   []LedgerEntry{},
	}
	flows := map[string]*PublicTokenFlow{}
	touch := func(value time.Time) {
		if value.IsZero() {
			return
		}
		if response.Stats.UpdatedAt == nil || value.After(*response.Stats.UpdatedAt) {
			updatedAt := value
			response.Stats.UpdatedAt = &updatedAt
		}
	}
	addFlow := func(entry LedgerEntry) {
		flow := flows[entry.Type]
		if flow == nil {
			flow = &PublicTokenFlow{
				Type:  entry.Type,
				Label: publicTokenEconomyFlowLabel(entry.Type),
			}
			flows[entry.Type] = flow
		}
		flow.AmountCents += entry.AmountCents
		flow.Count++
		if entry.Sequence > flow.LatestSequence {
			flow.LatestSequence = entry.Sequence
		}
		if flow.UpdatedAt == nil || entry.CreatedAt.After(*flow.UpdatedAt) {
			updatedAt := entry.CreatedAt
			flow.UpdatedAt = &updatedAt
		}
	}

	publicEntries := make([]LedgerEntry, 0, len(s.ledger))
	for _, entry := range s.ledger {
		projectID, taskID := publicLedgerScope(entry, projectIDs, taskProjectIDs)
		publicEntry := publicLedgerEntry(entry, projectID, taskID)
		publicEntries = append(publicEntries, publicEntry)
		response.Stats.LedgerEntryCount++
		touch(entry.CreatedAt)
		addFlow(publicEntry)
		switch entry.Type {
		case "payment_verified":
			response.Totals.VerifiedFundingCents += entry.AmountCents
		case "token_mint":
			response.Totals.MintedCents += entry.AmountCents
			response.Stats.TokenEventCount++
		case "platform_fee":
			response.Totals.PlatformFeeCents += entry.AmountCents
			response.Totals.TreasuryBalanceCents += entry.AmountCents
		case "project_reserve":
			response.Totals.ProjectReserveCents += entry.AmountCents
			response.Stats.EscrowEventCount++
		case "task_reserve":
			response.Totals.TaskReserveCents += entry.AmountCents
			response.Stats.EscrowEventCount++
		case "task_payment":
			response.Totals.ReleasedCents += entry.AmountCents
			response.Stats.PayoutCount++
		case "manual_credit":
			response.Totals.ReleasedCents += entry.AmountCents
			response.Totals.ManualCreditCents += entry.AmountCents
			response.Stats.PayoutCount++
		case "airdrop_claim":
			response.Totals.AirdropClaimCents += entry.AmountCents
			response.Stats.AirdropCount++
			response.Stats.TokenEventCount++
		case "presale_reservation":
			response.Totals.PresaleReserveCents += entry.AmountCents
			response.Stats.PresaleCount++
			response.Stats.TokenEventCount++
		case "token_launch_brief":
			response.Stats.TokenEventCount++
		}
	}
	if response.Totals.MintedCents == 0 {
		response.Totals.MintedCents = response.Totals.VerifiedFundingCents
	}
	response.Totals.TokenSupplyCents = response.Totals.MintedCents
	response.Totals.RemainingReserveCents = maxInt64(0, response.Totals.ProjectReserveCents-response.Totals.ReleasedCents)

	response.Flows = make([]PublicTokenFlow, 0, len(flows))
	for _, flow := range flows {
		response.Flows = append(response.Flows, *flow)
	}
	sort.Slice(response.Flows, func(i, j int) bool {
		if response.Flows[i].LatestSequence == response.Flows[j].LatestSequence {
			return response.Flows[i].Type < response.Flows[j].Type
		}
		return response.Flows[i].LatestSequence > response.Flows[j].LatestSequence
	})
	response.Stats.FlowCount = len(response.Flows)

	taskReserveBalance := maxInt64(0, response.Totals.TaskReserveCents-response.Totals.ReleasedCents)
	response.Balances = publicTokenEconomyBalances(response.Totals, response.Stats, taskReserveBalance)
	response.Stats.BalanceCount = len(response.Balances)

	for index := len(publicEntries) - 1; index >= 0 && len(response.RecentEntries) < 12; index-- {
		response.RecentEntries = append(response.RecentEntries, publicEntries[index])
	}
	return response
}

func (s *Store) publicLedgerScopeIndexesLocked() (map[string]bool, map[string]string) {
	projectIDs := map[string]bool{}
	taskProjectIDs := map[string]string{}
	for _, project := range s.projects {
		projectIDs[project.ID] = true
		for _, task := range project.Tasks {
			if task != nil {
				taskProjectIDs[task.ID] = project.ID
			}
		}
	}
	return projectIDs, taskProjectIDs
}

func publicLedgerEntry(entry LedgerEntry, projectID, taskID string) LedgerEntry {
	publicEntry := entry
	publicEntry.FromAccount = publicLedgerAccount(entry.FromAccount, projectID, taskID)
	publicEntry.ToAccount = publicLedgerAccount(entry.ToAccount, projectID, taskID)
	publicEntry.Reference = publicLedgerReference(projectID, taskID, entry.Sequence, entry.Reference)
	return publicEntry
}

func publicLedgerProofHash(entry LedgerEntry, publicPreviousHash string) string {
	payload := fmt.Sprintf(
		"%d|%s|%s|%s|%d|%s|%s|%s",
		entry.Sequence,
		entry.Type,
		entry.FromAccount,
		entry.ToAccount,
		entry.AmountCents,
		entry.Reference,
		publicPreviousHash,
		entry.CreatedAt.Format(time.RFC3339Nano),
	)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func normalizePublicLedgerEventLimit(limit int) int {
	if limit <= 0 {
		return defaultPublicLedgerEventLimit
	}
	if limit > maxPublicLiveFeedLimit {
		return maxPublicLiveFeedLimit
	}
	return limit
}

func publicTokenEconomyFlowLabel(entryType string) string {
	switch strings.TrimSpace(entryType) {
	case "payment_verified":
		return "Verified funding"
	case "token_mint":
		return "MRG token mint"
	case "platform_fee":
		return "Treasury fee"
	case "project_reserve":
		return "Project reserve"
	case "task_reserve":
		return "Task reserve"
	case "task_payment":
		return "Task payout"
	case "manual_credit":
		return "Manual credit"
	case "airdrop_claim":
		return "Airdrop claim"
	case "presale_reservation":
		return "Presale reservation"
	case "token_launch_brief":
		return "CEO token launch brief"
	default:
		return strings.ReplaceAll(entryType, "_", " ")
	}
}

func publicTokenEconomyBalances(totals PublicTokenEconomyTotals, stats PublicTokenEconomyStats, taskReserveBalance int64) []PublicTokenBalance {
	now := stats.UpdatedAt
	return []PublicTokenBalance{
		{
			ID:          "token_supply",
			Label:       "MRG token supply",
			Role:        "token_supply",
			AmountCents: totals.TokenSupplyCents,
			EntryCount:  stats.TokenEventCount,
			UpdatedAt:   now,
		},
		{
			ID:          "escrow_reserve",
			Label:       "Escrow reserve",
			Role:        "escrow_reserve",
			AmountCents: totals.RemainingReserveCents,
			EntryCount:  stats.EscrowEventCount,
			UpdatedAt:   now,
		},
		{
			ID:          "task_reserve",
			Label:       "Task reserve",
			Role:        "task_reserve",
			AmountCents: taskReserveBalance,
			EntryCount:  stats.EscrowEventCount,
			UpdatedAt:   now,
		},
		{
			ID:          "treasury",
			Label:       "Treasury",
			Role:        "treasury",
			AmountCents: totals.TreasuryBalanceCents,
			EntryCount:  stats.LedgerEntryCount,
			UpdatedAt:   now,
		},
		{
			ID:          "payouts",
			Label:       "Released rewards",
			Role:        "payouts",
			AmountCents: totals.ReleasedCents,
			EntryCount:  stats.PayoutCount,
			UpdatedAt:   now,
		},
		{
			ID:          "airdrop_claims",
			Label:       "Airdrop claims",
			Role:        "airdrop_claims",
			AmountCents: totals.AirdropClaimCents,
			EntryCount:  stats.AirdropCount,
			UpdatedAt:   now,
		},
		{
			ID:          "presale_reserve",
			Label:       "Presale reserve",
			Role:        "presale_reserve",
			AmountCents: totals.PresaleReserveCents,
			EntryCount:  stats.PresaleCount,
			UpdatedAt:   now,
		},
	}
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

package core

import (
	"fmt"
	"strings"
)

func (s *Store) VerifyLedger() LedgerVerificationResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	response := LedgerVerificationResponse{
		Valid:      true,
		EntryCount: len(s.ledger),
		LastHash:   strings.Repeat("0", 64),
	}

	previous := strings.Repeat("0", 64)
	for index, entry := range s.ledger {
		response.LastSequence = entry.Sequence
		response.LastHash = entry.EntryHash
		response.UpdatedAt = nonZeroTimePointer(entry.CreatedAt)

		expectedSequence := index + 1
		if entry.Sequence != expectedSequence {
			return invalidLedgerVerification(response, entry.Sequence, fmt.Sprintf("sequence mismatch: expected %d", expectedSequence))
		}
		if entry.PreviousHash != previous {
			return invalidLedgerVerification(response, entry.Sequence, "previous hash mismatch")
		}
		if expectedHash := ledgerEntryHash(entry); entry.EntryHash != expectedHash {
			return invalidLedgerVerification(response, entry.Sequence, "entry hash mismatch")
		}
		previous = entry.EntryHash
	}
	return response
}

func invalidLedgerVerification(response LedgerVerificationResponse, sequence int, message string) LedgerVerificationResponse {
	response.Valid = false
	response.BrokenSequence = sequence
	response.Error = message
	return response
}

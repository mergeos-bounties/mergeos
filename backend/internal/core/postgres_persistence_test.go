package core

import (
	"database/sql"
	"testing"
)

func TestStringFromNullHandlesNullablePostgresText(t *testing.T) {
	if got := stringFromNull(sql.NullString{}); got != "" {
		t.Fatalf("NULL string = %q, want empty string", got)
	}
	if got := stringFromNull(sql.NullString{String: "solana", Valid: true}); got != "solana" {
		t.Fatalf("valid string = %q, want solana", got)
	}
}

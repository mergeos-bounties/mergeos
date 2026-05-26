package core

import (
	"context"
	"testing"
	"time"
)

func TestFindOrCreateIdentity(t *testing.T) {
	s := newTestStore()

	ctx := context.Background()

	// Create new user via OAuth identity
	user, err := s.FindOrCreateIdentity(ctx, "google", "google_abc123", "test@gmail.com", "Test User")
	if err != nil {
		t.Fatalf("FindOrCreateIdentity failed: %v", err)
	}
	if user.ID == "" {
		t.Fatal("expected non-empty user ID")
	}
	if user.Name != "Test User" {
		t.Fatalf("expected name 'Test User', got %q", user.Name)
	}
	if user.Email != "test@gmail.com" {
		t.Fatalf("expected email 'test@gmail.com', got %q", user.Email)
	}
	if len(user.IdentityProviders) != 1 {
		t.Fatalf("expected 1 identity provider, got %d", len(user.IdentityProviders))
	}

	// Same identity should return same user
	user2, err := s.FindOrCreateIdentity(ctx, "google", "google_abc123", "test@gmail.com", "Test User")
	if err != nil {
		t.Fatalf("FindOrCreateIdentity duplicate failed: %v", err)
	}
	if user2.ID != user.ID {
		t.Fatalf("expected same user ID for same identity, got %s != %s", user2.ID, user.ID)
	}

	// Email match should link identity to existing user
	user3, err := s.FindOrCreateIdentity(ctx, "github", "github_456", "test@gmail.com", "Test User 2")
	if err != nil {
		t.Fatalf("FindOrCreateIdentity email match failed: %v", err)
	}
	if user3.ID != user.ID {
		t.Fatalf("expected same user ID for email match, got %s != %s", user3.ID, user.ID)
	}
	if len(user3.IdentityProviders) != 2 {
		t.Fatalf("expected 2 identity providers after linking, got %d", len(user3.IdentityProviders))
	}
}

func TestFindOrCreateIdentity_NewEmail(t *testing.T) {
	s := newTestStore()
	ctx := context.Background()

	user, err := s.FindOrCreateIdentity(ctx, "github", "github_789", "new@example.com", "New User")
	if err != nil {
		t.Fatalf("FindOrCreateIdentity failed: %v", err)
	}
	if user.Email != "new@example.com" {
		t.Fatalf("expected 'new@example.com', got %q", user.Email)
	}
}

func newTestStore() *Store {
	return NewStore(StoreOpts{
		StatePath:     "",
		DatabaseURL:   "",
		AdminEmail:    "admin@test.com",
		AdminPassword: "admin123",
		AdminAutoPromote: true,
		TokenSymbol:   "MRG",
	})
}

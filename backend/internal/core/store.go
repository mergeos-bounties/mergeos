package core

import (
	"context"
	"fmt"
	"time"
)

func (s *Store) FindOrCreateIdentity(ctx context.Context, provider, providerID, email, name string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	identityKey := fmt.Sprintf("%s:%s", provider, providerID)

	// Check if identity already exists
	for _, user := range s.data.Users {
		if user.IdentityProviders == nil { continue }
		if _, exists := user.IdentityProviders[identityKey]; exists {
			user.LastLoginAt = timePtr(time.Now())
			s.dirty = true
			return user, nil
		}
	}

	// Check if email matches existing user (link accounts)
	for _, user := range s.data.Users {
		if user.Email == email {
			if user.IdentityProviders == nil {
				user.IdentityProviders = make(map[string]string)
			}
			user.IdentityProviders[identityKey] = providerID
			user.LastLoginAt = timePtr(time.Now())
			s.dirty = true
			return user, nil
		}
	}

	// Create new user
	userID := s.nextID()
	now := time.Now().UTC()
	user := &User{
		ID:                userID,
		Name:              name,
		Email:             email,
		Role:              RoleClient,
		IdentityProviders: map[string]string{identityKey: providerID},
		CreatedAt:         now,
		LastLoginAt:       timePtr(now),
	}
	s.data.Users = append(s.data.Users, user)
	s.dirty = true
	return user, nil
}

func timePtr(t time.Time) *time.Time { return &t }
package core

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
)

const passwordIterations = 120000

func publicUser(user *User) PublicUser {
	if user == nil {
		return PublicUser{}
	}
	return PublicUser{
		ID:          user.ID,
		Name:        user.Name,
		CompanyName: user.CompanyName,
		Email:       user.Email,
		Role:        normalizeRole(user.Role),
		CreatedAt:   user.CreatedAt,
		LastLoginAt: user.LastLoginAt,
	}
}

func normalizeRole(role UserRole) UserRole {
	if role == RoleAdmin {
		return RoleAdmin
	}
	return RoleClient
}

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", errors.New("email is required")
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || strings.ToLower(parsed.Address) != email {
		return "", errors.New("email is invalid")
	}
	return email, nil
}

func newToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func hashPassword(password string) (string, string, error) {
	password = strings.TrimSpace(password)
	if len(password) < 8 {
		return "", "", errors.New("password must be at least 8 characters")
	}
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", "", err
	}
	salt := hex.EncodeToString(saltBytes)
	return salt, derivePasswordHash(password, salt), nil
}

func verifyPassword(password, salt, expected string) bool {
	actual := derivePasswordHash(strings.TrimSpace(password), salt)
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}

func derivePasswordHash(password, salt string) string {
	key := []byte(salt + ":" + password)
	sum := hmacSHA256(key, []byte("mergeos-password"))
	for i := 0; i < passwordIterations; i++ {
		sum = hmacSHA256(key, sum)
	}
	return hex.EncodeToString(sum)
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(data)
	return mac.Sum(nil)
}

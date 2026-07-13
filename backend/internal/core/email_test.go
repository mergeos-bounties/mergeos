package core

import (
	"testing"
)

func TestEmailSenderReturnsLoggedWhenSMTPNotConfigured(t *testing.T) {
	cfg := Config{}
	sender := NewEmailSender(cfg)
	result := sender.Send("user@example.com", "Test Subject", "Test body")
	if result != "logged:mail-not-configured" {
		t.Errorf("expected logged:mail-not-configured, got %s", result)
	}
}

func TestEmailSenderReturnsSkippedWhenNoRecipient(t *testing.T) {
	cfg := Config{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     "587",
		SMTPUsername: "user",
		SMTPPassword: "pass",
		SMTPFrom:     "noreply@test.local",
	}
	sender := NewEmailSender(cfg)
	result := sender.Send("", "Test Subject", "Test body")
	if result != "skipped:no-recipient" {
		t.Errorf("expected skipped:no-recipient, got %s", result)
	}
}

func TestEmailSenderReturnsSkippedWhenWhitespaceRecipient(t *testing.T) {
	cfg := Config{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     "587",
		SMTPUsername: "user",
		SMTPPassword: "pass",
		SMTPFrom:     "noreply@test.local",
	}
	sender := NewEmailSender(cfg)
	result := sender.Send("  ", "Test Subject", "Test body")
	if result != "skipped:no-recipient" {
		t.Errorf("expected skipped:no-recipient, got %s", result)
	}
}

func TestEmailSenderReturnsErrorOnBadHost(t *testing.T) {
	cfg := Config{
		SMTPHost:     "nonexistent.smtp.local",
		SMTPPort:     "25",
		SMTPUsername: "user",
		SMTPPassword: "pass",
		SMTPFrom:     "noreply@test.local",
	}
	sender := NewEmailSender(cfg)
	result := sender.Send("user@example.com", "Test Subject", "Test body")
	if len(result) < 6 || result[:6] != "error:" {
		t.Errorf("expected error: prefix, got %s", result)
	}
}

func TestEmailSenderSMTPReadyFalseWhenUnconfigured(t *testing.T) {
	cfg := Config{}
	if cfg.SMTPReady() {
		t.Error("expected SMTPReady() to be false with empty config")
	}
}

func TestEmailSenderSMTPReadyTrueWhenConfigured(t *testing.T) {
	cfg := Config{
		SMTPHost:     "smtp.example.com",
		SMTPUsername: "user",
		SMTPPassword: "pass",
		SMTPFrom:     "noreply@test.local",
	}
	if !cfg.SMTPReady() {
		t.Error("expected SMTPReady() to be true with valid config")
	}
}

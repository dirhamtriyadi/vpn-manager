package auth

import (
	"testing"
	"time"
)

func newTestService() *Service {
	return NewService("test-secret", time.Hour)
}

func TestIssueAndValidate(t *testing.T) {
	svc := newTestService()
	now := time.Unix(1_700_000_000, 0)

	token, exp, err := svc.Issue(42, now)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if !exp.After(now) {
		t.Fatal("expected expiry in the future")
	}

	id, err := svc.Validate(token, now)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if id != 42 {
		t.Fatalf("expected subject 42, got %d", id)
	}
}

func TestValidateExpired(t *testing.T) {
	svc := newTestService()
	now := time.Unix(1_700_000_000, 0)
	token, _, _ := svc.Issue(7, now)

	if _, err := svc.Validate(token, now.Add(2*time.Hour)); err != ErrExpiredToken {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}

func TestValidateTampered(t *testing.T) {
	svc := newTestService()
	now := time.Unix(1_700_000_000, 0)
	token, _, _ := svc.Issue(7, now)

	if _, err := svc.Validate(token+"x", now); err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken for tampered signature, got %v", err)
	}
	if _, err := svc.Validate("garbage", now); err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken for malformed token, got %v", err)
	}

	// A token signed with a different secret must not validate.
	other := NewService("other-secret", time.Hour)
	if _, err := svc.Validate(mustIssue(t, other, now), now); err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken for foreign secret, got %v", err)
	}
}

func mustIssue(t *testing.T, s *Service, now time.Time) string {
	t.Helper()
	token, _, err := s.Issue(7, now)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	return token
}

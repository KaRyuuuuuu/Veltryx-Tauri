package desktopsession

import (
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return &Store{
		path:    filepath.Join(t.TempDir(), "sessions.json"),
		records: make(map[string]Record),
	}
}

func TestStoreRegisterValidateDelete(t *testing.T) {
	s := newTestStore(t)
	record := s.Register(Record{
		SessionID: "s1",
		DeviceID:  "d1",
		User: User{
			ID:       "u1",
			Role:     "viewer",
			Username: "john",
		},
	})

	if record.ServerSignature == "" {
		t.Fatalf("expected server signature")
	}
	if !record.ExpiresAt.After(record.IssuedAt) {
		t.Fatalf("expected expiration after issue time")
	}

	validated, ok, reason := s.Validate("s1", "d1", "u1")
	if !ok || reason != "" {
		t.Fatalf("expected validation success, ok=%v reason=%q", ok, reason)
	}
	if validated.SessionID != "s1" {
		t.Fatalf("unexpected validated record: %+v", validated)
	}

	s.Delete("s1")
	if _, ok, reason = s.Validate("s1", "", ""); ok || reason != "session_not_found" {
		t.Fatalf("expected deleted session_not_found, ok=%v reason=%q", ok, reason)
	}
}

func TestStoreValidateFailures(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()
	s.records["expired"] = Record{
		SessionID: "expired",
		DeviceID:  "d1",
		User:      User{ID: "u1", Role: "viewer"},
		IssuedAt:  now.Add(-2 * time.Hour),
		ExpiresAt: now.Add(-time.Minute),
	}
	s.records["active"] = Record{
		SessionID: "active",
		DeviceID:  "d1",
		User:      User{ID: "u1", Role: "viewer"},
		IssuedAt:  now,
		ExpiresAt: now.Add(time.Hour),
	}

	if _, ok, reason := s.Validate("expired", "d1", "u1"); ok || reason != "session_expired" {
		t.Fatalf("expected session_expired, ok=%v reason=%q", ok, reason)
	}
	if _, ok, reason := s.Validate("active", "d2", "u1"); ok || reason != "device_mismatch" {
		t.Fatalf("expected device_mismatch, ok=%v reason=%q", ok, reason)
	}
	if _, ok, reason := s.Validate("active", "d1", "u2"); ok || reason != "user_mismatch" {
		t.Fatalf("expected user_mismatch, ok=%v reason=%q", ok, reason)
	}
}

func TestStoreListFiltersExpiredAndSorts(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()
	s.records["old"] = Record{
		SessionID: "old",
		User:      User{ID: "u1", Role: "viewer"},
		IssuedAt:  now.Add(-2 * time.Hour),
		ExpiresAt: now.Add(time.Hour),
	}
	s.records["new"] = Record{
		SessionID: "new",
		User:      User{ID: "u2", Role: "viewer"},
		IssuedAt:  now.Add(-time.Hour),
		ExpiresAt: now.Add(time.Hour),
	}
	s.records["expired"] = Record{
		SessionID: "expired",
		User:      User{ID: "u3", Role: "viewer"},
		IssuedAt:  now.Add(-3 * time.Hour),
		ExpiresAt: now.Add(-time.Minute),
	}

	out := s.List()
	if len(out) != 2 {
		t.Fatalf("expected 2 active records, got %d", len(out))
	}
	if out[0].SessionID != "new" || out[1].SessionID != "old" {
		t.Fatalf("expected sorted by issued_at desc, got %#v", out)
	}
}

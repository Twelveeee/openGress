package api

import (
	"errors"
	"testing"
	"time"
)

func TestAuthStoreRegisterCaseInsensitiveUnique(t *testing.T) {
	store := NewAuthStore("test-secret", time.Hour)
	if _, err := store.Register("UserOne", "pw1"); err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	if _, err := store.Register("userone", "pw2"); !errors.Is(err, ErrUserExists) {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}
}

func TestAuthStoreRegisterCaseInsensitiveUniqueAcrossRestart(t *testing.T) {
	sharedRepo := newInMemoryAuthUserRepo()
	first := NewAuthStoreWithRepository("test-secret", time.Hour, sharedRepo)
	if _, err := first.Register("Alpha", "pw1"); err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	second := NewAuthStoreWithRepository("test-secret", time.Hour, sharedRepo)
	if _, err := second.Register("ALPHA", "pw2"); !errors.Is(err, ErrUserExists) {
		t.Fatalf("expected ErrUserExists across restart, got %v", err)
	}
}

func TestAuthStoreAuthenticateCaseInsensitiveUsername(t *testing.T) {
	store := NewAuthStore("test-secret", time.Hour)
	if _, err := store.Register("MiXeD", "pw1"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if _, err := store.Authenticate("mixed", "pw1"); err != nil {
		t.Fatalf("authenticate failed: %v", err)
	}
}

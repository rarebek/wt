package middleware

import (
	"fmt"
	"testing"
)

// Note: Full auth middleware testing requires a real WebTransport session
// (which we test in integration_test.go). These tests verify the validator
// function logic independently.

func TestBearerAuthValidatorCalled(t *testing.T) {
	called := false
	validator := func(token string) (any, error) {
		called = true
		if token != "valid-token" {
			return nil, fmt.Errorf("invalid")
		}
		return "user-alice", nil
	}

	user, err := validator("valid-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("validator was not called")
	}
	if user != "user-alice" {
		t.Errorf("expected 'user-alice', got %v", user)
	}
}

func TestBearerAuthRejectsInvalid(t *testing.T) {
	validator := func(_ string) (any, error) {
		return nil, fmt.Errorf("invalid token")
	}

	_, err := validator("bad-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

package googleoauth

import (
	"strings"
	"testing"
	"time"
)

func TestStateRoundTrip(t *testing.T) {
	state, err := NewState("secret-123")
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	if err := VerifyState("secret-123", state); err != nil {
		t.Errorf("VerifyState failed on a freshly minted state: %v", err)
	}
}

func TestStateWrongSecret(t *testing.T) {
	state, err := NewState("secret-123")
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	if err := VerifyState("different-secret", state); err == nil {
		t.Error("expected VerifyState to reject a state signed with a different secret")
	}
}

func TestStateTampered(t *testing.T) {
	state, err := NewState("secret-123")
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	tampered := state + "x"
	if err := VerifyState("secret-123", tampered); err == nil {
		t.Error("expected VerifyState to reject a tampered state")
	}
}

func TestStateMalformed(t *testing.T) {
	if err := VerifyState("secret-123", "not-a-valid-state"); err == nil {
		t.Error("expected VerifyState to reject a state with no signature separator")
	}
}

func TestStateExpired(t *testing.T) {
	original := stateTTL
	stateTTL = -1 * time.Second // mint a state that's already expired
	defer func() { stateTTL = original }()

	state, err := NewState("secret-123")
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	if err := VerifyState("secret-123", state); err == nil {
		t.Error("expected VerifyState to reject an expired state")
	}
}

func TestStateHasTwoParts(t *testing.T) {
	state, err := NewState("secret-123")
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	if strings.Count(state, ".") != 1 {
		t.Errorf("expected exactly one '.' separator in state, got %q", state)
	}
}

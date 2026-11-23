package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGenerateAnonymousID(t *testing.T) {
	id := GenerateAnonymousID()
	if len(id) != 36 {
		t.Fatalf("expected 36-character UUID, got length %d: %q", len(id), id)
	}
	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("GenerateAnonymousID() did not produce a valid UUID: %v", err)
	}
	if parsed.String() != id {
		t.Fatalf("parsed UUID %q does not match original %q", parsed.String(), id)
	}
}

func TestGenerateAndValidateAnonymousToken(t *testing.T) {
	secret := "a-32-character-test-secret-value!!"
	anonID := GenerateAnonymousID()

	token, salt, err := GenerateTokenWithVersionAndAnonymous(secret, anonID, 1, true, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate anonymous token: %v", err)
	}

	claims, err := ValidateToken(secret, token, salt)
	if err != nil {
		t.Fatalf("failed to validate anonymous token: %v", err)
	}

	if claims.Subject != anonID {
		t.Fatalf("subject = %q, want %q", claims.Subject, anonID)
	}
	if claims.SessionVersion != 1 {
		t.Fatalf("session_version = %d, want 1", claims.SessionVersion)
	}
	if !claims.IsAnonymous {
		t.Fatal("expected is_anonymous claim to be true")
	}
}

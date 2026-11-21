package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndValidateToken(t *testing.T) {
	token, salt, err := GenerateToken("a-32-character-test-secret-value!!", "user-1", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ValidateToken("a-32-character-test-secret-value!!", token, salt)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "user-1" {
		t.Fatalf("subject = %q, want user-1", claims.Subject)
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.Time.After(time.Now()) {
		t.Fatal("expected a future expiration")
	}
}

func TestValidateTokenRejectsNonHS256Algorithm(t *testing.T) {
	secret, salt := "a-32-character-test-secret-value!!", "fixed-test-salt"
	claims := Claims{Subject: "user-1", RegisteredClaims: jwt.RegisteredClaims{
		Subject: "user-1", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString(signingKey(secret, salt))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateToken(secret, token, salt); err == nil {
		t.Fatal("expected HS512 token to be rejected")
	}
}

func TestTokenFingerprint(t *testing.T) {
	first := TokenFingerprint("header.payload.signature")
	if len(first) != 64 {
		t.Fatalf("fingerprint length = %d, want 64", len(first))
	}
	if first != TokenFingerprint("header.payload.signature") {
		t.Fatal("fingerprint must be deterministic")
	}
	if first == TokenFingerprint("different-token") {
		t.Fatal("different tokens must have different fingerprints")
	}
}

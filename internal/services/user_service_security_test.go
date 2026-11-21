package services

import (
	"testing"

	"github.com/souvik03-136/Go-Store/internal/models"
)

func TestProfileUpdateRequiresCurrentPassword(t *testing.T) {
	user, err := models.NewUser("u1", "alice", "alice@example.com", "current-password")
	if err != nil {
		t.Fatal(err)
	}
	for _, password := range []string{"", "wrong-password"} {
		if err := verifyCurrentPassword(user, password); err == nil {
			t.Fatalf("password %q unexpectedly accepted", password)
		}
	}
	if err := verifyCurrentPassword(user, "current-password"); err != nil {
		t.Fatalf("correct current password rejected: %v", err)
	}
}

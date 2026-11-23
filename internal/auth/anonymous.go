// internal/auth/anonymous.go

package auth

import "github.com/google/uuid"

// GenerateAnonymousID returns a standard random UUID string (36 characters) to identify
// an anonymous guest account in VARCHAR(36) database columns.
func GenerateAnonymousID() string {
	return uuid.New().String()
}

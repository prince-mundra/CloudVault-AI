package services

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/souvik03-136/Go-Store/internal/auth"
	"github.com/souvik03-136/Go-Store/internal/models"
)

type memoryUserStore struct {
	users      map[string]*models.User
	byEmail    map[string]*models.User
	byUsername map[string]*models.User
}

func newMemoryUserStore() *memoryUserStore {
	return &memoryUserStore{
		users:      make(map[string]*models.User),
		byEmail:    make(map[string]*models.User),
		byUsername: make(map[string]*models.User),
	}
}

func (m *memoryUserStore) CreateUser(u *models.User) error {
	if _, ok := m.users[u.ID]; ok {
		return errors.New("user already exists")
	}
	if _, ok := m.byEmail[u.Email]; ok {
		return errors.New("email already exists")
	}
	if _, ok := m.byUsername[u.Username]; ok {
		return errors.New("username already exists")
	}
	copied := *u
	m.users[u.ID] = &copied
	m.byEmail[u.Email] = &copied
	m.byUsername[u.Username] = &copied
	return nil
}

func (m *memoryUserStore) GetUserByID(id string) (*models.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	copied := *u
	return &copied, nil
}

func (m *memoryUserStore) GetUserByEmail(email string) (*models.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, errors.New("user not found")
	}
	copied := *u
	return &copied, nil
}

func (m *memoryUserStore) GetUserByUsername(username string) (*models.User, error) {
	u, ok := m.byUsername[username]
	if !ok {
		return nil, errors.New("user not found")
	}
	copied := *u
	return &copied, nil
}

func (m *memoryUserStore) GetAuthVersion(id string) (int64, error) {
	u, err := m.GetUserByID(id)
	if err != nil {
		return 0, err
	}
	return u.AuthVersion, nil
}

func (m *memoryUserStore) GetAuthInfo(id string) (int64, bool, error) {
	u, err := m.GetUserByID(id)
	if err != nil {
		return 0, false, err
	}
	return u.AuthVersion, u.IsAnonymous, nil
}

type memoryRevocationStore struct {
	revoked map[string]bool
}

func (r *memoryRevocationStore) IsRevoked(tokenHash string) (bool, error) {
	return r.revoked[tokenHash], nil
}

func (r *memoryRevocationStore) Revoke(tokenHash string, expiresAt time.Time) error {
	r.revoked[tokenHash] = true
	return nil
}

func TestAnonymousRegistrationAndTokenValidation(t *testing.T) {
	secret := "test-secret-at-least-32-characters-long!"
	userStore := newMemoryUserStore()
	revStore := &memoryRevocationStore{revoked: make(map[string]bool)}
	svc := NewAuthService(userStore, revStore, secret)

	anonID, pair, err := svc.RegisterAnonymous()
	if err != nil {
		t.Fatalf("RegisterAnonymous failed: %v", err)
	}

	if len(anonID) != 36 {
		t.Fatalf("expected 36-char UUID, got length %d: %q", len(anonID), anonID)
	}
	if pair.Token == "" || pair.Salt == "" {
		t.Fatal("expected non-empty token and salt")
	}

	// Verify persistence in user store
	saved, err := userStore.GetUserByID(anonID)
	if err != nil {
		t.Fatalf("user not found in store: %v", err)
	}
	if saved.Role != "anonymous" {
		t.Fatalf("role = %q, want 'anonymous'", saved.Role)
	}
	if !saved.IsAnonymous {
		t.Fatal("expected is_anonymous = true")
	}
	if saved.PasswordHash != "!ANONYMOUS_LOCKED!" {
		t.Fatalf("password_hash = %q, want locked sentinel", saved.PasswordHash)
	}
	if saved.AuthVersion != 1 {
		t.Fatalf("auth_version = %d, want 1", saved.AuthVersion)
	}

	// Verify token claims
	claims, err := auth.ValidateToken(secret, pair.Token, pair.Salt)
	if err != nil {
		t.Fatalf("token validation failed: %v", err)
	}
	if claims.Subject != anonID {
		t.Fatalf("claims.Subject = %q, want %q", claims.Subject, anonID)
	}
	if claims.SessionVersion != 1 {
		t.Fatalf("claims.SessionVersion = %d, want 1", claims.SessionVersion)
	}
	if !claims.IsAnonymous {
		t.Fatal("claims.IsAnonymous must be true")
	}

	// Verify AuthService.ValidateToken succeeds
	subject, err := svc.ValidateToken(pair.Token, pair.Salt)
	if err != nil {
		t.Fatalf("svc.ValidateToken failed: %v", err)
	}
	if subject != anonID {
		t.Fatalf("subject = %q, want %q", subject, anonID)
	}
}

func TestAnonymousUserCannotLoginViaOAuth(t *testing.T) {
	secret := "test-secret-at-least-32-characters-long!"
	userStore := newMemoryUserStore()
	revStore := &memoryRevocationStore{revoked: make(map[string]bool)}
	svc := NewAuthService(userStore, revStore, secret)

	anonID, _, err := svc.RegisterAnonymous()
	if err != nil {
		t.Fatalf("RegisterAnonymous failed: %v", err)
	}
	saved, err := userStore.GetUserByID(anonID)
	if err != nil {
		t.Fatal(err)
	}

	// Attempt login with anonymous user's email
	_, _, err = svc.LoginOAuth(saved.Email, "anyPassword123")
	if err == nil {
		t.Fatal("expected LoginOAuth to fail for anonymous user")
	}
	if !strings.Contains(err.Error(), "invalid credentials") {
		t.Fatalf("expected 'invalid credentials', got: %v", err)
	}
}

func TestOAuthRegistrationAndLoginCompatibility(t *testing.T) {
	secret := "test-secret-at-least-32-characters-long!"
	userStore := newMemoryUserStore()
	revStore := &memoryRevocationStore{revoked: make(map[string]bool)}
	svc := NewAuthService(userStore, revStore, secret)

	user, pair, err := svc.RegisterOAuth("alice", "alice@example.com", "Password123!")
	if err != nil {
		t.Fatalf("RegisterOAuth failed: %v", err)
	}
	if user.Role != "user" {
		t.Fatalf("role = %q, want 'user'", user.Role)
	}
	if user.IsAnonymous {
		t.Fatal("expected is_anonymous = false for registered user")
	}

	claims, err := auth.ValidateToken(secret, pair.Token, pair.Salt)
	if err != nil {
		t.Fatal(err)
	}
	if claims.IsAnonymous {
		t.Fatal("expected claims.IsAnonymous = false for registered user")
	}

	// Login with correct credentials
	loginUser, loginPair, err := svc.LoginOAuth("alice@example.com", "Password123!")
	if err != nil {
		t.Fatalf("LoginOAuth failed: %v", err)
	}
	if loginUser.ID != user.ID {
		t.Fatalf("loginUser.ID = %q, want %q", loginUser.ID, user.ID)
	}

	loginClaims, err := auth.ValidateToken(secret, loginPair.Token, loginPair.Salt)
	if err != nil {
		t.Fatal(err)
	}
	if loginClaims.IsAnonymous {
		t.Fatal("expected claims.IsAnonymous = false for login token")
	}

	// Login with wrong password fails
	_, _, err = svc.LoginOAuth("alice@example.com", "WrongPassword!")
	if err == nil {
		t.Fatal("expected wrong password to fail")
	}
}

func TestValidateTokenDetectsAnonymousClaimMismatch(t *testing.T) {
	secret := "test-secret-at-least-32-characters-long!"
	userStore := newMemoryUserStore()
	revStore := &memoryRevocationStore{revoked: make(map[string]bool)}
	svc := NewAuthService(userStore, revStore, secret)

	// Create registered user (is_anonymous = false in DB)
	regUser, _, err := svc.RegisterOAuth("bob", "bob@example.com", "Password123!")
	if err != nil {
		t.Fatal(err)
	}

	// Forge token for registered user claiming is_anonymous = true
	forgedToken, forgedSalt, err := auth.GenerateTokenWithVersionAndAnonymous(secret, regUser.ID, regUser.AuthVersion, true, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.ValidateToken(forgedToken, forgedSalt)
	if err == nil {
		t.Fatal("expected ValidateToken to reject token identity mismatch")
	}
	if !strings.Contains(err.Error(), "token identity mismatch") {
		t.Fatalf("expected 'token identity mismatch', got: %v", err)
	}

	// Create anonymous user (is_anonymous = true in DB)
	anonID, _, err := svc.RegisterAnonymous()
	if err != nil {
		t.Fatal(err)
	}

	// Forge token for anonymous user claiming is_anonymous = false
	forgedToken2, forgedSalt2, err := auth.GenerateTokenWithVersionAndAnonymous(secret, anonID, 1, false, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.ValidateToken(forgedToken2, forgedSalt2)
	if err == nil {
		t.Fatal("expected ValidateToken to reject token identity mismatch for anonymous account")
	}
	if !strings.Contains(err.Error(), "token identity mismatch") {
		t.Fatalf("expected 'token identity mismatch', got: %v", err)
	}
}

func TestRegisterOAuthSuccess(t *testing.T) {
	secret := "test-secret-at-least-32-characters-long!"
	userStore := newMemoryUserStore()
	revStore := &memoryRevocationStore{revoked: make(map[string]bool)}
	svc := NewAuthService(userStore, revStore, secret)

	user, pair, err := svc.RegisterOAuth("charlie", "charlie@example.com", "Password123!")
	if err != nil {
		t.Fatalf("RegisterOAuth failed: %v", err)
	}
	if user.Username != "charlie" {
		t.Fatalf("username = %q, want 'charlie'", user.Username)
	}
	if user.Email != "charlie@example.com" {
		t.Fatalf("email = %q, want 'charlie@example.com'", user.Email)
	}
	if user.Role != "user" || user.IsAnonymous {
		t.Fatalf("unexpected role/anonymous: role=%q, is_anon=%v", user.Role, user.IsAnonymous)
	}
	if pair.Token == "" || pair.Salt == "" {
		t.Fatal("expected non-empty token and salt")
	}
}

func TestRegisterOAuthDuplicateEmail(t *testing.T) {
	secret := "test-secret-at-least-32-characters-long!"
	userStore := newMemoryUserStore()
	revStore := &memoryRevocationStore{revoked: make(map[string]bool)}
	svc := NewAuthService(userStore, revStore, secret)

	_, _, err := svc.RegisterOAuth("user1", "duplicate@example.com", "Password123!")
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Try registering with same email but different username
	_, _, err = svc.RegisterOAuth("user2", "duplicate@example.com", "Password123!")
	if err == nil {
		t.Fatal("expected error for duplicate email, got nil")
	}
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got: %v", err)
	}
	if !strings.Contains(err.Error(), "email already exists") {
		t.Fatalf("expected error message to contain 'email already exists', got: %v", err)
	}
}

func TestRegisterOAuthDuplicateUsername(t *testing.T) {
	secret := "test-secret-at-least-32-characters-long!"
	userStore := newMemoryUserStore()
	revStore := &memoryRevocationStore{revoked: make(map[string]bool)}
	svc := NewAuthService(userStore, revStore, secret)

	_, _, err := svc.RegisterOAuth("sameuser", "user1@example.com", "Password123!")
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Try registering with same username but different email
	_, _, err = svc.RegisterOAuth("sameuser", "user2@example.com", "Password123!")
	if err == nil {
		t.Fatal("expected error for duplicate username, got nil")
	}
	if !errors.Is(err, ErrUsernameAlreadyExists) {
		t.Fatalf("expected ErrUsernameAlreadyExists, got: %v", err)
	}
	if !strings.Contains(err.Error(), "username already exists") {
		t.Fatalf("expected error message to contain 'username already exists', got: %v", err)
	}
}

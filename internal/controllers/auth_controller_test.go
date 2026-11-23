package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/souvik03-136/Go-Store/internal/models"
	"github.com/souvik03-136/Go-Store/internal/services"
)

type mockUserAccountStore struct {
	users      map[string]*models.User
	byEmail    map[string]*models.User
	byUsername map[string]*models.User
}

func newMockUserAccountStore() *mockUserAccountStore {
	return &mockUserAccountStore{
		users:      make(map[string]*models.User),
		byEmail:    make(map[string]*models.User),
		byUsername: make(map[string]*models.User),
	}
}

func (m *mockUserAccountStore) CreateUser(u *models.User) error {
	copied := *u
	m.users[u.ID] = &copied
	m.byEmail[u.Email] = &copied
	m.byUsername[u.Username] = &copied
	return nil
}

func (m *mockUserAccountStore) GetUserByID(id string) (*models.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	copied := *u
	return &copied, nil
}

func (m *mockUserAccountStore) GetUserByEmail(email string) (*models.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, errors.New("user not found")
	}
	copied := *u
	return &copied, nil
}

func (m *mockUserAccountStore) GetUserByUsername(username string) (*models.User, error) {
	u, ok := m.byUsername[username]
	if !ok {
		return nil, errors.New("user not found")
	}
	copied := *u
	return &copied, nil
}

func (m *mockUserAccountStore) GetAuthVersion(id string) (int64, error) {
	return 1, nil
}

func (m *mockUserAccountStore) GetAuthInfo(id string) (int64, bool, error) {
	return 1, false, nil
}

type mockRevocationStore struct{}

func (m *mockRevocationStore) IsRevoked(tokenHash string) (bool, error) {
	return false, nil
}

func (m *mockRevocationStore) Revoke(tokenHash string, expiresAt time.Time) error {
	return nil
}

func setupAuthRouter() (*gin.Engine, *mockUserAccountStore) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	userStore := newMockUserAccountStore()
	revStore := &mockRevocationStore{}
	authSvc := services.NewAuthService(userStore, revStore, "test-secret-at-least-32-characters-long!")
	authCtrl := NewAuthController(authSvc)

	router.POST("/v1/auth/oauth/register", authCtrl.RegisterOAuthUser)
	return router, userStore
}

func TestRegisterOAuthUser_Success(t *testing.T) {
	router, _ := setupAuthRouter()

	body := map[string]string{
		"username": "johndoe",
		"email":    "john@example.com",
		"password": "Password123!",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/oauth/register", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["token"] == nil || resp["token"] == "" {
		t.Fatal("expected non-empty token")
	}
	if resp["salt"] == nil || resp["salt"] == "" {
		t.Fatal("expected non-empty salt")
	}
	userObj, ok := resp["user"].(map[string]any)
	if !ok || userObj["username"] != "johndoe" || userObj["email"] != "john@example.com" {
		t.Fatalf("unexpected user object: %v", userObj)
	}
}

func TestRegisterOAuthUser_DuplicateEmail(t *testing.T) {
	router, _ := setupAuthRouter()

	// First registration
	body1 := map[string]string{
		"username": "user1",
		"email":    "duplicate@example.com",
		"password": "Password123!",
	}
	payload1, _ := json.Marshal(body1)
	req1 := httptest.NewRequest(http.MethodPost, "/v1/auth/oauth/register", bytes.NewReader(payload1))
	req1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("initial register failed: %d %s", rec1.Code, rec1.Body.String())
	}

	// Second registration with same email, different username
	body2 := map[string]string{
		"username": "user2",
		"email":    "duplicate@example.com",
		"password": "Password123!",
	}
	payload2, _ := json.Marshal(body2)
	req2 := httptest.NewRequest(http.MethodPost, "/v1/auth/oauth/register", bytes.NewReader(payload2))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected status 409 Conflict for duplicate email, got %d. Body: %s", rec2.Code, rec2.Body.String())
	}
	if !strings.Contains(rec2.Body.String(), "email already exists") {
		t.Fatalf("expected error message to mention email already exists, got: %s", rec2.Body.String())
	}
}

func TestRegisterOAuthUser_DuplicateUsername(t *testing.T) {
	router, _ := setupAuthRouter()

	// First registration
	body1 := map[string]string{
		"username": "sameusername",
		"email":    "user1@example.com",
		"password": "Password123!",
	}
	payload1, _ := json.Marshal(body1)
	req1 := httptest.NewRequest(http.MethodPost, "/v1/auth/oauth/register", bytes.NewReader(payload1))
	req1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("initial register failed: %d %s", rec1.Code, rec1.Body.String())
	}

	// Second registration with same username, different email
	body2 := map[string]string{
		"username": "sameusername",
		"email":    "user2@example.com",
		"password": "Password123!",
	}
	payload2, _ := json.Marshal(body2)
	req2 := httptest.NewRequest(http.MethodPost, "/v1/auth/oauth/register", bytes.NewReader(payload2))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected status 409 Conflict for duplicate username, got %d. Body: %s", rec2.Code, rec2.Body.String())
	}
	if !strings.Contains(rec2.Body.String(), "username already exists") {
		t.Fatalf("expected error message to mention username already exists, got: %s", rec2.Body.String())
	}
}

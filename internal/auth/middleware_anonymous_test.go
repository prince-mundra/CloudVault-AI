package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeAuthInfoChecker struct {
	version     int64
	isAnonymous bool
	err         error
}

func (f fakeAuthInfoChecker) GetAuthVersion(string) (int64, error) {
	return f.version, f.err
}

func (f fakeAuthInfoChecker) GetAuthInfo(string) (int64, bool, error) {
	return f.version, f.isAnonymous, f.err
}

func TestJWTAuthMiddlewareAnonymousValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "a-32-character-test-secret-value!!"
	rev := &fakeRevocationChecker{}

	tests := []struct {
		name         string
		tokenAnon    bool
		tokenVersion int64
		dbAnon       bool
		dbVersion    int64
		dbErr        error
		wantStatus   int
	}{
		{
			name:         "valid anonymous user matches db",
			tokenAnon:    true,
			tokenVersion: 1,
			dbAnon:       true,
			dbVersion:    1,
			wantStatus:   http.StatusNoContent,
		},
		{
			name:         "valid registered user matches db",
			tokenAnon:    false,
			tokenVersion: 1,
			dbAnon:       false,
			dbVersion:    1,
			wantStatus:   http.StatusNoContent,
		},
		{
			name:         "mismatch: token claims anonymous but db is registered",
			tokenAnon:    true,
			tokenVersion: 1,
			dbAnon:       false,
			dbVersion:    1,
			wantStatus:   http.StatusUnauthorized,
		},
		{
			name:         "mismatch: token claims registered but db is anonymous",
			tokenAnon:    false,
			tokenVersion: 1,
			dbAnon:       true,
			dbVersion:    1,
			wantStatus:   http.StatusUnauthorized,
		},
		{
			name:         "version mismatch: anonymous session expired",
			tokenAnon:    true,
			tokenVersion: 1,
			dbAnon:       true,
			dbVersion:    2,
			wantStatus:   http.StatusUnauthorized,
		},
		{
			name:         "account deleted: anonymous user not found in db",
			tokenAnon:    true,
			tokenVersion: 1,
			dbAnon:       true,
			dbVersion:    1,
			dbErr:        errors.New("user not found"),
			wantStatus:   http.StatusUnauthorized,
		},
		{
			name:         "legacy zero session version rejected",
			tokenAnon:    true,
			tokenVersion: 0,
			dbAnon:       true,
			dbVersion:    1,
			wantStatus:   http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, salt, err := GenerateTokenWithVersionAndAnonymous(secret, "user-uuid-1", tc.tokenVersion, tc.tokenAnon, time.Hour)
			if err != nil {
				t.Fatalf("failed to generate token: %v", err)
			}

			router := gin.New()
			router.Use(JWTAuthMiddleware(secret, rev, fakeAuthInfoChecker{
				version:     tc.dbVersion,
				isAnonymous: tc.dbAnon,
				err:         tc.dbErr,
			}))
			router.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-Salt", salt)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)

			if res.Code != tc.wantStatus {
				t.Fatalf("status %d, want %d; body: %s", res.Code, tc.wantStatus, res.Body.String())
			}
		})
	}
}

func TestAdminRoleMiddlewareRejectsAnonymousUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(SubjectKey, "anon-user-uuid")
		c.Next()
	})
	router.Use(AdminRoleMiddleware(fakeRoleChecker{role: "anonymous"}))
	router.GET("/admin", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest("GET", "/admin", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("status %d, want 403 Forbidden", res.Code)
	}
}

type legacyOnlyVersionChecker struct {
	version int64
}

func (l legacyOnlyVersionChecker) GetAuthVersion(string) (int64, error) {
	return l.version, nil
}

func TestJWTAuthMiddlewareFailsClosedWhenIdentityCheckerUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "a-32-character-test-secret-value!!"
	rev := &fakeRevocationChecker{}

	token, salt, err := GenerateTokenWithVersionAndAnonymous(secret, "user-uuid-1", 1, true, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	tests := []struct {
		name       string
		middleware gin.HandlerFunc
	}{
		{
			name:       "fails closed when checker is omitted",
			middleware: JWTAuthMiddleware(secret, rev),
		},
		{
			name:       "fails closed when checker is nil",
			middleware: JWTAuthMiddleware(secret, rev, nil),
		},
		{
			name:       "fails closed when checker does not implement UserAuthInfoChecker",
			middleware: JWTAuthMiddleware(secret, rev, legacyOnlyVersionChecker{version: 1}),
		},
		{
			name: "fails closed when database returns unexpected error",
			middleware: JWTAuthMiddleware(secret, rev, fakeAuthInfoChecker{
				err: errors.New("database connection refused"),
			}),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(tc.middleware)
			router.GET("/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })

			req := httptest.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-Salt", salt)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)

			if res.Code != http.StatusServiceUnavailable {
				t.Fatalf("status %d, want %d (ServiceUnavailable); body: %s", res.Code, http.StatusServiceUnavailable, res.Body.String())
			}
		})
	}
}

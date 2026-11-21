package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeRevocationChecker struct {
	revoked  bool
	err      error
	lastHash string
}

func (f *fakeRevocationChecker) IsRevoked(hash string) (bool, error) {
	f.lastHash = hash
	return f.revoked, f.err
}

func TestJWTAuthMiddlewareChecksRevocation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "a-32-character-test-secret-value!!"
	token, salt, err := GenerateTokenWithVersion(secret, "user-1", 1, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name       string
		checker    *fakeRevocationChecker
		wantStatus int
	}{
		{name: "valid", checker: &fakeRevocationChecker{}, wantStatus: http.StatusNoContent},
		{name: "revoked", checker: &fakeRevocationChecker{revoked: true}, wantStatus: http.StatusUnauthorized},
		{name: "checker unavailable", checker: &fakeRevocationChecker{err: errors.New("db unavailable")}, wantStatus: http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(JWTAuthMiddleware(secret, tc.checker))
			router.GET("/protected", func(ctx *gin.Context) { ctx.Status(http.StatusNoContent) })
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-Salt", salt)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", res.Code, tc.wantStatus)
			}
			if tc.checker.lastHash != TokenFingerprint(token) {
				t.Fatal("middleware did not check token fingerprint")
			}
		})
	}
}

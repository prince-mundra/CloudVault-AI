package auth

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeVersionChecker struct {
	version int64
	err     error
}

func (f fakeVersionChecker) GetAuthVersion(string) (int64, error) { return f.version, f.err }
func (f fakeVersionChecker) GetAuthInfo(string) (int64, bool, error) { return f.version, false, f.err }

func TestCORSMiddlewareAllowsOnlyConfiguredOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORSMiddleware([]string{"https://app.example"}))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	good := httptest.NewRequest("GET", "/", nil)
	good.Header.Set("Origin", "https://app.example")
	goodRes := httptest.NewRecorder()
	router.ServeHTTP(goodRes, good)
	if goodRes.Code != http.StatusNoContent || goodRes.Header().Get("Access-Control-Allow-Origin") != "https://app.example" {
		t.Fatalf("allowed origin response: %d %q", goodRes.Code, goodRes.Header().Get("Access-Control-Allow-Origin"))
	}
	bad := httptest.NewRequest("GET", "/", nil)
	bad.Header.Set("Origin", "https://evil.example")
	badRes := httptest.NewRecorder()
	router.ServeHTTP(badRes, bad)
	if badRes.Code != http.StatusForbidden {
		t.Fatalf("disallowed origin status=%d", badRes.Code)
	}
}

func TestIPRateLimiterConcurrentLimit(t *testing.T) {
	lim := NewIPRateLimiter(20, time.Minute)
	now := time.Now()
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, _ := lim.Allow("192.0.2.1", now)
			if ok {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if allowed != 20 {
		t.Fatalf("allowed %d requests; want 20", allowed)
	}
}

func TestJWTAuthMiddlewareEnforcesSessionVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "a-32-character-test-secret-value!!"
	rev := &fakeRevocationChecker{}
	for _, tc := range []struct {
		name                    string
		tokenVersion, dbVersion int64
		want                    int
	}{{"current", 2, 2, http.StatusNoContent}, {"stale", 1, 2, http.StatusUnauthorized}, {"legacy", 0, 2, http.StatusUnauthorized}} {
		t.Run(tc.name, func(t *testing.T) {
			token, salt, err := GenerateTokenWithVersion(secret, "user-1", tc.tokenVersion, time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			router := gin.New()
			router.Use(JWTAuthMiddleware(secret, rev, fakeVersionChecker{version: tc.dbVersion}))
			router.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-Salt", salt)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != tc.want {
				t.Fatalf("status %d, want %d", res.Code, tc.want)
			}
		})
	}
}

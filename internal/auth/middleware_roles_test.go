package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeRoleChecker struct {
	role string
	err  error
}

func (f fakeRoleChecker) GetUserRole(string) (string, error) { return f.role, f.err }

func TestAdminRoleMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name       string
		checker    fakeRoleChecker
		wantStatus int
	}{
		{name: "admin", checker: fakeRoleChecker{role: "admin"}, wantStatus: http.StatusNoContent},
		{name: "regular user", checker: fakeRoleChecker{role: "user"}, wantStatus: http.StatusForbidden},
		{name: "anonymous identity", checker: fakeRoleChecker{err: errors.New("user not found")}, wantStatus: http.StatusForbidden},
		{name: "database unavailable", checker: fakeRoleChecker{err: errors.New("db unavailable")}, wantStatus: http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(ctx *gin.Context) { ctx.Set(SubjectKey, "user-1"); ctx.Next() })
			router.Use(AdminRoleMiddleware(tc.checker))
			router.GET("/admin", func(ctx *gin.Context) { ctx.Status(http.StatusNoContent) })
			res := httptest.NewRecorder()
			router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/admin", nil))
			if res.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", res.Code, tc.wantStatus)
			}
		})
	}
}

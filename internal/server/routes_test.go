package server

import (
	"database/sql"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestInitRoutesRegistersWithoutConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	initRoutes(router, (*sql.DB)(nil), nil, "test-secret", 5<<30, 50<<20, []string{"http://localhost:5173"}, nil)
	if len(router.Routes()) == 0 {
		t.Fatal("expected API routes to be registered")
	}
	registered := map[string]bool{}
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = true
	}
	for _, route := range []string{"POST /v1/search/semantic", "GET /v1/files/:id/ingestion", "GET /v1/files/:id/summary"} {
		if !registered[route] {
			t.Fatalf("expected Phase 2 route %q", route)
		}
	}
}

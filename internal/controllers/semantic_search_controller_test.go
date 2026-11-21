package controllers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/souvik03-136/Go-Store/internal/aiclient"
	"github.com/souvik03-136/Go-Store/internal/auth"
)

type fakeSemanticSearcher struct {
	userID      string
	query       string
	limit       int
	contentType string
	err         error
	calls       int
}

func (f *fakeSemanticSearcher) Search(_ context.Context, userID, query string, limit int, contentType string) (*aiclient.SearchResponse, error) {
	f.calls++
	f.userID, f.query, f.limit, f.contentType = userID, query, limit, contentType
	if f.err != nil {
		return nil, f.err
	}
	return &aiclient.SearchResponse{Results: []aiclient.SearchHit{{FileID: "file-1", Text: "authorized snippet"}}}, nil
}

func semanticRouter(searcher SemanticSearcher) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	controller := NewSemanticSearchController(searcher)
	router.POST("/search", func(ctx *gin.Context) {
		ctx.Set(auth.SubjectKey, "authenticated-user")
		controller.Search(ctx)
	})
	return router
}

func TestSemanticSearchUsesAuthenticatedIdentityAndContentType(t *testing.T) {
	searcher := &fakeSemanticSearcher{}
	router := semanticRouter(searcher)
	request := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"query":"report","limit":5,"content_type":"application/pdf","user_id":"attacker"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	if searcher.calls != 1 || searcher.userID != "authenticated-user" || searcher.query != "report" || searcher.limit != 5 || searcher.contentType != "application/pdf" {
		t.Fatalf("unexpected search call: %+v", searcher)
	}
	if !strings.Contains(response.Body.String(), "authorized snippet") {
		t.Fatalf("search response missing result: %s", response.Body.String())
	}
}

func TestSemanticSearchRejectsInvalidContentTypeBeforeCallingService(t *testing.T) {
	searcher := &fakeSemanticSearcher{}
	router := semanticRouter(searcher)
	request := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"query":"report","content_type":"application/%"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusUnprocessableEntity, response.Body.String())
	}
	if searcher.calls != 0 {
		t.Fatal("invalid content_type reached the AI service")
	}
}

func TestSemanticSearchMapsUpstreamFailureToUnavailable(t *testing.T) {
	searcher := &fakeSemanticSearcher{err: errors.New("internal endpoint detail")}
	router := semanticRouter(searcher)
	request := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"query":"report"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if strings.Contains(response.Body.String(), "internal endpoint detail") {
		t.Fatal("upstream error leaked to the client")
	}
}

func TestSemanticSearchDisabledReturnsControlledUnavailable(t *testing.T) {
	// The server always wires a client; an empty URL/secret is its disabled state.
	router := semanticRouter(aiclient.New("", ""))
	request := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"query":"report"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusServiceUnavailable, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "AI retrieval is not configured") {
		t.Fatalf("response does not explain disabled AI retrieval: %s", response.Body.String())
	}
}

func TestValidContentTypePrefix(t *testing.T) {
	for _, value := range []string{"image/", "application/pdf", "text/plain"} {
		if !validContentTypePrefix(value) {
			t.Errorf("expected %q to be accepted", value)
		}
	}
	for _, value := range []string{"", "/pdf", "application/%", "text/plain; charset=utf-8", "application//pdf"} {
		if validContentTypePrefix(value) {
			t.Errorf("expected %q to be rejected", value)
		}
	}
}

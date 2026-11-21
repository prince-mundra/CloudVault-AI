package controllers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/souvik03-136/Go-Store/internal/auth"
	"github.com/souvik03-136/Go-Store/internal/repository"
)

type fakeIngestionStatusReader struct {
	fileID  string
	ownerID string
	status  *repository.IngestionStatus
	err     error
}

func (f *fakeIngestionStatusReader) GetStatus(_ context.Context, fileID, ownerID string) (*repository.IngestionStatus, error) {
	f.fileID, f.ownerID = fileID, ownerID
	return f.status, f.err
}

func ingestionRouter(reader IngestionStatusReader, authenticated bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	controller := NewIngestionController(reader)
	router.GET("/files/:id/ingestion", func(ctx *gin.Context) {
		if authenticated {
			ctx.Set(auth.SubjectKey, "owner-1")
		}
		controller.GetStatus(ctx)
	})
	return router
}

func TestIngestionStatusUsesAuthenticatedOwner(t *testing.T) {
	reader := &fakeIngestionStatusReader{status: &repository.IngestionStatus{State: "succeeded", Attempts: 1, UpdatedAt: time.Now()}}
	response := httptest.NewRecorder()
	ingestionRouter(reader, true).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/files/file-1/ingestion", nil))
	if response.Code != http.StatusOK || reader.fileID != "file-1" || reader.ownerID != "owner-1" {
		t.Fatalf("status=%d lookup=(%q,%q) body=%s", response.Code, reader.fileID, reader.ownerID, response.Body.String())
	}
}

func TestIngestionStatusRequiresSubject(t *testing.T) {
	reader := &fakeIngestionStatusReader{}
	response := httptest.NewRecorder()
	ingestionRouter(reader, false).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/files/file-1/ingestion", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if reader.fileID != "" {
		t.Fatal("status lookup ran without an authenticated subject")
	}
}

func TestIngestionStatusSanitizesRepositoryErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want int
	}{
		{name: "not found", err: repository.ErrIngestionStatusNotFound, want: http.StatusNotFound},
		{name: "database failure", err: errors.New("database password leaked"), want: http.StatusServiceUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			reader := &fakeIngestionStatusReader{err: test.err}
			response := httptest.NewRecorder()
			ingestionRouter(reader, true).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/files/file-1/ingestion", nil))
			if response.Code != test.want {
				t.Fatalf("status = %d, want %d", response.Code, test.want)
			}
			if strings.Contains(response.Body.String(), "database password leaked") {
				t.Fatal("repository error leaked to response")
			}
		})
	}
}

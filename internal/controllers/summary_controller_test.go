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

type fakeSummaryStatusReader struct {
	fileID string
	userID string
	status *repository.SummaryStatus
	err    error
}

func (f *fakeSummaryStatusReader) GetStatus(_ context.Context, fileID, userID string) (*repository.SummaryStatus, error) {
	f.fileID, f.userID = fileID, userID
	return f.status, f.err
}

func summaryRouter(reader SummaryStatusReader, authenticated bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	controller := NewSummaryController(reader)
	router.GET("/files/:id/summary", func(ctx *gin.Context) {
		if authenticated {
			ctx.Set(auth.SubjectKey, "reader-1")
		}
		controller.GetStatus(ctx)
	})
	return router
}

func TestSummaryStatusUsesAuthenticatedUser(t *testing.T) {
	reader := &fakeSummaryStatusReader{status: &repository.SummaryStatus{State: "succeeded", Attempts: 1, Summary: "A concise summary", Model: "llama3.2:3b", UpdatedAt: time.Now()}}
	response := httptest.NewRecorder()
	summaryRouter(reader, true).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/files/file-1/summary", nil))
	if response.Code != http.StatusOK || reader.fileID != "file-1" || reader.userID != "reader-1" || !strings.Contains(response.Body.String(), "A concise summary") {
		t.Fatalf("status=%d lookup=(%q,%q) body=%s", response.Code, reader.fileID, reader.userID, response.Body.String())
	}
}

func TestSummaryStatusRequiresSubject(t *testing.T) {
	reader := &fakeSummaryStatusReader{}
	response := httptest.NewRecorder()
	summaryRouter(reader, false).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/files/file-1/summary", nil))
	if response.Code != http.StatusUnauthorized || reader.fileID != "" {
		t.Fatalf("status=%d lookup=%q body=%s", response.Code, reader.fileID, response.Body.String())
	}
}

func TestSummaryStatusSanitizesRepositoryErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want int
	}{
		{name: "not found", err: repository.ErrSummaryStatusNotFound, want: http.StatusNotFound},
		{name: "database failure", err: errors.New("database password leaked"), want: http.StatusServiceUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			reader := &fakeSummaryStatusReader{err: test.err}
			response := httptest.NewRecorder()
			summaryRouter(reader, true).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/files/file-1/summary", nil))
			if response.Code != test.want || strings.Contains(response.Body.String(), "database password leaked") {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

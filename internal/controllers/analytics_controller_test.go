package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/souvik03-136/Go-Store/internal/auth"
	"github.com/souvik03-136/Go-Store/internal/models"
	"github.com/souvik03-136/Go-Store/internal/services"
)

type mockStorageAnalyticsRepo struct {
	analytics *models.StorageAnalytics
	err       error
}

func (m *mockStorageAnalyticsRepo) GetStorageAnalytics(ownerID string) (*models.StorageAnalytics, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.analytics != nil {
		copy := *m.analytics
		return &copy, nil
	}
	return &models.StorageAnalytics{
		TotalFiles: 5,
		UsedBytes:  1024 * 1024,
	}, nil
}

func TestAnalyticsController_RequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := services.NewAnalyticsService(&mockStorageAnalyticsRepo{}, 5000000000)
	ctrl := NewAnalyticsController(svc)

	r := gin.New()
	r.GET("/analytics/storage", ctrl.GetStorage)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/analytics/storage", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAnalyticsController_ReturnsStorageStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockStorageAnalyticsRepo{
		analytics: &models.StorageAnalytics{
			TotalFiles: 42,
			UsedBytes:  1048576,
		},
	}
	svc := services.NewAnalyticsService(repo, 5368709120) // 5GB
	ctrl := NewAnalyticsController(svc)

	r := gin.New()
	r.GET("/analytics/storage", func(c *gin.Context) {
		c.Set(auth.SubjectKey, "user-abc")
		ctrl.GetStorage(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/analytics/storage", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Data models.StorageAnalytics `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Data.TotalFiles != 42 || resp.Data.UsedBytes != 1048576 || resp.Data.QuotaBytes != 5368709120 {
		t.Fatalf("unexpected stats: %+v", resp.Data)
	}
}

func TestAnalyticsController_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockStorageAnalyticsRepo{
		err: errors.New("db error"),
	}
	svc := services.NewAnalyticsService(repo, 5368709120)
	ctrl := NewAnalyticsController(svc)

	r := gin.New()
	r.GET("/analytics/storage", func(c *gin.Context) {
		c.Set(auth.SubjectKey, "user-abc")
		ctrl.GetStorage(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/analytics/storage", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

package services

import (
	"errors"
	"testing"

	"github.com/souvik03-136/Go-Store/internal/models"
)

type fakeStorageAnalyticsRepo struct {
	result  *models.StorageAnalytics
	err     error
	ownerID string
}

func (f *fakeStorageAnalyticsRepo) GetStorageAnalytics(ownerID string) (*models.StorageAnalytics, error) {
	f.ownerID = ownerID
	return f.result, f.err
}

func TestGetStorageAnalyticsQuota(t *testing.T) {
	repo := &fakeStorageAnalyticsRepo{result: &models.StorageAnalytics{TotalFiles: 3, UsedBytes: 600}}
	stats, err := NewAnalyticsService(repo, 1000).GetStorageAnalytics("owner-1")
	if err != nil {
		t.Fatal(err)
	}
	if stats.QuotaBytes != 1000 || stats.RemainingBytes != 400 {
		t.Fatalf("quota values = (%d,%d)", stats.QuotaBytes, stats.RemainingBytes)
	}
	if repo.ownerID != "owner-1" {
		t.Fatalf("owner passed = %q", repo.ownerID)
	}
}

func TestGetStorageAnalyticsClampsOverQuota(t *testing.T) {
	repo := &fakeStorageAnalyticsRepo{result: &models.StorageAnalytics{UsedBytes: 1200}}
	stats, err := NewAnalyticsService(repo, 1000).GetStorageAnalytics("owner-1")
	if err != nil {
		t.Fatal(err)
	}
	if stats.RemainingBytes != 0 {
		t.Fatalf("remaining = %d, want 0", stats.RemainingBytes)
	}
}

func TestGetStorageAnalyticsErrors(t *testing.T) {
	repo := &fakeStorageAnalyticsRepo{err: errors.New("database offline")}
	if _, err := NewAnalyticsService(repo, 1000).GetStorageAnalytics("owner-1"); err == nil {
		t.Fatal("expected repository error")
	}
	if _, err := NewAnalyticsService(repo, 1000).GetStorageAnalytics(""); err == nil {
		t.Fatal("expected missing owner error")
	}
}

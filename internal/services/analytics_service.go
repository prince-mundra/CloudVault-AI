package services

import (
	"errors"

	"github.com/souvik03-136/Go-Store/internal/models"
)

type StorageAnalyticsRepository interface {
	GetStorageAnalytics(ownerID string) (*models.StorageAnalytics, error)
}

type AnalyticsService struct {
	repo       StorageAnalyticsRepository
	quotaBytes int64
}

func NewAnalyticsService(repo StorageAnalyticsRepository, quotaBytes int64) *AnalyticsService {
	return &AnalyticsService{repo: repo, quotaBytes: quotaBytes}
}

func (s *AnalyticsService) GetStorageAnalytics(ownerID string) (*models.StorageAnalytics, error) {
	if ownerID == "" {
		return nil, errors.New("owner id is required")
	}
	stats, err := s.repo.GetStorageAnalytics(ownerID)
	if err != nil {
		return nil, err
	}
	stats.QuotaBytes = s.quotaBytes
	stats.RemainingBytes = s.quotaBytes - stats.UsedBytes - stats.ReservedBytes
	if stats.RemainingBytes < 0 {
		stats.RemainingBytes = 0
	}
	return stats, nil
}

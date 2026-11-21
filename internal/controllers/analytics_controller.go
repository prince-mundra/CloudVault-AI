package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/souvik03-136/Go-Store/internal/merrors"
	"github.com/souvik03-136/Go-Store/internal/services"
)

type AnalyticsController struct{ analyticsSvc *services.AnalyticsService }

func NewAnalyticsController(analyticsSvc *services.AnalyticsService) *AnalyticsController {
	return &AnalyticsController{analyticsSvc: analyticsSvc}
}

func (c *AnalyticsController) GetStorage(ctx *gin.Context) {
	ownerID := requesterID(ctx)
	if ownerID == "" {
		merrors.Unauthorized(ctx, "authentication required")
		return
	}
	stats, err := c.analyticsSvc.GetStorageAnalytics(ownerID)
	if err != nil {
		merrors.InternalServer(ctx, "failed to load storage analytics")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": stats})
}

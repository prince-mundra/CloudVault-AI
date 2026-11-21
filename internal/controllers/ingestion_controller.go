package controllers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/souvik03-136/Go-Store/internal/auth"
	"github.com/souvik03-136/Go-Store/internal/merrors"
	"github.com/souvik03-136/Go-Store/internal/repository"
)

type IngestionStatusReader interface {
	GetStatus(ctx context.Context, fileID, ownerID string) (*repository.IngestionStatus, error)
}

type IngestionController struct{ statuses IngestionStatusReader }

func NewIngestionController(statuses IngestionStatusReader) *IngestionController {
	return &IngestionController{statuses: statuses}
}

func (c *IngestionController) GetStatus(ctx *gin.Context) {
	owner, ok := ctx.Get(auth.SubjectKey)
	ownerID, valid := owner.(string)
	if !ok || !valid || ownerID == "" {
		merrors.Unauthorized(ctx, "authentication required")
		return
	}
	status, err := c.statuses.GetStatus(ctx.Request.Context(), ctx.Param("id"), ownerID)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		if errors.Is(err, repository.ErrIngestionStatusNotFound) {
			merrors.NotFound(ctx, "file or ingestion status not found")
			return
		}
		merrors.ServiceUnavailable(ctx, "ingestion status unavailable")
		return
	}
	ctx.JSON(http.StatusOK, status)
}

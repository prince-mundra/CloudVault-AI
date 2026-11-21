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

type SummaryStatusReader interface {
	GetStatus(ctx context.Context, fileID, userID string) (*repository.SummaryStatus, error)
}

type SummaryController struct{ statuses SummaryStatusReader }

func NewSummaryController(statuses SummaryStatusReader) *SummaryController {
	return &SummaryController{statuses: statuses}
}

func (c *SummaryController) GetStatus(ctx *gin.Context) {
	user, ok := ctx.Get(auth.SubjectKey)
	userID, valid := user.(string)
	if !ok || !valid || userID == "" {
		merrors.Unauthorized(ctx, "authentication required")
		return
	}
	status, err := c.statuses.GetStatus(ctx.Request.Context(), ctx.Param("id"), userID)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		if errors.Is(err, repository.ErrSummaryStatusNotFound) {
			merrors.NotFound(ctx, "file or summary status not found")
			return
		}
		merrors.ServiceUnavailable(ctx, "document summary status unavailable")
		return
	}
	ctx.JSON(http.StatusOK, status)
}

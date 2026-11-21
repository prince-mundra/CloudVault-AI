package controllers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/souvik03-136/Go-Store/internal/aiclient"
	"github.com/souvik03-136/Go-Store/internal/auth"
	"github.com/souvik03-136/Go-Store/internal/merrors"
)

type SemanticSearcher interface {
	Search(context.Context, string, string, int, string) (*aiclient.SearchResponse, error)
}

type SemanticSearchController struct{ searcher SemanticSearcher }

func NewSemanticSearchController(searcher SemanticSearcher) *SemanticSearchController {
	return &SemanticSearchController{searcher: searcher}
}

func (c *SemanticSearchController) Search(ctx *gin.Context) {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 16<<10)
	var body struct {
		Query       string `json:"query" binding:"required"`
		Limit       int    `json:"limit"`
		ContentType string `json:"content_type"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		merrors.BadRequest(ctx, "query is required")
		return
	}
	body.Query = strings.TrimSpace(body.Query)
	body.ContentType = strings.TrimSpace(body.ContentType)
	if len([]rune(body.Query)) < 2 || len([]rune(body.Query)) > 1000 {
		merrors.Validation(ctx, "query must be between 2 and 1000 characters")
		return
	}
	if len([]rune(body.ContentType)) > 128 || (body.ContentType != "" && !validContentTypePrefix(body.ContentType)) {
		merrors.Validation(ctx, "content_type must be a MIME type or type prefix such as image/")
		return
	}
	if body.Limit == 0 {
		body.Limit = 10
	}
	if body.Limit < 1 || body.Limit > 20 {
		merrors.Validation(ctx, "limit must be between 1 and 20")
		return
	}
	userID, ok := ctx.Get(auth.SubjectKey)
	id, valid := userID.(string)
	if !ok || !valid || id == "" {
		merrors.Unauthorized(ctx, "authentication required")
		return
	}
	if c.searcher == nil {
		merrors.ServiceUnavailable(ctx, "AI retrieval is not configured")
		return
	}
	result, err := c.searcher.Search(ctx.Request.Context(), id, body.Query, body.Limit, body.ContentType)
	if err != nil {
		if errors.Is(err, aiclient.ErrNotConfigured) {
			merrors.ServiceUnavailable(ctx, "AI retrieval is not configured")
			return
		}
		merrors.ServiceUnavailable(ctx, "AI retrieval is unavailable")
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func validContentTypePrefix(value string) bool {
	parts := strings.Split(value, "/")
	if len(parts) != 2 || parts[0] == "" {
		return false
	}
	validToken := func(token string) bool {
		for _, ch := range token {
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || strings.ContainsRune("!#$&^_.+-", ch) {
				continue
			}
			return false
		}
		return true
	}
	return validToken(parts[0]) && validToken(parts[1])
}

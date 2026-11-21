package controllers

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/souvik03-136/Go-Store/internal/merrors"
	"github.com/souvik03-136/Go-Store/internal/services"
)

type ShareLinkController struct{ service *services.ShareLinkService }

func NewShareLinkController(service *services.ShareLinkService) *ShareLinkController {
	return &ShareLinkController{service: service}
}

type createShareLinkRequest struct {
	Password         string `json:"password"`
	ExpiresInSeconds int64  `json:"expires_in_seconds"`
}

func (c *ShareLinkController) Create(ctx *gin.Context) {
	var req createShareLinkRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		merrors.Validation(ctx, err.Error())
		return
	}
	if req.ExpiresInSeconds < 0 || req.ExpiresInSeconds > int64((30*24*time.Hour)/time.Second) {
		merrors.Validation(ctx, "expires_in_seconds must be between 0 and 30 days")
		return
	}
	lifetime := time.Duration(req.ExpiresInSeconds) * time.Second
	link, token, err := c.service.Create(ctx.Request.Context(), requesterID(ctx), ctx.Param("id"), req.Password, lifetime)
	if err != nil {
		if err.Error() == "permission denied" {
			merrors.Forbidden(ctx, "only the owner may create share links")
		} else if err.Error() == "file not found" {
			merrors.NotFound(ctx, "file not found")
		} else if strings.Contains(err.Error(), "lifetime") || strings.Contains(err.Error(), "share password") {
			merrors.Validation(ctx, err.Error())
		} else {
			merrors.InternalServer(ctx, "share link creation failed")
		}
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"data": link, "token": token, "download_path": "/v1/public/share-links/" + token + "/download"})
}
func (c *ShareLinkController) List(ctx *gin.Context) {
	items, err := c.service.List(ctx.Request.Context(), requesterID(ctx), ctx.Param("id"))
	if err != nil {
		if err.Error() == "permission denied" {
			merrors.Forbidden(ctx, "only the owner may list share links")
		} else if err.Error() == "file not found" {
			merrors.NotFound(ctx, "file not found")
		} else {
			merrors.InternalServer(ctx, "share link listing failed")
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": items})
}
func (c *ShareLinkController) Revoke(ctx *gin.Context) {
	if err := c.service.Revoke(ctx.Request.Context(), requesterID(ctx), ctx.Param("id"), ctx.Param("link_id")); err != nil {
		if err.Error() == "permission denied" {
			merrors.Forbidden(ctx, "only the owner may revoke share links")
		} else if err.Error() == "share link not found" {
			merrors.NotFound(ctx, "share link not found")
		} else {
			merrors.InternalServer(ctx, "share link revocation failed")
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "share link revoked"})
}
func (c *ShareLinkController) Download(ctx *gin.Context) {
	file, reader, err := c.service.Download(ctx.Request.Context(), ctx.Param("token"), ctx.GetHeader("X-Share-Password"))
	if err != nil {
		if errors.Is(err, context.Canceled) {
			ctx.AbortWithStatus(http.StatusRequestTimeout)
			return
		}
		if strings.Contains(err.Error(), "password") {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "share password required or invalid"})
		} else if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "unavailable") || strings.Contains(err.Error(), "invalid") {
			ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "share link unavailable"})
		} else {
			merrors.InternalServer(ctx, "shared download failed")
		}
		return
	}
	defer reader.Close()
	contentType := file.ContentType
	if _, _, err := mime.ParseMediaType(contentType); err != nil {
		contentType = "application/octet-stream"
	}
	filename := path.Base(strings.ReplaceAll(file.Name, "\\", "/"))
	ctx.DataFromReader(http.StatusOK, file.Size, contentType, reader, map[string]string{"Content-Disposition": mime.FormatMediaType("attachment", map[string]string{"filename": filename}), "X-Content-Type-Options": "nosniff", "Cache-Control": "no-store"})
}

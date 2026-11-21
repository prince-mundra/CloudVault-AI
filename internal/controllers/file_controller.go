// internal/controllers/file_controller.go
package controllers

import (
	"errors"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/souvik03-136/Go-Store/internal/auth"
	"github.com/souvik03-136/Go-Store/internal/merrors"
	"github.com/souvik03-136/Go-Store/internal/repository"
	"github.com/souvik03-136/Go-Store/internal/services"
)

type FileController struct {
	fileSvc        *services.FileService
	maxUploadBytes int64
}

func NewFileController(fileSvc *services.FileService, maxUploadBytes int64) *FileController {
	return &FileController{fileSvc: fileSvc, maxUploadBytes: maxUploadBytes}
}

type updateFileRequest struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
}
type moveFileRequest struct {
	FolderID *string `json:"folder_id"`
}

func requesterID(ctx *gin.Context) string {
	v, ok := ctx.Get(auth.SubjectKey)
	if !ok {
		return ""
	}
	id, _ := v.(string)
	return id
}

func (c *FileController) CreateFile(ctx *gin.Context) {
	owner := requesterID(ctx)
	if owner == "" {
		merrors.Unauthorized(ctx, "authentication required")
		return
	}
	// The body cap includes bounded multipart headers/boundaries; the file itself
	// remains limited independently by the configured MAX_UPLOAD_BYTES value.
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, c.maxUploadBytes+(1<<20))
	h, err := ctx.FormFile("file")
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "request body too large") {
			ctx.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "multipart request exceeds upload limit plus form overhead"})
			return
		}
		merrors.BadRequest(ctx, "a file field is required")
		return
	}
	if h.Size > c.maxUploadBytes {
		ctx.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file exceeds configured upload limit"})
		return
	}
	file, duplicate, err := c.fileSvc.UploadFile(ctx.Request.Context(), h, owner)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrQuotaExceeded):
			ctx.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "storage quota exceeded"})
		case errors.Is(err, repository.ErrUploadInProgress):
			ctx.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "identical upload already in progress"})
		case strings.Contains(err.Error(), "configured upload limit"), strings.Contains(err.Error(), "size does not match"), strings.Contains(err.Error(), "file is required"):
			merrors.BadRequest(ctx, err.Error())
		default:
			merrors.InternalServer(ctx, "file upload failed")
		}
		return
	}
	status := http.StatusCreated
	if duplicate {
		status = http.StatusOK
	}
	ctx.JSON(status, gin.H{"data": file, "deduplicated": duplicate})
}
func (c *FileController) GetFileByID(ctx *gin.Context) {
	f, err := c.fileSvc.GetAccessibleFileByID(ctx.Param("id"), requesterID(ctx), "read")
	if err != nil {
		if err.Error() == "permission denied" {
			merrors.Forbidden(ctx, "you do not have access to this file")
		} else {
			merrors.NotFound(ctx, "file not found")
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": f})
}
func (c *FileController) DownloadFile(ctx *gin.Context) {
	f, r, err := c.fileSvc.DownloadOwnedFile(ctx.Request.Context(), ctx.Param("id"), requesterID(ctx))
	if err != nil {
		if err.Error() == "permission denied" {
			merrors.Forbidden(ctx, "you do not have access to this file")
			return
		}
		if err.Error() == "file not found" {
			merrors.NotFound(ctx, "file not found")
			return
		}
		merrors.InternalServer(ctx, "file download failed")
		return
	}
	defer r.Close()
	contentType := f.ContentType
	if _, _, err := mime.ParseMediaType(contentType); err != nil {
		contentType = "application/octet-stream"
	}
	filename := path.Base(strings.ReplaceAll(f.Name, "\\", "/"))
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
	ctx.DataFromReader(http.StatusOK, f.Size, contentType, r, map[string]string{"Content-Disposition": disposition, "X-Content-Type-Options": "nosniff"})
}
func (c *FileController) MoveToFolder(ctx *gin.Context) {
	var req moveFileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		merrors.Validation(ctx, err.Error())
		return
	}
	f, err := c.fileSvc.MoveToFolder(requesterID(ctx), ctx.Param("id"), req.FolderID)
	if err != nil {
		switch err.Error() {
		case "permission denied":
			merrors.Forbidden(ctx, "you do not own this file")
		case "file not found", "folder not found":
			merrors.NotFound(ctx, err.Error())
		default:
			merrors.InternalServer(ctx, "file move failed")
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": f})
}
func (c *FileController) ListMyFiles(ctx *gin.Context) {
	owner := requesterID(ctx)
	if owner == "" {
		merrors.Unauthorized(ctx, "authentication required")
		return
	}
	limit, offset := 50, 0
	var err error
	if raw := ctx.Query("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			merrors.Validation(ctx, "limit must be between 1 and 100")
			return
		}
	}
	if raw := ctx.Query("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 {
			merrors.Validation(ctx, "offset must be a non-negative integer")
			return
		}
	}
	var folderID *string
	if raw, ok := ctx.GetQuery("folder_id"); ok {
		if raw == "root" {
			raw = ""
		}
		folderID = &raw
	}
	files, err := c.fileSvc.SearchFilesByOwner(owner, ctx.Query("search"), ctx.Query("content_type"), ctx.Query("sort"), ctx.Query("order"), folderID, limit, offset)
	if err != nil {
		if err.Error() == "folder not found" {
			merrors.NotFound(ctx, err.Error())
			return
		}
		if strings.HasPrefix(err.Error(), "sort must") || strings.HasPrefix(err.Error(), "order must") {
			merrors.Validation(ctx, err.Error())
			return
		}
		merrors.InternalServer(ctx, "failed to list files")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": files, "pagination": gin.H{"limit": limit, "offset": offset}})
}
func (c *FileController) UpdateFile(ctx *gin.Context) {
	var req updateFileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		merrors.Validation(ctx, err.Error())
		return
	}
	f, err := c.fileSvc.UpdateFile(ctx.Param("id"), requesterID(ctx), req.Name, req.ContentType, 0)
	if err != nil {
		switch err.Error() {
		case "permission denied":
			merrors.Forbidden(ctx, "you do not have write access")
		case "file not found":
			merrors.NotFound(ctx, "file not found")
		default:
			merrors.Validation(ctx, err.Error())
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": f})
}
func (c *FileController) DeleteFile(ctx *gin.Context) {
	if err := c.fileSvc.DeleteFile(ctx.Request.Context(), ctx.Param("id"), requesterID(ctx)); err != nil {
		switch err.Error() {
		case "permission denied":
			merrors.Forbidden(ctx, "only the file owner may permanently delete it")
		case "file not found":
			merrors.NotFound(ctx, "file not found")
		default:
			merrors.InternalServer(ctx, "failed to queue file deletion")
		}
		return
	}
	ctx.JSON(http.StatusAccepted, gin.H{"message": "file deletion queued; storage cleanup will be retried if unavailable"})
}

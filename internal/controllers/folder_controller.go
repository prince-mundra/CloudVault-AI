package controllers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/souvik03-136/Go-Store/internal/merrors"
	"github.com/souvik03-136/Go-Store/internal/services"
)

type FolderController struct{ service *services.FolderService }

func NewFolderController(service *services.FolderService) *FolderController {
	return &FolderController{service: service}
}

type folderRequest struct {
	Name     string  `json:"name"`
	ParentID *string `json:"parent_id"`
}

type folderUpdateRequest struct {
	Name     string          `json:"name"`
	ParentID json.RawMessage `json:"parent_id"`
}

func (c *FolderController) Create(ctx *gin.Context) {
	var req folderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		merrors.Validation(ctx, err.Error())
		return
	}
	if req.ParentID != nil && strings.TrimSpace(*req.ParentID) == "" {
		req.ParentID = nil
	}
	f, err := c.service.Create(requesterID(ctx), req.Name, req.ParentID)
	if err != nil {
		if strings.Contains(err.Error(), "parent folder not found") {
			merrors.NotFound(ctx, "parent folder not found")
			return
		}
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key") {
			merrors.Conflict(ctx, "a folder with that name already exists here")
			return
		}
		merrors.Validation(ctx, err.Error())
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"data": f})
}

func (c *FolderController) List(ctx *gin.Context) {
	var parentID *string
	if raw := ctx.Query("parent_id"); raw != "" {
		parentID = &raw
	}
	folders, err := c.service.List(requesterID(ctx), parentID)
	if err != nil {
		merrors.NotFound(ctx, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": folders})
}

func (c *FolderController) Update(ctx *gin.Context) {
	var req folderUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		merrors.Validation(ctx, err.Error())
		return
	}
	moveParent := len(req.ParentID) != 0
	var parentID *string
	if moveParent && string(req.ParentID) != "null" {
		var parent string
		if err := json.Unmarshal(req.ParentID, &parent); err != nil {
			merrors.Validation(ctx, "parent_id must be a folder ID or null")
			return
		}
		parent = strings.TrimSpace(parent)
		if parent != "" {
			parentID = &parent
		}
	}
	f, err := c.service.Update(requesterID(ctx), ctx.Param("id"), req.Name, parentID, moveParent)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			merrors.NotFound(ctx, err.Error())
			return
		}
		if strings.Contains(err.Error(), "cycle") || strings.Contains(err.Error(), "own parent") {
			merrors.Validation(ctx, err.Error())
			return
		}
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key") {
			merrors.Conflict(ctx, err.Error())
			return
		}
		merrors.InternalServer(ctx, "folder update failed")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": f})
}

func (c *FolderController) Delete(ctx *gin.Context) {
	if err := c.service.Delete(requesterID(ctx), ctx.Param("id")); err != nil {
		if strings.Contains(err.Error(), "not found") {
			merrors.NotFound(ctx, "folder not found")
			return
		}
		if strings.Contains(err.Error(), "not empty") {
			merrors.Conflict(ctx, "folder must be empty before deletion")
			return
		}
		merrors.InternalServer(ctx, "folder deletion failed")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "folder deleted"})
}

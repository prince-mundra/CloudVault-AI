package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/souvik03-136/Go-Store/internal/merrors"
	"github.com/souvik03-136/Go-Store/internal/services"
)

type SharingController struct{ service *services.SharingService }

func NewSharingController(service *services.SharingService) *SharingController {
	return &SharingController{service: service}
}

type grantShareRequest struct {
	UserID    string `json:"user_id" binding:"required"`
	CanRead   bool   `json:"can_read"`
	CanWrite  bool   `json:"can_write"`
	CanDelete bool   `json:"can_delete"`
}

func (c *SharingController) Grant(ctx *gin.Context) {
	var req grantShareRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		merrors.Validation(ctx, err.Error())
		return
	}
	p, err := c.service.Grant(requesterID(ctx), ctx.Param("id"), req.UserID, req.CanRead, req.CanWrite, req.CanDelete)
	if err != nil {
		switch err.Error() {
		case "permission denied":
			merrors.Forbidden(ctx, "only the owner may change shares")
		case "recipient not found":
			merrors.NotFound(ctx, "recipient not found")
		default:
			merrors.Validation(ctx, err.Error())
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": p})
}

func (c *SharingController) List(ctx *gin.Context) {
	permissions, err := c.service.List(requesterID(ctx), ctx.Param("id"))
	if err != nil {
		if err.Error() == "permission denied" {
			merrors.Forbidden(ctx, "only the owner may view shares")
		} else {
			merrors.NotFound(ctx, "file not found")
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": permissions})
}

func (c *SharingController) Revoke(ctx *gin.Context) {
	if err := c.service.Revoke(requesterID(ctx), ctx.Param("id"), ctx.Param("user_id")); err != nil {
		if err.Error() == "permission denied" {
			merrors.Forbidden(ctx, "only the owner may revoke shares")
		} else {
			merrors.NotFound(ctx, "share or file not found")
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "share revoked"})
}

func (c *SharingController) SharedWithMe(ctx *gin.Context) {
	files, err := c.service.SharedWithMe(requesterID(ctx))
	if err != nil {
		merrors.InternalServer(ctx, "failed to list shared files")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": files})
}

// Leave removes only the authenticated recipient's own share reference.
func (c *SharingController) Leave(ctx *gin.Context) {
	if err := c.service.Leave(requesterID(ctx), ctx.Param("id")); err != nil {
		if err.Error() == "permission denied" || err.Error() == "owner cannot remove own reference" {
			merrors.Forbidden(ctx, "only a recipient may remove their own shared reference")
		} else {
			merrors.NotFound(ctx, "shared file not found")
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "shared reference removed"})
}

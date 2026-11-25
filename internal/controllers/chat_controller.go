package controllers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/souvik03-136/Go-Store/internal/merrors"
	"github.com/souvik03-136/Go-Store/internal/repository"
	"github.com/souvik03-136/Go-Store/internal/services"
)

type ChatService interface {
	SendMessage(ctx context.Context, userID, conversationID, message string, fileIDs []string) (*services.ChatTurnResult, error)
	ListConversations(ctx context.Context, userID string) ([]*repository.Conversation, error)
	GetConversation(ctx context.Context, id, userID string) (*repository.ConversationWithMessages, error)
	CreateConversation(ctx context.Context, userID, title string) (*repository.Conversation, error)
	DeleteConversation(ctx context.Context, id, userID string) error
}

type ChatController struct {
	svc ChatService
}

func NewChatController(svc ChatService) *ChatController {
	return &ChatController{svc: svc}
}

func (c *ChatController) SendMessage(ctx *gin.Context) {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 64<<10)
	var body struct {
		ConversationID string   `json:"conversation_id"`
		Message        string   `json:"message" binding:"required"`
		FileIDs        []string `json:"file_ids"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		merrors.BadRequest(ctx, "message is required")
		return
	}
	userID := requesterID(ctx)
	if userID == "" {
		merrors.Unauthorized(ctx, "authentication required")
		return
	}

	result, err := c.svc.SendMessage(ctx.Request.Context(), userID, body.ConversationID, body.Message, body.FileIDs)
	if err != nil {
		if errors.Is(err, services.ErrInvalidMessage) {
			merrors.Validation(ctx, err.Error())
			return
		}
		if errors.Is(err, repository.ErrConversationNotFound) {
			merrors.NotFound(ctx, "conversation not found")
			return
		}
		merrors.ServiceUnavailable(ctx, "AI assistant unavailable")
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *ChatController) ListConversations(ctx *gin.Context) {
	userID := requesterID(ctx)
	if userID == "" {
		merrors.Unauthorized(ctx, "authentication required")
		return
	}
	convs, err := c.svc.ListConversations(ctx.Request.Context(), userID)
	if err != nil {
		merrors.InternalServer(ctx, "failed to list conversations")
		return
	}
	if convs == nil {
		convs = []*repository.Conversation{}
	}
	ctx.JSON(http.StatusOK, gin.H{"data": convs})
}

func (c *ChatController) GetConversation(ctx *gin.Context) {
	userID := requesterID(ctx)
	if userID == "" {
		merrors.Unauthorized(ctx, "authentication required")
		return
	}
	id := ctx.Param("id")
	if strings.TrimSpace(id) == "" {
		merrors.BadRequest(ctx, "conversation id is required")
		return
	}
	conv, err := c.svc.GetConversation(ctx.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrConversationNotFound) {
			merrors.NotFound(ctx, "conversation not found")
			return
		}
		merrors.InternalServer(ctx, "failed to read conversation")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": conv})
}

func (c *ChatController) CreateConversation(ctx *gin.Context) {
	userID := requesterID(ctx)
	if userID == "" {
		merrors.Unauthorized(ctx, "authentication required")
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	_ = ctx.ShouldBindJSON(&body)
	conv, err := c.svc.CreateConversation(ctx.Request.Context(), userID, body.Title)
	if err != nil {
		merrors.InternalServer(ctx, "failed to create conversation")
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"data": conv})
}

func (c *ChatController) DeleteConversation(ctx *gin.Context) {
	userID := requesterID(ctx)
	if userID == "" {
		merrors.Unauthorized(ctx, "authentication required")
		return
	}
	id := ctx.Param("id")
	if strings.TrimSpace(id) == "" {
		merrors.BadRequest(ctx, "conversation id is required")
		return
	}
	err := c.svc.DeleteConversation(ctx.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrConversationNotFound) {
			merrors.NotFound(ctx, "conversation not found")
			return
		}
		merrors.InternalServer(ctx, "failed to delete conversation")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "conversation deleted"})
}

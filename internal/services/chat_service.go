package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/souvik03-136/Go-Store/internal/aiclient"
	"github.com/souvik03-136/Go-Store/internal/repository"
)

var (
	ErrInvalidMessage = errors.New("message must be between 2 and 4000 characters")
)

type AIChatter interface {
	Chat(ctx context.Context, req *aiclient.ChatRequest) (*aiclient.ChatResponse, error)
}

type ChatRepo interface {
	CreateConversation(ctx context.Context, userID, title string) (*repository.Conversation, error)
	ListConversations(ctx context.Context, userID string) ([]*repository.Conversation, error)
	GetConversation(ctx context.Context, id, userID string) (*repository.Conversation, error)
	GetConversationWithMessages(ctx context.Context, id, userID string) (*repository.ConversationWithMessages, error)
	GetRecentMessages(ctx context.Context, conversationID, userID string, limit int) ([]aiclient.ChatMessage, error)
	AddTurn(ctx context.Context, conversationID, userID string, userContent, assistantContent string, citations []aiclient.Citation, newTitle string) (*repository.ConversationMessage, *repository.ConversationMessage, error)
	DeleteConversation(ctx context.Context, id, userID string) error
}

type ChatTurnResult struct {
	ConversationID   string                         `json:"conversation_id"`
	UserMessage      repository.ConversationMessage `json:"user_message"`
	AssistantMessage repository.ConversationMessage `json:"assistant_message"`
}

type ChatService struct {
	repo ChatRepo
	ai   AIChatter
}

func NewChatService(repo ChatRepo, ai AIChatter) *ChatService {
	return &ChatService{repo: repo, ai: ai}
}

func (s *ChatService) ListConversations(ctx context.Context, userID string) ([]*repository.Conversation, error) {
	return s.repo.ListConversations(ctx, userID)
}

func (s *ChatService) CreateConversation(ctx context.Context, userID, title string) (*repository.Conversation, error) {
	return s.repo.CreateConversation(ctx, userID, title)
}

func (s *ChatService) GetConversation(ctx context.Context, id, userID string) (*repository.ConversationWithMessages, error) {
	return s.repo.GetConversationWithMessages(ctx, id, userID)
}

func (s *ChatService) DeleteConversation(ctx context.Context, id, userID string) error {
	return s.repo.DeleteConversation(ctx, id, userID)
}

func (s *ChatService) SendMessage(ctx context.Context, userID, conversationID, message string, fileIDs []string) (*ChatTurnResult, error) {
	trimmed := strings.TrimSpace(message)
	runeCount := len([]rune(trimmed))
	if runeCount < 2 || runeCount > 4000 {
		return nil, ErrInvalidMessage
	}

	// Resolve or create conversation
	var conv *repository.Conversation
	var err error
	var isNewConv bool

	if conversationID == "" {
		conv, err = s.repo.CreateConversation(ctx, userID, "New Conversation")
		if err != nil {
			return nil, fmt.Errorf("creating conversation: %w", err)
		}
		conversationID = conv.ID
		isNewConv = true
	} else {
		conv, err = s.repo.GetConversation(ctx, conversationID, userID)
		if err != nil {
			return nil, err
		}
	}

	// Load previous history (validated server-side) without holding open a transaction
	var history []aiclient.ChatMessage
	if !isNewConv {
		history, err = s.repo.GetRecentMessages(ctx, conversationID, userID, 10)
		if err != nil {
			return nil, fmt.Errorf("reading conversation history: %w", err)
		}
	}

	if s.ai == nil {
		return nil, errors.New("AI service is not configured")
	}

	// Call local Ollama RAG pipeline without holding any database transaction
	aiReq := &aiclient.ChatRequest{
		UserID:  userID,
		Message: trimmed,
		History: history,
		FileIDs: fileIDs,
	}
	aiResp, err := s.ai.Chat(ctx, aiReq)
	if err != nil {
		return nil, fmt.Errorf("generating AI response: %w", err)
	}

	// Compute automated title from first user query if needed
	newTitle := ""
	if conv.Title == "New Conversation" || conv.Title == "" {
		newTitle = generateTitle(trimmed)
	}

	// Persist the user message and assistant answer with citations in the database
	userMsg, asstMsg, err := s.repo.AddTurn(
		ctx,
		conversationID,
		userID,
		trimmed,
		aiResp.Answer,
		aiResp.Citations,
		newTitle,
	)
	if err != nil {
		return nil, fmt.Errorf("saving chat turn: %w", err)
	}

	return &ChatTurnResult{
		ConversationID:   conversationID,
		UserMessage:      *userMsg,
		AssistantMessage: *asstMsg,
	}, nil
}

func generateTitle(text string) string {
	runes := []rune(text)
	if len(runes) > 50 {
		return string(runes[:50]) + "..."
	}
	return text
}

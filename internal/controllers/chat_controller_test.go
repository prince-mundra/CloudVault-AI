package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/souvik03-136/Go-Store/internal/auth"
	"github.com/souvik03-136/Go-Store/internal/repository"
	"github.com/souvik03-136/Go-Store/internal/services"
)

type mockChatService struct {
	turnResult *services.ChatTurnResult
	sendErr    error
	convs      []*repository.Conversation
	convWithMs *repository.ConversationWithMessages
	getErr     error
	deleteErr  error
}

func (m *mockChatService) SendMessage(ctx context.Context, userID, conversationID, message string, fileIDs []string) (*services.ChatTurnResult, error) {
	return m.turnResult, m.sendErr
}

func (m *mockChatService) ListConversations(ctx context.Context, userID string) ([]*repository.Conversation, error) {
	return m.convs, nil
}

func (m *mockChatService) GetConversation(ctx context.Context, id, userID string) (*repository.ConversationWithMessages, error) {
	return m.convWithMs, m.getErr
}

func (m *mockChatService) CreateConversation(ctx context.Context, userID, title string) (*repository.Conversation, error) {
	return &repository.Conversation{ID: "new-c", UserID: userID, Title: title, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}

func (m *mockChatService) DeleteConversation(ctx context.Context, id, userID string) error {
	return m.deleteErr
}

func setupChatTestRouter(svc ChatService, authUser string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := NewChatController(svc)
	r.Use(func(c *gin.Context) {
		if authUser != "" {
			c.Set(auth.SubjectKey, authUser)
		}
		c.Next()
	})
	r.POST("/v1/chat", ctrl.SendMessage)
	r.GET("/v1/conversations", ctrl.ListConversations)
	r.POST("/v1/conversations", ctrl.CreateConversation)
	r.GET("/v1/conversations/:id", ctrl.GetConversation)
	r.DELETE("/v1/conversations/:id", ctrl.DeleteConversation)
	return r
}

func TestChatControllerSendMessageRequiresAuth(t *testing.T) {
	r := setupChatTestRouter(&mockChatService{}, "")
	body := bytes.NewReader([]byte(`{"message":"hello"}`))
	req := httptest.NewRequest(http.MethodPost, "/v1/chat", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
	}
}

func TestChatControllerSendMessageSuccess(t *testing.T) {
	svc := &mockChatService{
		turnResult: &services.ChatTurnResult{
			ConversationID: "conv-1",
			UserMessage: repository.ConversationMessage{
				ID:             "u1",
				ConversationID: "conv-1",
				Role:           "user",
				Content:        "What is RAG?",
			},
			AssistantMessage: repository.ConversationMessage{
				ID:             "a1",
				ConversationID: "conv-1",
				Role:           "assistant",
				Content:        "RAG is retrieval augmented generation.",
			},
		},
	}
	r := setupChatTestRouter(svc, "user-123")
	payload := []byte(`{"message":"What is RAG?","file_ids":["f1"]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}
	var res services.ChatTurnResult
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.ConversationID != "conv-1" || res.AssistantMessage.Content != "RAG is retrieval augmented generation." {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestChatControllerGetConversationNotFound(t *testing.T) {
	svc := &mockChatService{getErr: repository.ErrConversationNotFound}
	r := setupChatTestRouter(svc, "user-123")
	req := httptest.NewRequest(http.MethodGet, "/v1/conversations/not-found", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", w.Code)
	}
}

func TestChatControllerDeleteConversation(t *testing.T) {
	svc := &mockChatService{}
	r := setupChatTestRouter(svc, "user-123")
	req := httptest.NewRequest(http.MethodDelete, "/v1/conversations/conv-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}

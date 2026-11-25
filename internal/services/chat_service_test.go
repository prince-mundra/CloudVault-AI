package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/souvik03-136/Go-Store/internal/aiclient"
	"github.com/souvik03-136/Go-Store/internal/repository"
)

type mockChatRepo struct {
	convs          map[string]*repository.Conversation
	messages       map[string][]repository.ConversationMessage
	createCalls    int
	addTurnCalls   int
	deleteCalls    int
	lastTitleAdded string
}

func newMockChatRepo() *mockChatRepo {
	return &mockChatRepo{
		convs:    make(map[string]*repository.Conversation),
		messages: make(map[string][]repository.ConversationMessage),
	}
}

func (m *mockChatRepo) CreateConversation(ctx context.Context, userID, title string) (*repository.Conversation, error) {
	m.createCalls++
	id := "conv-123"
	c := &repository.Conversation{
		ID:        id,
		UserID:    userID,
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.convs[id] = c
	return c, nil
}

func (m *mockChatRepo) ListConversations(ctx context.Context, userID string) ([]*repository.Conversation, error) {
	var list []*repository.Conversation
	for _, c := range m.convs {
		if c.UserID == userID {
			list = append(list, c)
		}
	}
	return list, nil
}

func (m *mockChatRepo) GetConversation(ctx context.Context, id, userID string) (*repository.Conversation, error) {
	c, ok := m.convs[id]
	if !ok || c.UserID != userID {
		return nil, repository.ErrConversationNotFound
	}
	return c, nil
}

func (m *mockChatRepo) GetConversationWithMessages(ctx context.Context, id, userID string) (*repository.ConversationWithMessages, error) {
	c, err := m.GetConversation(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	return &repository.ConversationWithMessages{
		Conversation: *c,
		Messages:     m.messages[id],
	}, nil
}

func (m *mockChatRepo) GetRecentMessages(ctx context.Context, conversationID, userID string, limit int) ([]aiclient.ChatMessage, error) {
	if _, err := m.GetConversation(ctx, conversationID, userID); err != nil {
		return nil, err
	}
	var res []aiclient.ChatMessage
	for _, msg := range m.messages[conversationID] {
		res = append(res, aiclient.ChatMessage{Role: msg.Role, Content: msg.Content})
	}
	return res, nil
}

func (m *mockChatRepo) AddTurn(ctx context.Context, conversationID, userID string, userContent, assistantContent string, citations []aiclient.Citation, newTitle string) (*repository.ConversationMessage, *repository.ConversationMessage, error) {
	m.addTurnCalls++
	m.lastTitleAdded = newTitle
	c, err := m.GetConversation(ctx, conversationID, userID)
	if err != nil {
		return nil, nil, err
	}
	if newTitle != "" {
		c.Title = newTitle
	}
	u := repository.ConversationMessage{
		ID:             "msg-u",
		ConversationID: conversationID,
		Role:           "user",
		Content:        userContent,
		CreatedAt:      time.Now(),
	}
	a := repository.ConversationMessage{
		ID:             "msg-a",
		ConversationID: conversationID,
		Role:           "assistant",
		Content:        assistantContent,
		Citations:      citations,
		CreatedAt:      time.Now(),
	}
	m.messages[conversationID] = append(m.messages[conversationID], u, a)
	return &u, &a, nil
}

func (m *mockChatRepo) DeleteConversation(ctx context.Context, id, userID string) error {
	m.deleteCalls++
	if _, err := m.GetConversation(ctx, id, userID); err != nil {
		return err
	}
	delete(m.convs, id)
	delete(m.messages, id)
	return nil
}

type mockAIChatter struct {
	lastReq *aiclient.ChatRequest
	resp    *aiclient.ChatResponse
	err     error
}

func (m *mockAIChatter) Chat(ctx context.Context, req *aiclient.ChatRequest) (*aiclient.ChatResponse, error) {
	m.lastReq = req
	return m.resp, m.err
}

func TestSendMessageValidation(t *testing.T) {
	repo := newMockChatRepo()
	ai := &mockAIChatter{}
	svc := NewChatService(repo, ai)

	if _, err := svc.SendMessage(context.Background(), "u1", "", "a", nil); !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("expected ErrInvalidMessage, got %v", err)
	}
	if _, err := svc.SendMessage(context.Background(), "u1", "", "   ", nil); !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("expected ErrInvalidMessage, got %v", err)
	}
}

func TestSendMessageAutoCreateConversationAndAddTurn(t *testing.T) {
	repo := newMockChatRepo()
	page := 1
	ai := &mockAIChatter{
		resp: &aiclient.ChatResponse{
			Answer: "Here is the answer [1].",
			Citations: []aiclient.Citation{
				{
					SourceNumber: 1,
					FileID:       "f1",
					FileName:     "doc.pdf",
					PageNumber:   &page,
					Score:        0.92,
				},
			},
		},
	}
	svc := NewChatService(repo, ai)

	res, err := svc.SendMessage(context.Background(), "u1", "", "What is CloudVault?", []string{"f1"})
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if repo.createCalls != 1 {
		t.Fatalf("expected 1 create call, got %d", repo.createCalls)
	}
	if repo.addTurnCalls != 1 {
		t.Fatalf("expected 1 addTurn call, got %d", repo.addTurnCalls)
	}
	if res.ConversationID != "conv-123" {
		t.Fatalf("unexpected conversation ID %s", res.ConversationID)
	}
	if res.AssistantMessage.Content != "Here is the answer [1]." {
		t.Fatalf("unexpected assistant content: %s", res.AssistantMessage.Content)
	}
	if len(res.AssistantMessage.Citations) != 1 {
		t.Fatalf("expected 1 citation, got %d", len(res.AssistantMessage.Citations))
	}
	if ai.lastReq.UserID != "u1" || ai.lastReq.Message != "What is CloudVault?" || len(ai.lastReq.FileIDs) != 1 {
		t.Fatalf("unexpected AI request: %+v", ai.lastReq)
	}
}

func TestSendMessageWithExistingConversationLoadsHistory(t *testing.T) {
	repo := newMockChatRepo()
	conv, _ := repo.CreateConversation(context.Background(), "u1", "Existing Analysis")
	repo.messages[conv.ID] = []repository.ConversationMessage{
		{Role: "user", Content: "Previous question"},
		{Role: "assistant", Content: "Previous answer"},
	}

	ai := &mockAIChatter{
		resp: &aiclient.ChatResponse{
			Answer:    "Follow up answer",
			Citations: []aiclient.Citation{},
		},
	}
	svc := NewChatService(repo, ai)

	res, err := svc.SendMessage(context.Background(), "u1", conv.ID, "Follow up question", nil)
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if res.ConversationID != conv.ID {
		t.Fatalf("unexpected conversation ID: %s", res.ConversationID)
	}
	if len(ai.lastReq.History) != 2 {
		t.Fatalf("expected 2 history items sent to AI, got %d", len(ai.lastReq.History))
	}
	if ai.lastReq.History[0].Content != "Previous question" {
		t.Fatalf("unexpected history item content: %s", ai.lastReq.History[0].Content)
	}
}

func TestSendMessageRejectsOtherUserConversation(t *testing.T) {
	repo := newMockChatRepo()
	conv, _ := repo.CreateConversation(context.Background(), "other-user", "Private")
	ai := &mockAIChatter{}
	svc := NewChatService(repo, ai)

	_, err := svc.SendMessage(context.Background(), "u1", conv.ID, "Sneaky question", nil)
	if !errors.Is(err, repository.ErrConversationNotFound) {
		t.Fatalf("expected ErrConversationNotFound, got %v", err)
	}
}

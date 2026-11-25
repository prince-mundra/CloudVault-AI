package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/souvik03-136/Go-Store/internal/aiclient"
)

var (
	ErrConversationNotFound = errors.New("conversation not found")
)

type Conversation struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ConversationMessage struct {
	ID             string              `json:"id"`
	ConversationID string              `json:"conversation_id"`
	Role           string              `json:"role"`
	Content        string              `json:"content"`
	Citations      []aiclient.Citation `json:"citations"`
	CreatedAt      time.Time           `json:"created_at"`
}

type ConversationWithMessages struct {
	Conversation
	Messages []ConversationMessage `json:"messages"`
}

type ChatRepository struct {
	db *sql.DB
}

func NewChatRepository(db *sql.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

func (r *ChatRepository) CreateConversation(ctx context.Context, userID, title string) (*Conversation, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "New Conversation"
	}
	if len([]rune(title)) > 255 {
		title = string([]rune(title)[:255])
	}
	conv := &Conversation{
		ID:        uuid.New().String(),
		UserID:    userID,
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO conversations (id, user_id, title, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`, conv.ID, conv.UserID, conv.Title, conv.CreatedAt, conv.UpdatedAt).Scan(&conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating conversation: %w", err)
	}
	return conv, nil
}

func (r *ChatRepository) ListConversations(ctx context.Context, userID string) ([]*Conversation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, title, created_at, updated_at
		FROM conversations
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("listing conversations: %w", err)
	}
	defer rows.Close()

	var convs []*Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning conversation: %w", err)
		}
		convs = append(convs, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating conversations: %w", err)
	}
	return convs, nil
}

func (r *ChatRepository) GetConversation(ctx context.Context, id, userID string) (*Conversation, error) {
	var c Conversation
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, title, created_at, updated_at
		FROM conversations
		WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&c.ID, &c.UserID, &c.Title, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrConversationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("reading conversation: %w", err)
	}
	return &c, nil
}

func (r *ChatRepository) GetConversationWithMessages(ctx context.Context, id, userID string) (*ConversationWithMessages, error) {
	conv, err := r.GetConversation(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, conversation_id, role, content, citations, created_at
		FROM conversation_messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
	`, id)
	if err != nil {
		return nil, fmt.Errorf("reading conversation messages: %w", err)
	}
	defer rows.Close()

	messages := make([]ConversationMessage, 0)
	for rows.Next() {
		var m ConversationMessage
		var citationsJSON []byte
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &citationsJSON, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning message: %w", err)
		}
		if len(citationsJSON) > 0 {
			if err := json.Unmarshal(citationsJSON, &m.Citations); err != nil {
				m.Citations = []aiclient.Citation{}
			}
		} else {
			m.Citations = []aiclient.Citation{}
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating messages: %w", err)
	}

	return &ConversationWithMessages{
		Conversation: *conv,
		Messages:     messages,
	}, nil
}

func (r *ChatRepository) GetRecentMessages(ctx context.Context, conversationID, userID string, limit int) ([]aiclient.ChatMessage, error) {
	if _, err := r.GetConversation(ctx, conversationID, userID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 10
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT role, content
		FROM conversation_messages
		WHERE conversation_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("reading recent messages: %w", err)
	}
	defer rows.Close()

	var descMessages []aiclient.ChatMessage
	for rows.Next() {
		var m aiclient.ChatMessage
		if err := rows.Scan(&m.Role, &m.Content); err != nil {
			return nil, fmt.Errorf("scanning recent message: %w", err)
		}
		descMessages = append(descMessages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating recent messages: %w", err)
	}

	// Reverse to chronological order
	n := len(descMessages)
	messages := make([]aiclient.ChatMessage, n)
	for i, m := range descMessages {
		messages[n-1-i] = m
	}
	return messages, nil
}

func (r *ChatRepository) AddTurn(
	ctx context.Context,
	conversationID, userID string,
	userContent, assistantContent string,
	citations []aiclient.Citation,
	newTitle string,
) (*ConversationMessage, *ConversationMessage, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("starting chat transaction: %w", err)
	}
	defer tx.Rollback()

	// Verify conversation exists and belongs to user
	var existingTitle string
	err = tx.QueryRowContext(ctx, `
		SELECT title FROM conversations WHERE id = $1 AND user_id = $2 FOR UPDATE
	`, conversationID, userID).Scan(&existingTitle)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, ErrConversationNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("locking conversation: %w", err)
	}

	now := time.Now()
	userMsg := &ConversationMessage{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		Role:           "user",
		Content:        userContent,
		Citations:      []aiclient.Citation{},
		CreatedAt:      now,
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO conversation_messages (id, conversation_id, role, content, citations, created_at)
		VALUES ($1, $2, $3, $4, '[]'::jsonb, $5)
	`, userMsg.ID, userMsg.ConversationID, userMsg.Role, userMsg.Content, userMsg.CreatedAt); err != nil {
		return nil, nil, fmt.Errorf("inserting user message: %w", err)
	}

	citationsJSON, err := json.Marshal(citations)
	if err != nil {
		citationsJSON = []byte("[]")
	}
	asstMsg := &ConversationMessage{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		Role:           "assistant",
		Content:        assistantContent,
		Citations:      citations,
		CreatedAt:      now.Add(time.Millisecond),
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO conversation_messages (id, conversation_id, role, content, citations, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, asstMsg.ID, asstMsg.ConversationID, asstMsg.Role, asstMsg.Content, citationsJSON, asstMsg.CreatedAt); err != nil {
		return nil, nil, fmt.Errorf("inserting assistant message: %w", err)
	}

	titleToSet := existingTitle
	if newTitle != "" && (existingTitle == "New Conversation" || existingTitle == "") {
		titleToSet = strings.TrimSpace(newTitle)
		if len([]rune(titleToSet)) > 255 {
			titleToSet = string([]rune(titleToSet)[:255])
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE conversations SET title = $1, updated_at = $2 WHERE id = $3
	`, titleToSet, asstMsg.CreatedAt, conversationID); err != nil {
		return nil, nil, fmt.Errorf("updating conversation timestamp: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("committing chat turn: %w", err)
	}

	return userMsg, asstMsg, nil
}

func (r *ChatRepository) DeleteConversation(ctx context.Context, id, userID string) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM conversations WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		return fmt.Errorf("deleting conversation: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking deleted rows: %w", err)
	}
	if rows == 0 {
		return ErrConversationNotFound
	}
	return nil
}

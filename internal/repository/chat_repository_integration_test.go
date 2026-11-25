package repository

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/souvik03-136/Go-Store/internal/aiclient"
)

func TestChatRepositoryLifecycleAndUserIsolation(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN is not set; PostgreSQL integration test skipped")
	}
	u, err := url.Parse(dsn)
	if err != nil || u.Scheme == "" {
		t.Skip("POSTGRES_TEST_DSN must be a postgres URL")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if err := admin.Ping(); err != nil {
		t.Skipf("PostgreSQL is unavailable: %v", err)
	}
	schema := "cv_chat_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.Exec(`CREATE SCHEMA ` + schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(`DROP SCHEMA ` + schema + ` CASCADE`)
	q := u.Query()
	q.Set("options", "-csearch_path="+schema)
	u.RawQuery = q.Encode()
	db, err := sql.Open("postgres", u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, statement := range []string{
		`CREATE TABLE users(id VARCHAR(36) PRIMARY KEY, username VARCHAR(255) UNIQUE, email VARCHAR(255) UNIQUE, password_hash VARCHAR(255))`,
		`CREATE TABLE conversations(
			id VARCHAR(36) PRIMARY KEY,
			user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title VARCHAR(255) NOT NULL DEFAULT 'New Conversation',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE conversation_messages(
			id VARCHAR(36) PRIMARY KEY,
			conversation_id VARCHAR(36) NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
			role VARCHAR(20) NOT NULL CHECK (role IN ('user', 'assistant')),
			content TEXT NOT NULL,
			citations JSONB NOT NULL DEFAULT '[]'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("setup failed on %q: %v", statement, err)
		}
	}

	userA := uuid.NewString()
	userB := uuid.NewString()
	if _, err := db.Exec(`INSERT INTO users(id, username, email, password_hash) VALUES ($1, 'userA', 'a@cv.io', 'h'), ($2, 'userB', 'b@cv.io', 'h')`, userA, userB); err != nil {
		t.Fatal(err)
	}

	repo := NewChatRepository(db)
	ctx := context.Background()

	// 1. Create conversation for user A
	convA, err := repo.CreateConversation(ctx, userA, "User A Project")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if convA.Title != "User A Project" {
		t.Fatalf("unexpected title: %s", convA.Title)
	}

	// 2. User B cannot read or access User A's conversation
	if _, err := repo.GetConversation(ctx, convA.ID, userB); !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("expected ErrConversationNotFound for user B, got %v", err)
	}
	if _, err := repo.GetConversationWithMessages(ctx, convA.ID, userB); !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("expected ErrConversationNotFound for user B messages, got %v", err)
	}
	if _, err := repo.GetRecentMessages(ctx, convA.ID, userB, 10); !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("expected ErrConversationNotFound for user B recent, got %v", err)
	}

	// 3. User A can add turns with structured citations
	page := 4
	cit := []aiclient.Citation{
		{
			SourceNumber: 1,
			FileID:       "f1",
			FileName:     "Quarterly_Report.pdf",
			ChunkIndex:   2,
			PageNumber:   &page,
			ChunkText:    "Revenue was up 20%",
			Score:        0.91,
		},
	}
	userMsg, asstMsg, err := repo.AddTurn(ctx, convA.ID, userA, "What was revenue?", "Revenue was up 20% [1].", cit, "Quarterly Revenue Analysis")
	if err != nil {
		t.Fatalf("AddTurn: %v", err)
	}
	if userMsg.Role != "user" || asstMsg.Role != "assistant" {
		t.Fatalf("unexpected message roles: %s, %s", userMsg.Role, asstMsg.Role)
	}
	if len(asstMsg.Citations) != 1 || asstMsg.Citations[0].FileName != "Quarterly_Report.pdf" {
		t.Fatalf("unexpected citations stored: %+v", asstMsg.Citations)
	}

	// 4. Retrieve conversation with messages
	fullConv, err := repo.GetConversationWithMessages(ctx, convA.ID, userA)
	if err != nil {
		t.Fatalf("GetConversationWithMessages: %v", err)
	}
	if len(fullConv.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(fullConv.Messages))
	}
	if fullConv.Messages[1].Citations[0].ChunkText != "Revenue was up 20%" {
		t.Fatalf("unexpected citation text: %s", fullConv.Messages[1].Citations[0].ChunkText)
	}

	// 5. Delete conversation cascades
	if err := repo.DeleteConversation(ctx, convA.ID, userA); err != nil {
		t.Fatalf("DeleteConversation: %v", err)
	}
	var msgCount int
	if err := db.QueryRow(`SELECT count(*) FROM conversation_messages WHERE conversation_id = $1`, convA.ID).Scan(&msgCount); err != nil {
		t.Fatal(err)
	}
	if msgCount != 0 {
		t.Fatalf("expected 0 messages remaining, got %d", msgCount)
	}
}

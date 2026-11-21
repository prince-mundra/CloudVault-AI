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
)

func TestSummaryStatusOwnerAndReadableShareScope(t *testing.T) {
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
	schema := "cv_summary_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
		`CREATE TABLE files(id VARCHAR(36) PRIMARY KEY,owner_id VARCHAR(36),state VARCHAR(20))`,
		`CREATE TABLE permissions(file_id VARCHAR(36),user_id VARCHAR(36),can_read BOOLEAN)`,
		`CREATE TABLE document_summary_jobs(file_id VARCHAR(36),state VARCHAR(20),attempts INTEGER,last_error TEXT,updated_at TIMESTAMPTZ)`,
		`CREATE TABLE document_summaries(file_id VARCHAR(36),summary_text TEXT,summary_model VARCHAR(120))`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("creating isolated schema table: %v", err)
		}
	}
	fileID := uuid.NewString()
	if _, err := db.Exec(`INSERT INTO files(id,owner_id,state) VALUES($1,'owner','active')`, fileID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO permissions(file_id,user_id,can_read) VALUES($1,'reader',TRUE),($1,'writer-without-read',FALSE)`, fileID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO document_summary_jobs(file_id,state,attempts,last_error,updated_at) VALUES($1,'succeeded',1,'',NOW())`, fileID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO document_summaries(file_id,summary_text,summary_model) VALUES($1,'approved summary','local-model')`, fileID); err != nil {
		t.Fatal(err)
	}
	repo := NewSummaryRepository(db)
	for _, principal := range []string{"owner", "reader"} {
		status, err := repo.GetStatus(context.Background(), fileID, principal)
		if err != nil || status.Summary != "approved summary" {
			t.Fatalf("principal %q status=%+v err=%v", principal, status, err)
		}
	}
	for _, principal := range []string{"outsider", "writer-without-read"} {
		if _, err := repo.GetStatus(context.Background(), fileID, principal); !errors.Is(err, ErrSummaryStatusNotFound) {
			t.Fatalf("principal %q error=%v, want not-found", principal, err)
		}
	}
	if _, err := db.Exec(`UPDATE files SET state='deleting' WHERE id=$1`, fileID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetStatus(context.Background(), fileID, "owner"); !errors.Is(err, ErrSummaryStatusNotFound) {
		t.Fatalf("inactive file error=%v, want not-found", err)
	}
}

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

	// Test revocation of read access
	activeFile := uuid.NewString()
	if _, err := db.Exec(`INSERT INTO files(id,owner_id,state) VALUES($1,'owner','active')`, activeFile); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO permissions(file_id,user_id,can_read) VALUES($1,'temp-reader',TRUE)`, activeFile); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO document_summary_jobs(file_id,state,attempts,last_error,updated_at) VALUES($1,'succeeded',1,'',NOW())`, activeFile); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO document_summaries(file_id,summary_text,summary_model) VALUES($1,'live summary','model')`, activeFile); err != nil {
		t.Fatal(err)
	}
	// Verify temp-reader can access
	st, err := repo.GetStatus(context.Background(), activeFile, "temp-reader")
	if err != nil || st.Summary != "live summary" {
		t.Fatalf("temp-reader failed to access before revocation: %v", err)
	}
	// Revoke access (delete permission row)
	if _, err := db.Exec(`DELETE FROM permissions WHERE file_id=$1 AND user_id='temp-reader'`, activeFile); err != nil {
		t.Fatal(err)
	}
	// Verify temp-reader immediately loses access
	if _, err := repo.GetStatus(context.Background(), activeFile, "temp-reader"); !errors.Is(err, ErrSummaryStatusNotFound) {
		t.Fatalf("revoked user did not receive not-found: %v", err)
	}

	// Test Ingestion Repository
	ingestionRepo := NewIngestionRepository(db)
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS document_ingestion_jobs(id BIGSERIAL PRIMARY KEY, file_id VARCHAR(36) UNIQUE, state VARCHAR(20), attempts INTEGER, last_error TEXT, updated_at TIMESTAMPTZ)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO document_ingestion_jobs(file_id,state,attempts,last_error,updated_at) VALUES($1,'processing',2,'',NOW())`, activeFile); err != nil {
		t.Fatal(err)
	}
	// Owner can read ingestion status
	ingStatus, err := ingestionRepo.GetStatus(context.Background(), activeFile, "owner")
	if err != nil || ingStatus.State != "processing" || ingStatus.Attempts != 2 {
		t.Fatalf("owner failed to get ingestion status: %v", err)
	}
	// Non-owner cannot read ingestion status
	if _, err := ingestionRepo.GetStatus(context.Background(), activeFile, "other-user"); !errors.Is(err, ErrIngestionStatusNotFound) {
		t.Fatalf("non-owner accessed ingestion status: %v", err)
	}
}

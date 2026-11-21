package repository

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/souvik03-136/Go-Store/internal/models"
)

func TestUploadReservationConcurrentQuota(t *testing.T) {
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
	schema := "cv_phase1_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	db.SetMaxOpenConns(4)
	statements := []string{
		`CREATE TABLE users(id VARCHAR(36) PRIMARY KEY)`,
		`CREATE TABLE files(id VARCHAR(36),name TEXT,path TEXT,url TEXT,size BIGINT,content_type TEXT,owner_id VARCHAR(36),folder_id VARCHAR(36),created_at TIMESTAMPTZ,updated_at TIMESTAMPTZ,sha256 CHAR(64),state VARCHAR(20))`,
		`CREATE TABLE upload_reservations(id VARCHAR(36) PRIMARY KEY,owner_id VARCHAR(36),sha256 CHAR(64),size_bytes BIGINT,object_key TEXT,expires_at TIMESTAMPTZ,UNIQUE(owner_id,sha256))`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("creating isolated schema table: %v", err)
		}
	}
	owner := uuid.NewString()
	if _, err := db.Exec(`INSERT INTO users(id) VALUES($1)`, owner); err != nil {
		t.Fatal(err)
	}
	repo := NewUploadRepository(db)
	type result struct{ err error }
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for _, hash := range []string{strings.Repeat("a", 64), strings.Repeat("b", 64)} {
		wg.Add(1)
		go func(hash string) {
			defer wg.Done()
			_, _, err := repo.Reserve(context.Background(), owner, hash, owner+"/"+hash, 70, 100)
			results <- result{err}
		}(hash)
	}
	wg.Wait()
	close(results)
	success, quota := 0, 0
	for item := range results {
		if item.err == nil {
			success++
		} else if errors.Is(item.err, ErrQuotaExceeded) {
			quota++
		} else {
			t.Fatalf("unexpected reservation error: %v", item.err)
		}
	}
	if success != 1 || quota != 1 {
		t.Fatalf("success=%d quota_rejections=%d; want one each", success, quota)
	}
}

func TestFinalizeQueuesIngestionAtomically(t *testing.T) {
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
	schema := "cv_finalize_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	statements := []string{
		`CREATE TABLE users(id VARCHAR(36) PRIMARY KEY)`,
		`CREATE TABLE files(id VARCHAR(36) PRIMARY KEY,name TEXT,path TEXT,url TEXT,size BIGINT,content_type TEXT,owner_id VARCHAR(36),folder_id VARCHAR(36),created_at TIMESTAMPTZ,updated_at TIMESTAMPTZ,sha256 CHAR(64),state VARCHAR(20))`,
		`CREATE TABLE upload_reservations(id VARCHAR(36) PRIMARY KEY,owner_id VARCHAR(36),sha256 CHAR(64),size_bytes BIGINT,object_key TEXT,expires_at TIMESTAMPTZ,UNIQUE(owner_id,sha256))`,
		`CREATE TABLE file_activity(owner_id VARCHAR(36),file_id VARCHAR(36),file_name TEXT,action TEXT,size_bytes BIGINT,actor_id VARCHAR(36))`,
		`CREATE TABLE audit_events(actor_id VARCHAR(36),actor_subject TEXT,owner_id VARCHAR(36),resource_type TEXT,resource_id VARCHAR(36),action TEXT,outcome TEXT,details JSONB)`,
		`CREATE TABLE document_ingestion_jobs(id BIGSERIAL PRIMARY KEY,file_id VARCHAR(36) UNIQUE,state TEXT CHECK(state='blocked'))`,
		`CREATE TABLE document_summary_jobs(id BIGSERIAL PRIMARY KEY,file_id VARCHAR(36) UNIQUE,state TEXT CHECK(state='blocked'))`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("creating isolated schema table: %v", err)
		}
	}
	owner, fileID, hash := uuid.NewString(), uuid.NewString(), strings.Repeat("c", 64)
	if _, err := db.Exec(`INSERT INTO users(id) VALUES($1)`, owner); err != nil {
		t.Fatal(err)
	}
	repo := NewUploadRepository(db)
	reservation, duplicate, err := repo.Reserve(context.Background(), owner, hash, owner+"/"+hash, 16, 100)
	if err != nil || duplicate != nil {
		t.Fatalf("reserve failed: duplicate=%v err=%v", duplicate, err)
	}
	now := time.Now().UTC()
	file := &models.File{ID: fileID, Name: "notes.txt", Path: reservation.ObjectKey, Size: 16, ContentType: "text/plain", OwnerID: owner, CreatedAt: now, UpdatedAt: now, SHA256: hash, State: "active"}
	if err := repo.Finalize(context.Background(), reservation.ID, file, owner); err == nil {
		t.Fatal("expected queue constraint failure")
	}
	var files, audits, activities, reservations, summaryJobs int
	_ = db.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&files)
	_ = db.QueryRow(`SELECT COUNT(*) FROM audit_events`).Scan(&audits)
	_ = db.QueryRow(`SELECT COUNT(*) FROM file_activity`).Scan(&activities)
	_ = db.QueryRow(`SELECT COUNT(*) FROM upload_reservations`).Scan(&reservations)
	_ = db.QueryRow(`SELECT COUNT(*) FROM document_summary_jobs`).Scan(&summaryJobs)
	if files != 0 || audits != 0 || activities != 0 || reservations != 1 || summaryJobs != 0 {
		t.Fatalf("failed finalize was not atomic: files=%d audits=%d activities=%d reservations=%d summaries=%d", files, audits, activities, reservations, summaryJobs)
	}
	if _, err := db.Exec(`ALTER TABLE document_ingestion_jobs DROP CONSTRAINT document_ingestion_jobs_state_check`); err != nil {
		t.Fatal(err)
	}
	if err := repo.Finalize(context.Background(), reservation.ID, file, owner); err == nil {
		t.Fatal("expected summary queue constraint failure")
	}
	_ = db.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&files)
	_ = db.QueryRow(`SELECT COUNT(*) FROM document_ingestion_jobs`).Scan(&summaryJobs)
	if files != 0 || summaryJobs != 0 {
		t.Fatalf("summary enqueue failure left partial upload: files=%d ingestion_jobs=%d", files, summaryJobs)
	}
	if _, err := db.Exec(`ALTER TABLE document_summary_jobs DROP CONSTRAINT document_summary_jobs_state_check`); err != nil {
		t.Fatal(err)
	}
	if err := repo.Finalize(context.Background(), reservation.ID, file, owner); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := db.QueryRow(`SELECT state FROM document_ingestion_jobs WHERE file_id=$1`, fileID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "queued" {
		t.Fatalf("ingestion state = %q, want queued", state)
	}
	if err := db.QueryRow(`SELECT state FROM document_summary_jobs WHERE file_id=$1`, fileID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "queued" {
		t.Fatalf("summary state = %q, want queued", state)
	}
	_ = db.QueryRow(`SELECT COUNT(*) FROM upload_reservations`).Scan(&reservations)
	if reservations != 0 {
		t.Fatalf("finalized upload left %d quota reservations", reservations)
	}
}

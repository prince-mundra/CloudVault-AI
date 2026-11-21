package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/souvik03-136/Go-Store/internal/models"
	"github.com/souvik03-136/Go-Store/internal/repository"
	"github.com/souvik03-136/Go-Store/internal/storage"
)

func multipartHeader(t *testing.T, name string, payload []byte, declared string) *multipart.FileHeader {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	part, err := w.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(payload)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if err := req.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	h := req.MultipartForm.File["file"][0]
	if declared != "" {
		h.Header.Set("Content-Type", declared)
	}
	return h
}

func TestInspectUploadHashesActualBytesAndSniffsMIME(t *testing.T) {
	payload := []byte("actual plain text\n")
	h := multipartHeader(t, "x.txt", payload, "image/png")
	hash, ct, err := inspectUpload(context.Background(), h, 1024)
	if err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256(payload)
	if hash != hex.EncodeToString(want[:]) {
		t.Fatalf("SHA256 = %q, want %x", hash, want)
	}
	if ct != "text/plain; charset=utf-8" {
		t.Fatalf("detected MIME = %q", ct)
	}
}
func TestInspectUploadEnforcesActualFileLimit(t *testing.T) {
	h := multipartHeader(t, "x.bin", bytes.Repeat([]byte("x"), 16), "application/octet-stream")
	if _, _, err := inspectUpload(context.Background(), h, 8); err == nil {
		t.Fatal("expected file size limit error")
	}
	h.Size++
	if _, _, err := inspectUpload(context.Background(), h, 32); err == nil {
		t.Fatal("expected multipart size mismatch error")
	}
}

type fakeFiles struct{ file *models.File }

func (f *fakeFiles) GetFileByID(id string) (*models.File, error) {
	if f.file == nil || f.file.ID != id {
		return nil, errors.New("file not found")
	}
	copy := *f.file
	return &copy, nil
}
func (f *fakeFiles) ListFilesByOwner(string) ([]*models.File, error) { return nil, nil }
func (f *fakeFiles) SearchOwned(string, string, string, string, string, *string, int, int) ([]*models.File, error) {
	return nil, nil
}
func (f *fakeFiles) UpdateFile(*models.File) error { return nil }

type fakeFolders struct{}

func (fakeFolders) GetOwned(string, string) (*models.Folder, error) { return nil, nil }

type fakePermissions struct {
	p       *models.Permission
	deleted bool
}

func (f *fakePermissions) GetPermission(string, string) (*models.Permission, error) { return f.p, nil }
func (f *fakePermissions) DeletePermission(string, string) error                    { f.deleted = true; return nil }

type fakeActivity struct {
	actor  string
	action string
}

func (f *fakeActivity) Record(_ string, actor string, _ *string, _ string, action string, _ int64) error {
	f.actor = actor
	f.action = action
	return nil
}

type fakeUploads struct {
	reservation             *repository.UploadReservation
	duplicate               *models.File
	reserveErr, finalizeErr error
	expired                 []repository.UploadReservation
	released                int
	finalized               int
}

func (f *fakeUploads) Reserve(_ context.Context, owner, hash, key string, size, quota int64) (*repository.UploadReservation, *models.File, error) {
	if f.reserveErr != nil {
		return nil, nil, f.reserveErr
	}
	if f.duplicate != nil {
		return nil, f.duplicate, nil
	}
	if f.reservation == nil {
		f.reservation = &repository.UploadReservation{ID: "reservation", OwnerID: owner, SHA256: hash, ObjectKey: key, SizeBytes: size, ExpiresAt: time.Now().Add(time.Hour)}
	}
	return f.reservation, nil, nil
}
func (f *fakeUploads) Expired(context.Context, string) ([]repository.UploadReservation, error) {
	return nil, nil
}
func (f *fakeUploads) ReleaseExpired(context.Context, string) error { return nil }
func (f *fakeUploads) Release(context.Context, string) error        { f.released++; return nil }
func (f *fakeUploads) Finalize(context.Context, string, *models.File, string) error {
	f.finalized++
	return f.finalizeErr
}
func (f *fakeUploads) ExpiredAll(context.Context) ([]repository.UploadReservation, error) {
	return f.expired, nil
}

type fakeAudit struct {
	events []repository.AuditEvent
	err    error
}

func (f *fakeAudit) Record(_ context.Context, e repository.AuditEvent) error {
	f.events = append(f.events, e)
	return f.err
}

type fakeDeleteQueue struct{ requests int }

func (f *fakeDeleteQueue) Request(context.Context, *models.File, string) error {
	f.requests++
	return nil
}
func (*fakeDeleteQueue) Claim(context.Context) (*repository.FileDeleteJob, error)     { return nil, nil }
func (*fakeDeleteQueue) Complete(context.Context, *repository.FileDeleteJob) error    { return nil }
func (*fakeDeleteQueue) Fail(context.Context, *repository.FileDeleteJob, error) error { return nil }

type fakeStorage struct {
	uploadErr, deleteErr, downloadErr error
	uploaded, deleted                 int
	payload                           []byte
}

func (f *fakeStorage) UploadFile(context.Context, *multipart.FileHeader, string) (string, error) {
	f.uploaded++
	return "https://provider.invalid/object", f.uploadErr
}
func (f *fakeStorage) DownloadFile(context.Context, string) (io.ReadCloser, error) {
	if f.downloadErr != nil {
		return nil, f.downloadErr
	}
	return io.NopCloser(strings.NewReader(string(f.payload))), nil
}
func (f *fakeStorage) DeleteFile(context.Context, string) error { f.deleted++; return f.deleteErr }

var _ storage.Storage = (*fakeStorage)(nil)

func testFileService(file *models.File, uploads *fakeUploads, audit *fakeAudit, store *fakeStorage, deletes *fakeDeleteQueue, activity *fakeActivity, permissions *fakePermissions) *FileService {
	return NewFileService(&fakeFiles{file: file}, fakeFolders{}, permissions, activity, uploads, audit, deletes, store, 1<<20, 1<<20)
}

func TestUploadStorageFailureReleasesReservation(t *testing.T) {
	uploads := &fakeUploads{}
	store := &fakeStorage{uploadErr: errors.New("provider down")}
	svc := testFileService(nil, uploads, &fakeAudit{}, store, &fakeDeleteQueue{}, nil, &fakePermissions{})
	h := multipartHeader(t, "x.txt", []byte("hello"), "")
	if _, _, err := svc.UploadFile(context.Background(), h, "user-1"); err == nil {
		t.Fatal("expected upload failure")
	}
	if store.deleted != 1 || uploads.released != 1 || uploads.finalized != 0 {
		t.Fatalf("cleanup state: deletes=%d released=%d finalized=%d", store.deleted, uploads.released, uploads.finalized)
	}
}
func TestUploadFinalizeFailureDeletesObjectAndReleasesReservation(t *testing.T) {
	uploads := &fakeUploads{finalizeErr: errors.New("database down")}
	store := &fakeStorage{}
	svc := testFileService(nil, uploads, &fakeAudit{}, store, &fakeDeleteQueue{}, nil, &fakePermissions{})
	h := multipartHeader(t, "x.txt", []byte("hello"), "")
	if _, _, err := svc.UploadFile(context.Background(), h, "user-1"); err == nil {
		t.Fatal("expected finalization failure")
	}
	if store.uploaded != 1 || store.deleted != 1 || uploads.released != 1 {
		t.Fatalf("rollback state: uploaded=%d deleted=%d released=%d", store.uploaded, store.deleted, uploads.released)
	}
}

func TestUploadDuplicateDoesNotWriteStorage(t *testing.T) {
	existing := &models.File{ID: "existing", OwnerID: "user-1", Name: "original.txt", Path: "private"}
	uploads := &fakeUploads{duplicate: existing}
	store := &fakeStorage{}
	svc := testFileService(existing, uploads, &fakeAudit{}, store, &fakeDeleteQueue{}, nil, &fakePermissions{})
	h := multipartHeader(t, "copy.txt", []byte("identical"), "")
	file, duplicate, err := svc.UploadFile(context.Background(), h, "user-1")
	if err != nil || !duplicate || file.ID != existing.ID {
		t.Fatalf("file=%+v duplicate=%v err=%v", file, duplicate, err)
	}
	if store.uploaded != 0 {
		t.Fatalf("duplicate triggered %d storage writes", store.uploaded)
	}
}

func TestSharedCanDeleteNeverDeletesOriginal(t *testing.T) {
	file := &models.File{ID: "file-1", OwnerID: "owner-1", Path: "owner-1/hash"}
	perms := &fakePermissions{p: &models.Permission{CanRead: true, CanDelete: true}}
	deletes := &fakeDeleteQueue{}
	store := &fakeStorage{}
	svc := testFileService(file, &fakeUploads{}, &fakeAudit{}, store, deletes, nil, perms)
	if err := svc.DeleteFile(context.Background(), file.ID, "recipient-1"); err == nil {
		t.Fatal("recipient unexpectedly deleted owner's file")
	}
	if deletes.requests != 0 || store.deleted != 0 {
		t.Fatalf("shared delete caused irreversible work: queued=%d object-deletes=%d", deletes.requests, store.deleted)
	}
}
func TestDownloadAuditsActualActor(t *testing.T) {
	file := &models.File{ID: "file-1", OwnerID: "owner-1", Name: "readme.txt", Path: "owner-1/hash", Size: 4, ContentType: "text/plain"}
	audit := &fakeAudit{}
	activity := &fakeActivity{}
	store := &fakeStorage{payload: []byte("data")}
	svc := testFileService(file, &fakeUploads{}, audit, store, &fakeDeleteQueue{}, activity, &fakePermissions{})
	_, reader, err := svc.DownloadOwnedFile(context.Background(), file.ID, "owner-1")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, reader)
	_ = reader.Close()
	if len(audit.events) != 1 || audit.events[0].ActorSubject != "owner-1" || audit.events[0].Action != "download" {
		t.Fatalf("incorrect audit event: %+v", audit.events)
	}
	if activity.actor != "owner-1" || activity.action != "download" {
		t.Fatalf("incorrect activity actor/action: %q %q", activity.actor, activity.action)
	}
}

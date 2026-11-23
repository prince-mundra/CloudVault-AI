package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/souvik03-136/Go-Store/internal/models"
	"github.com/souvik03-136/Go-Store/internal/repository"
)

type memoryFileMetadataRepo struct {
	files map[string]*models.File
}

func (m *memoryFileMetadataRepo) GetFileByID(id string) (*models.File, error) {
	f, ok := m.files[id]
	if !ok {
		return nil, repository.ErrReservationNotFound
	}
	cp := *f
	return &cp, nil
}
func (m *memoryFileMetadataRepo) ListFilesByOwner(ownerID string) ([]*models.File, error) {
	var list []*models.File
	for _, f := range m.files {
		if f.OwnerID == ownerID {
			cp := *f
			list = append(list, &cp)
		}
	}
	return list, nil
}
func (m *memoryFileMetadataRepo) SearchOwned(ownerID, _, _, _, _ string, _ *string, _, _ int) ([]*models.File, error) {
	return m.ListFilesByOwner(ownerID)
}
func (m *memoryFileMetadataRepo) UpdateFile(f *models.File) error {
	cp := *f
	m.files[f.ID] = &cp
	return nil
}

type memoryUploadStore struct {
	reservations map[string]*repository.UploadReservation
	finalized    []*models.File
}

func newMemoryUploadStore() *memoryUploadStore {
	return &memoryUploadStore{
		reservations: make(map[string]*repository.UploadReservation),
	}
}

func (m *memoryUploadStore) Reserve(_ context.Context, owner, hash, key string, size, quota int64) (*repository.UploadReservation, *models.File, error) {
	res := &repository.UploadReservation{
		ID:        uuid.NewString(),
		OwnerID:   owner,
		SHA256:    hash,
		SizeBytes: size,
		ObjectKey: key,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	m.reservations[res.ID] = res
	return res, nil, nil
}
func (m *memoryUploadStore) Expired(_ context.Context, _ string) ([]repository.UploadReservation, error) {
	return nil, nil
}
func (m *memoryUploadStore) ReleaseExpired(_ context.Context, _ string) error { return nil }
func (m *memoryUploadStore) Release(_ context.Context, id string) error {
	delete(m.reservations, id)
	return nil
}
func (m *memoryUploadStore) Finalize(_ context.Context, resID string, f *models.File, _ string) error {
	res, ok := m.reservations[resID]
	if !ok {
		return repository.ErrReservationNotFound
	}
	if res.OwnerID != f.OwnerID {
		return repository.ErrReservationNotFound
	}
	delete(m.reservations, resID)
	m.finalized = append(m.finalized, f)
	return nil
}
func (m *memoryUploadStore) ExpiredAll(_ context.Context) ([]repository.UploadReservation, error) {
	return nil, nil
}

func TestAnonymousUserUploadAndOwnership(t *testing.T) {
	anonID := uuid.NewString()
	store := &fakeStorage{}
	fileRepo := &memoryFileMetadataRepo{files: make(map[string]*models.File)}
	uploadStore := newMemoryUploadStore()
	activityRepo := &fakeActivity{}
	auditRepo := &fakeAudit{}

	svc := NewFileService(
		fileRepo,
		fakeFolders{},
		&fakePermissions{},
		activityRepo,
		uploadStore,
		auditRepo,
		nil,
		store,
		10*1024*1024,
		5*1024*1024,
	)

	payload := []byte("%PDF-1.4\n%synthetic test pdf content\n%%EOF")
	header := multipartHeader(t, "document.pdf", payload, "application/pdf")

	file, duplicate, err := svc.UploadFile(context.Background(), header, anonID)
	if err != nil {
		t.Fatalf("UploadFile failed for anonymous user: %v", err)
	}
	if duplicate {
		t.Fatal("expected non-duplicate upload")
	}

	if file.OwnerID != anonID {
		t.Fatalf("file.OwnerID = %q, want anonymous ID %q", file.OwnerID, anonID)
	}
	if len(uploadStore.finalized) != 1 {
		t.Fatalf("finalized files = %d, want 1", len(uploadStore.finalized))
	}
	if uploadStore.finalized[0].OwnerID != anonID {
		t.Fatalf("finalized owner = %q, want %q", uploadStore.finalized[0].OwnerID, anonID)
	}

	// Save to fileRepo to test access checks
	fileRepo.files[file.ID] = file

	// Anonymous owner can access file
	accessible, err := svc.GetOwnedFileByID(file.ID, anonID)
	if err != nil {
		t.Fatalf("GetOwnedFileByID failed for owner: %v", err)
	}
	if accessible.ID != file.ID {
		t.Fatalf("accessible.ID = %q, want %q", accessible.ID, file.ID)
	}

	// Other user cannot access anonymous file
	otherUser := uuid.NewString()
	_, err = svc.GetOwnedFileByID(file.ID, otherUser)
	if err == nil {
		t.Fatal("expected GetOwnedFileByID to fail for non-owner")
	}
	if err.Error() != "permission denied" {
		t.Fatalf("expected 'permission denied', got: %v", err)
	}
}

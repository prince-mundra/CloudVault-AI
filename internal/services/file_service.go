// internal/services/file_service.go
package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"strings"

	"github.com/souvik03-136/Go-Store/internal/models"
	"github.com/souvik03-136/Go-Store/internal/repository"
	"github.com/souvik03-136/Go-Store/internal/storage"
)

type FileService struct {
	fileRepo                   FileMetadataRepository
	folderRepo                 FolderOwnerLookup
	permissionRepo             FilePermissionRepository
	activityRepo               FileActivityRecorder
	uploadRepo                 UploadReservationStore
	auditRepo                  AuditRecorder
	deleteRepo                 FileDeleteQueue
	storage                    storage.Storage
	quotaBytes, maxUploadBytes int64
}

type FileMetadataRepository interface {
	GetFileByID(string) (*models.File, error)
	ListFilesByOwner(string) ([]*models.File, error)
	SearchOwned(string, string, string, string, string, *string, int, int) ([]*models.File, error)
	UpdateFile(*models.File) error
}
type FolderOwnerLookup interface {
	GetOwned(string, string) (*models.Folder, error)
}
type FilePermissionRepository interface {
	GetPermission(string, string) (*models.Permission, error)
	DeletePermission(string, string) error
}
type FileActivityRecorder interface {
	Record(string, string, *string, string, string, int64) error
}
type UploadReservationStore interface {
	Reserve(context.Context, string, string, string, int64, int64) (*repository.UploadReservation, *models.File, error)
	Expired(context.Context, string) ([]repository.UploadReservation, error)
	ReleaseExpired(context.Context, string) error
	Release(context.Context, string) error
	Finalize(context.Context, string, *models.File, string) error
	ExpiredAll(context.Context) ([]repository.UploadReservation, error)
}
type AuditRecorder interface {
	Record(context.Context, repository.AuditEvent) error
}
type FileDeleteQueue interface {
	Request(context.Context, *models.File, string) error
	Claim(context.Context) (*repository.FileDeleteJob, error)
	Complete(context.Context, *repository.FileDeleteJob) error
	Fail(context.Context, *repository.FileDeleteJob, error) error
}

func NewFileService(fileRepo FileMetadataRepository, folderRepo FolderOwnerLookup, permissionRepo FilePermissionRepository, activityRepo FileActivityRecorder, uploadRepo UploadReservationStore, auditRepo AuditRecorder, deleteRepo FileDeleteQueue, store storage.Storage, quotaBytes, maxUploadBytes int64) *FileService {
	return &FileService{fileRepo: fileRepo, folderRepo: folderRepo, permissionRepo: permissionRepo, activityRepo: activityRepo, uploadRepo: uploadRepo, auditRepo: auditRepo, deleteRepo: deleteRepo, storage: store, quotaBytes: quotaBytes, maxUploadBytes: maxUploadBytes}
}

func (s *FileService) MoveToFolder(ownerID, fileID string, folderID *string) (*models.File, error) {
	file, err := s.fileRepo.GetFileByID(fileID)
	if err != nil {
		return nil, err
	}
	if !file.IsOwner(ownerID) {
		return nil, errors.New("permission denied")
	}
	if folderID != nil {
		if _, err := s.folderRepo.GetOwned(*folderID, ownerID); err != nil {
			return nil, errors.New("folder not found")
		}
	}
	file.FolderID = folderID
	if err := s.updateWithAudit(file, ownerID, "move", map[string]any{"folder_id": folderID}); err != nil {
		return nil, err
	}
	return file, nil
}

func (s *FileService) UploadFile(ctx context.Context, header *multipart.FileHeader, ownerID string) (*models.File, bool, error) {
	if header == nil {
		return nil, false, errors.New("file is required")
	}
	if ownerID == "" {
		return nil, false, errors.New("owner id is required")
	}
	hash, contentType, err := inspectUpload(ctx, header, s.maxUploadBytes)
	if err != nil {
		return nil, false, err
	}
	if header.Header == nil {
		header.Header = make(textproto.MIMEHeader)
	}
	header.Header.Set("Content-Type", contentType)
	// The per-upload suffix prevents cleanup of an older deleting object from
	// racing with a re-upload of the same content. Active duplicates are returned
	// before storage is touched, so this does not duplicate settled content.
	objectKey := fmt.Sprintf("%s/%s/%s", ownerID, hash, generateID())
	if err := s.reapExpiredReservations(ctx, ownerID); err != nil {
		return nil, false, fmt.Errorf("recovering expired upload reservations: %w", err)
	}
	reservation, duplicate, err := s.uploadRepo.Reserve(ctx, ownerID, hash, objectKey, header.Size, s.quotaBytes)
	if err != nil {
		return nil, false, err
	}
	if duplicate != nil {
		return duplicate, true, nil
	}
	if _, err := s.storage.UploadFile(ctx, header, objectKey); err != nil {
		if cleanupErr := s.storage.DeleteFile(ctx, objectKey); cleanupErr == nil {
			_ = s.uploadRepo.Release(ctx, reservation.ID)
		}
		return nil, false, fmt.Errorf("uploading file to storage: %w", err)
	}
	file := models.NewFile(generateID(), header.Filename, objectKey, "", contentType, ownerID, header.Size)
	file.SHA256 = hash
	if err := s.uploadRepo.Finalize(ctx, reservation.ID, file, ownerID); err != nil {
		if cleanupErr := s.storage.DeleteFile(ctx, objectKey); cleanupErr == nil {
			_ = s.uploadRepo.Release(ctx, reservation.ID)
		}
		return nil, false, fmt.Errorf("finalizing uploaded file: %w", err)
	}
	return file, false, nil
}
func (s *FileService) reapExpiredReservations(ctx context.Context, ownerID string) error {
	items, err := s.uploadRepo.Expired(ctx, ownerID)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := s.storage.DeleteFile(ctx, item.ObjectKey); err != nil {
			return fmt.Errorf("cleaning abandoned object: %w", err)
		}
		if err := s.uploadRepo.ReleaseExpired(ctx, item.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *FileService) GetFileByID(id string) (*models.File, error) {
	if id == "" {
		return nil, errors.New("file id is required")
	}
	return s.fileRepo.GetFileByID(id)
}
func (s *FileService) GetOwnedFileByID(id, requesterID string) (*models.File, error) {
	if requesterID == "" {
		return nil, errors.New("permission denied")
	}
	f, err := s.GetFileByID(id)
	if err != nil {
		return nil, err
	}
	if !f.IsOwner(requesterID) {
		return nil, errors.New("permission denied")
	}
	return f, nil
}
func (s *FileService) GetAccessibleFileByID(id, requesterID, access string) (*models.File, error) {
	f, err := s.GetFileByID(id)
	if err != nil {
		return nil, err
	}
	if requesterID != "" && f.IsOwner(requesterID) {
		return f, nil
	}
	if requesterID == "" {
		return nil, errors.New("permission denied")
	}
	p, err := s.permissionRepo.GetPermission(requesterID, id)
	if err != nil || p == nil || !p.CanAccess(access) {
		return nil, errors.New("permission denied")
	}
	return f, nil
}

func (s *FileService) DownloadOwnedFile(ctx context.Context, id, requesterID string) (*models.File, io.ReadCloser, error) {
	f, err := s.GetAccessibleFileByID(id, requesterID, "read")
	if err != nil {
		return nil, nil, err
	}
	r, err := s.storage.DownloadFile(ctx, f.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("opening stored file: %w", err)
	}
	if err := s.recordAudit(requesterID, f.OwnerID, "file", f.ID, "download", map[string]any{"size_bytes": f.Size}); err != nil {
		_ = r.Close()
		return nil, nil, err
	}
	if s.activityRepo != nil {
		_ = s.activityRepo.Record(f.OwnerID, requesterID, &f.ID, f.Name, "download", f.Size)
	}
	return f, r, nil
}
func (s *FileService) ListFilesByOwner(ownerID string) ([]*models.File, error) {
	if ownerID == "" {
		return nil, errors.New("owner id is required")
	}
	return s.fileRepo.ListFilesByOwner(ownerID)
}
func (s *FileService) SearchFilesByOwner(ownerID, query, contentType, sortBy, direction string, folderID *string, limit, offset int) ([]*models.File, error) {
	if ownerID == "" {
		return nil, errors.New("owner id is required")
	}
	if sortBy == "" {
		sortBy = "updated_at"
	}
	switch sortBy {
	case "name", "size", "created_at", "updated_at":
	default:
		return nil, errors.New("sort must be name, size, created_at, or updated_at")
	}
	if direction == "" {
		direction = "desc"
	}
	if !strings.EqualFold(direction, "asc") && !strings.EqualFold(direction, "desc") {
		return nil, errors.New("order must be asc or desc")
	}
	if folderID != nil && *folderID != "" {
		if _, err := s.folderRepo.GetOwned(*folderID, ownerID); err != nil {
			return nil, errors.New("folder not found")
		}
	}
	return s.fileRepo.SearchOwned(ownerID, query, contentType, sortBy, direction, folderID, limit, offset)
}
func (s *FileService) UpdateFile(id, requesterID, name, contentType string, size int64) (*models.File, error) {
	if id == "" {
		return nil, errors.New("file id is required")
	}
	if contentType != "" || size > 0 {
		return nil, errors.New("content_type and size are immutable file properties")
	}
	f, err := s.GetAccessibleFileByID(id, requesterID, "write")
	if err != nil {
		return nil, err
	}
	if name != "" {
		if err := f.Rename(strings.TrimSpace(name)); err != nil {
			return nil, err
		}
	}
	if err := s.updateWithAudit(f, requesterID, "metadata_update", map[string]any{"name": f.Name}); err != nil {
		return nil, err
	}
	return f, nil
}

// DeleteFile can only enqueue deletion for the owner; a recipient's can_delete
// grant never authorizes destruction of the owner's canonical object.
func (s *FileService) DeleteFile(ctx context.Context, id, requesterID string) error {
	if id == "" {
		return errors.New("file id is required")
	}
	f, err := s.GetOwnedFileByID(id, requesterID)
	if err != nil {
		return err
	}
	return s.deleteRepo.Request(ctx, f, requesterID)
}
func (s *FileService) RemoveSharedReference(ctx context.Context, fileID, requesterID string) error {
	if requesterID == "" {
		return errors.New("permission denied")
	}
	f, err := s.GetFileByID(fileID)
	if err != nil {
		return err
	}
	if f.OwnerID == requesterID {
		return errors.New("owner cannot remove own reference")
	}
	if err := s.permissionRepo.DeletePermission(requesterID, fileID); err != nil {
		return err
	}
	return s.recordAudit(requesterID, f.OwnerID, "file", f.ID, "share_leave", map[string]any{})
}
func (s *FileService) recordAudit(actorID, ownerID, resourceType, resourceID, action string, details map[string]any) error {
	if s.auditRepo == nil {
		return errors.New("audit repository unavailable")
	}
	return s.auditRepo.Record(context.Background(), repository.AuditEvent{ActorID: actorID, ActorSubject: actorID, OwnerID: ownerID, ResourceType: resourceType, ResourceID: resourceID, Action: action, Outcome: "success", Details: details})
}

func (s *FileService) RunMaintenance(ctx context.Context) error {
	for i := 0; i < 20; i++ {
		job, err := s.deleteRepo.Claim(ctx)
		if err != nil {
			return err
		}
		if job == nil {
			break
		}
		if err := s.storage.DeleteFile(ctx, job.ObjectKey); err != nil {
			_ = s.deleteRepo.Fail(ctx, job, err)
			continue
		}
		if err := s.deleteRepo.Complete(ctx, job); err != nil {
			_ = s.deleteRepo.Fail(ctx, job, err)
		}
	}
	items, err := s.uploadRepo.ExpiredAll(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := s.storage.DeleteFile(ctx, item.ObjectKey); err != nil {
			continue
		}
		if err := s.uploadRepo.ReleaseExpired(ctx, item.ID); err != nil {
			return err
		}
	}
	return nil
}

type atomicFileMetadataAuditor interface {
	UpdateFileAndAudit(context.Context, *models.File, repository.AuditEvent) error
}

func (s *FileService) updateWithAudit(file *models.File, actorID, action string, details map[string]any) error {
	event := repository.AuditEvent{ActorID: actorID, ActorSubject: actorID, OwnerID: file.OwnerID, ResourceType: "file", ResourceID: file.ID, Action: action, Outcome: "success", Details: details}
	if atomic, ok := s.fileRepo.(atomicFileMetadataAuditor); ok {
		return atomic.UpdateFileAndAudit(context.Background(), file, event)
	}
	if err := s.fileRepo.UpdateFile(file); err != nil {
		return fmt.Errorf("persisting file metadata: %w", err)
	}
	if s.auditRepo == nil {
		return errors.New("audit repository unavailable")
	}
	return s.auditRepo.Record(context.Background(), event)
}

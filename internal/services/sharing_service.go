package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/souvik03-136/Go-Store/internal/models"
	"github.com/souvik03-136/Go-Store/internal/repository"
)

type SharingService struct {
	files *repository.FileRepository
	users *repository.UserRepository
	perms *repository.PermissionRepository
	audit *repository.AuditRepository
}

func NewSharingService(files *repository.FileRepository, users *repository.UserRepository, perms *repository.PermissionRepository, audit *repository.AuditRepository) *SharingService {
	return &SharingService{files: files, users: users, perms: perms, audit: audit}
}
func (s *SharingService) Grant(ownerID, fileID, recipientID string, canRead, canWrite, canDelete bool) (*models.Permission, error) {
	file, err := s.files.GetFileByID(fileID)
	if err != nil {
		return nil, err
	}
	if file.OwnerID != ownerID {
		return nil, errors.New("permission denied")
	}
	if recipientID == "" || recipientID == ownerID {
		return nil, errors.New("recipient must be another registered user")
	}
	if _, err := s.users.GetUserByID(recipientID); err != nil {
		return nil, errors.New("recipient not found")
	}
	if canWrite || canDelete {
		canRead = true
	}
	now := time.Now()
	p := &models.Permission{ID: uuid.NewString(), FileID: fileID, UserID: recipientID, CanRead: canRead, CanWrite: canWrite, CanDelete: canDelete, CreatedAt: now, UpdatedAt: now}
	event := repository.AuditEvent{ActorID: ownerID, ActorSubject: ownerID, OwnerID: file.OwnerID, ResourceType: "file", ResourceID: fileID, Action: "share_grant", Outcome: "success", Details: map[string]any{"recipient_id": recipientID, "can_read": canRead, "can_write": canWrite, "can_delete": canDelete}}
	if err := s.perms.UpsertPermissionAndAudit(context.Background(), p, event); err != nil {
		return nil, err
	}
	return p, nil
}
func (s *SharingService) List(ownerID, fileID string) ([]*models.Permission, error) {
	file, err := s.files.GetFileByID(fileID)
	if err != nil {
		return nil, err
	}
	if file.OwnerID != ownerID {
		return nil, errors.New("permission denied")
	}
	return s.perms.ListPermissionsByFile(fileID)
}
func (s *SharingService) Revoke(ownerID, fileID, recipientID string) error {
	file, err := s.files.GetFileByID(fileID)
	if err != nil {
		return err
	}
	if file.OwnerID != ownerID {
		return errors.New("permission denied")
	}
	event := repository.AuditEvent{ActorID: ownerID, ActorSubject: ownerID, OwnerID: file.OwnerID, ResourceType: "file", ResourceID: fileID, Action: "share_revoke", Outcome: "success", Details: map[string]any{"recipient_id": recipientID}}
	return s.perms.DeletePermissionAndAudit(context.Background(), recipientID, fileID, event)
}
func (s *SharingService) Leave(userID, fileID string) error {
	file, err := s.files.GetFileByID(fileID)
	if err != nil {
		return err
	}
	if file.OwnerID == userID {
		return errors.New("owner cannot remove own reference")
	}
	event := repository.AuditEvent{ActorID: userID, ActorSubject: userID, OwnerID: file.OwnerID, ResourceType: "file", ResourceID: fileID, Action: "share_leave", Outcome: "success", Details: map[string]any{}}
	return s.perms.DeletePermissionAndAudit(context.Background(), userID, fileID, event)
}
func (s *SharingService) SharedWithMe(userID string) ([]*models.File, error) {
	if userID == "" {
		return nil, errors.New("authentication required")
	}
	return s.files.ListSharedWithUser(userID)
}
func (s *SharingService) record(actor, owner, resource, action string, details map[string]any) error {
	if s.audit == nil {
		return errors.New("audit repository unavailable")
	}
	return s.audit.Record(context.Background(), repository.AuditEvent{ActorID: actor, ActorSubject: actor, OwnerID: owner, ResourceType: "file", ResourceID: resource, Action: action, Outcome: "success", Details: details})
}

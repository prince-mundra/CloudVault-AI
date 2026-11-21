package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/souvik03-136/Go-Store/internal/models"
	"github.com/souvik03-136/Go-Store/internal/repository"
	"github.com/souvik03-136/Go-Store/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

const (
	maxShareLinkAge     = 30 * 24 * time.Hour
	defaultShareLinkAge = 7 * 24 * time.Hour
)

type ShareLinkService struct {
	files    *repository.FileRepository
	links    *repository.ShareLinkRepository
	storage  storage.Storage
	audit    *repository.AuditRepository
	activity *repository.FileActivityRepository
}

func NewShareLinkService(files *repository.FileRepository, links *repository.ShareLinkRepository, store storage.Storage, audit *repository.AuditRepository, activity *repository.FileActivityRepository) *ShareLinkService {
	return &ShareLinkService{files: files, links: links, storage: store, audit: audit, activity: activity}
}

func (s *ShareLinkService) Create(ctx context.Context, ownerID, fileID, password string, lifetime time.Duration) (*models.ShareLink, string, error) {
	file, err := s.files.GetFileByID(fileID)
	if err != nil {
		return nil, "", err
	}
	if file.OwnerID != ownerID {
		return nil, "", errors.New("permission denied")
	}
	if lifetime <= 0 {
		lifetime = defaultShareLinkAge
	}
	if lifetime > maxShareLinkAge {
		return nil, "", errors.New("share link lifetime cannot exceed 30 days")
	}
	if len(password) > 72 {
		return nil, "", errors.New("share password must be at most 72 bytes")
	}
	if password != "" && len(password) < 8 {
		return nil, "", errors.New("share password must be at least 8 bytes")
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", fmt.Errorf("generating share token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	digest := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(digest[:])
	var passwordHash string
	if password != "" {
		b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, "", fmt.Errorf("hashing share password: %w", err)
		}
		passwordHash = string(b)
	}
	link := &models.ShareLink{ID: uuid.NewString(), FileID: fileID, OwnerID: ownerID, PasswordProtected: passwordHash != "", ExpiresAt: time.Now().Add(lifetime), TokenHash: hash, PasswordHash: passwordHash}
	if err := s.links.Create(ctx, link); err != nil {
		return nil, "", err
	}
	if err := s.record(ctx, ownerID, file.OwnerID, link.ID, "share_link_create", map[string]any{"file_id": fileID, "expires_at": link.ExpiresAt, "password_protected": link.PasswordProtected}); err != nil {
		_ = s.links.Revoke(ctx, link.ID, ownerID)
		return nil, "", err
	}
	return link, token, nil
}
func (s *ShareLinkService) List(ctx context.Context, ownerID, fileID string) ([]*models.ShareLink, error) {
	f, err := s.files.GetFileByID(fileID)
	if err != nil {
		return nil, err
	}
	if f.OwnerID != ownerID {
		return nil, errors.New("permission denied")
	}
	return s.links.ListByOwner(ctx, ownerID, fileID)
}
func (s *ShareLinkService) Revoke(ctx context.Context, ownerID, fileID, linkID string) error {
	f, err := s.files.GetFileByID(fileID)
	if err != nil {
		return err
	}
	if f.OwnerID != ownerID {
		return errors.New("permission denied")
	}
	if err := s.links.Revoke(ctx, linkID, ownerID); err != nil {
		return err
	}
	return s.record(ctx, ownerID, ownerID, linkID, "share_link_revoke", map[string]any{"file_id": fileID})
}

func (s *ShareLinkService) Download(ctx context.Context, token, password string) (*models.File, io.ReadCloser, error) {
	if token == "" || len(token) > 256 {
		return nil, nil, errors.New("share link not found or expired")
	}
	sum := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(sum[:])
	link, err := s.links.FindActive(ctx, hash)
	if err != nil {
		return nil, nil, err
	}
	if link.PasswordProtected {
		if bcrypt.CompareHashAndPassword([]byte(link.PasswordHash), []byte(password)) != nil {
			_ = s.recordAs(ctx, "", "sharelink:"+link.ID, "", link.ID, "share_link_access_denied", "failure", map[string]any{"reason": "invalid_password"})
			return nil, nil, errors.New("share password is required or invalid")
		}
	}
	file, err := s.files.GetFileByID(link.FileID)
	if err != nil {
		return nil, nil, errors.New("file is unavailable")
	}
	if file.OwnerID != link.OwnerID {
		return nil, nil, errors.New("share link is invalid")
	}
	r, err := s.storage.DownloadFile(ctx, file.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("opening shared object: %w", err)
	}
	if err := s.links.Touch(ctx, link.ID); err != nil {
		_ = r.Close()
		return nil, nil, err
	}
	if err := s.recordAs(ctx, "", "sharelink:"+link.ID, file.OwnerID, link.ID, "share_link_download", "success", map[string]any{"file_id": file.ID, "size_bytes": file.Size}); err != nil {
		_ = r.Close()
		return nil, nil, err
	}
	if s.activity != nil {
		_ = s.activity.Record(file.OwnerID, "", &file.ID, file.Name, "download", file.Size)
	}
	return file, r, nil
}
func (s *ShareLinkService) record(ctx context.Context, actorID, ownerID, resourceID, action string, details map[string]any) error {
	subject := actorID
	if subject == "" {
		subject = "anonymous"
	}
	return s.recordAs(ctx, actorID, subject, ownerID, resourceID, action, "success", details)
}
func (s *ShareLinkService) recordAs(ctx context.Context, actorID, actorSubject, ownerID, resourceID, action, outcome string, details map[string]any) error {
	if s.audit == nil {
		return errors.New("audit repository unavailable")
	}
	return s.audit.Record(ctx, repository.AuditEvent{ActorID: actorID, ActorSubject: actorSubject, OwnerID: ownerID, ResourceType: "share_link", ResourceID: resourceID, Action: action, Outcome: outcome, Details: details})
}

package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/souvik03-136/Go-Store/internal/models"
	"github.com/souvik03-136/Go-Store/internal/repository"
)

type FolderService struct{ repo *repository.FolderRepository }

func NewFolderService(repo *repository.FolderRepository) *FolderService {
	return &FolderService{repo: repo}
}

func (s *FolderService) Create(ownerID, name string, parentID *string) (*models.Folder, error) {
	name = strings.TrimSpace(name)
	if ownerID == "" || name == "" {
		return nil, errors.New("owner and folder name are required")
	}
	if len(name) > 255 {
		return nil, errors.New("folder name exceeds 255 characters")
	}
	if parentID != nil {
		if _, err := s.repo.GetOwned(*parentID, ownerID); err != nil {
			return nil, errors.New("parent folder not found")
		}
	}
	now := time.Now()
	f := &models.Folder{ID: generateID(), OwnerID: ownerID, ParentID: parentID, Name: name, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.Create(f); err != nil {
		return nil, fmt.Errorf("saving folder: %w", err)
	}
	return f, nil
}

func (s *FolderService) List(ownerID string, parentID *string) ([]*models.Folder, error) {
	if ownerID == "" {
		return nil, errors.New("owner is required")
	}
	if parentID != nil {
		if _, err := s.repo.GetOwned(*parentID, ownerID); err != nil {
			return nil, errors.New("parent folder not found")
		}
	}
	return s.repo.List(ownerID, parentID)
}

func (s *FolderService) Update(ownerID, id, name string, parentID *string, parentProvided bool) (*models.Folder, error) {
	f, err := s.repo.GetOwned(id, ownerID)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name != "" {
		f.Name = name
	}
	if len(f.Name) > 255 {
		return nil, errors.New("folder name exceeds 255 characters")
	}
	if parentProvided {
		if parentID != nil {
			if *parentID == id {
				return nil, errors.New("folder cannot be its own parent")
			}
			if _, err := s.repo.GetOwned(*parentID, ownerID); err != nil {
				return nil, errors.New("parent folder not found")
			}
			for cursor := parentID; cursor != nil; {
				if *cursor == id {
					return nil, errors.New("folder move would create a cycle")
				}
				ancestor, err := s.repo.GetOwned(*cursor, ownerID)
				if err != nil {
					return nil, errors.New("parent folder not found")
				}
				cursor = ancestor.ParentID
			}
		}
		f.ParentID = parentID
	}
	if err := s.repo.Update(f); err != nil {
		return nil, fmt.Errorf("saving folder changes: %w", err)
	}
	return f, nil
}

func (s *FolderService) Delete(ownerID, id string) error {
	if _, err := s.repo.GetOwned(id, ownerID); err != nil {
		return err
	}
	empty, err := s.repo.IsEmpty(id)
	if err != nil {
		return err
	}
	if !empty {
		return errors.New("folder is not empty")
	}
	return s.repo.Delete(id, ownerID)
}

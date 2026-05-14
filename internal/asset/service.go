package asset

import (
	"errors"
	"team-management/internal/models"
	"time"
)

type AssetService interface {
	GetNoteByID(requesterID, noteID int64) (*models.Note, error)
	UpdateNote(requesterID, noteID int64, title, content string, folderID *int64) (*models.Note, error)
	CreateNote(requesterID int64, title, content string, folderID *int64) (*models.Note, error)
	GetUserNotes(requesterID int64) ([]*models.Note, error)
	DeleteNote(requesterID, noteID int64) error
	ShareNote(requesterID, noteID, sharedWithUserID int64, permission string) error
	RemoveNoteShare(requesterID, noteID, sharedWithUserID int64) error
	GetNoteShares(requesterID, noteID int64) ([]*models.NoteShare, error)
	CreateFolder(requesterID int64, name string) (*models.Folder, error)
	GetUserFolders(requesterID int64) ([]*models.Folder, error)
	GetSharedNotes(requesterID int64) ([]*models.Note, error)
	DeleteFolder(requesterID, folderID int64) error
}

type assetServiceImpl struct {
	repo AssetRepository
}

func NewAssetService(repo AssetRepository) AssetService {
	return &assetServiceImpl{repo: repo}
}

func (s *assetServiceImpl) GetNoteByID(requesterID, noteID int64) (*models.Note, error) {
	note, err := s.repo.getNoteByID(noteID)
	if err != nil {
		return nil, err
	}

	// Check if requester is the owner of the note
	if note.OwnerID == requesterID {
		return note, nil
	}

	// Check folder share level
	if note.FolderID > 0 {
		folderShareLevel, err := s.repo.GetFolderShareLevel(note.FolderID, requesterID)
		if err != nil {
			return nil, err
		}
		if folderShareLevel == "write" || folderShareLevel == "read" {
			return note, nil
		}
	}

	// Check note share level
	noteShareLevel, err := s.repo.GetNoteShareLevel(note.ID, requesterID)
	if err != nil {
		return nil, err
	}
	if noteShareLevel == "write" || noteShareLevel == "read" {
		return note, nil
	}

	// Check if requester is a manager of the owner
	isManager, err := s.repo.IsManagerOfOwner(requesterID, note.OwnerID)
	if err != nil {
		return nil, err
	}
	if isManager {
		return note, nil
	}

	return nil, errors.New("access denied: you do not have permission to view this note")
}

func (s *assetServiceImpl) UpdateNote(requesterID, noteID int64, title, content string, folderID *int64) (*models.Note, error) {
	note, err := s.repo.getNoteByID(noteID)
	if err != nil {
		return nil, err
	}

	// Check if requester is the owner of the note
	if note.OwnerID == requesterID {
		note.Title = title
		note.Content = content
		if folderID != nil {
			note.FolderID = *folderID
		}
		note.UpdatedAt = time.Now()
		err = s.repo.updateNote(note)
		if err != nil {
			return nil, err
		}
		return note, nil
	}

	// Check folder share level
	if note.FolderID > 0 {
		folderShareLevel, err := s.repo.GetFolderShareLevel(note.FolderID, requesterID)
		if err != nil {
			return nil, err
		}
		if folderShareLevel == "write" {
			note.Title = title
			note.Content = content
			if folderID != nil {
				note.FolderID = *folderID
			}
			note.UpdatedAt = time.Now()
			err = s.repo.updateNote(note)
			if err != nil {
				return nil, err
			}
			return note, nil
		}
	}

	// Check note share level
	noteShareLevel, err := s.repo.GetNoteShareLevel(note.ID, requesterID)
	if err != nil {
		return nil, err
	}
	if noteShareLevel == "write" {
		note.Title = title
		note.Content = content
		if folderID != nil {
			note.FolderID = *folderID
		}
		note.UpdatedAt = time.Now()
		err = s.repo.updateNote(note)
		if err != nil {
			return nil, err
		}
		return note, nil
	}

	return nil, errors.New("access denied: you do not have permission to edit this note")
}

func (s *assetServiceImpl) CreateNote(requesterID int64, title, content string, folderID *int64) (*models.Note, error) {
	note := &models.Note{
		OwnerID: requesterID,
		Title:   title,
		Content: content,
	}
	if folderID != nil {
		note.FolderID = *folderID
	}

	return s.repo.createNote(note)
}

func (s *assetServiceImpl) GetUserNotes(requesterID int64) ([]*models.Note, error) {
	return s.repo.getUserNotes(requesterID)
}

func (s *assetServiceImpl) DeleteNote(requesterID, noteID int64) error {
	note, err := s.repo.getNoteByID(noteID)
	if err != nil {
		return err
	}

	// Only owner can delete
	if note.OwnerID != requesterID {
		return errors.New("access denied: only owner can delete this note")
	}

	return s.repo.deleteNote(noteID)
}

func (s *assetServiceImpl) ShareNote(requesterID, noteID, sharedWithUserID int64, permission string) error {
	note, err := s.repo.getNoteByID(noteID)
	if err != nil {
		return err
	}

	// Only owner can share
	if note.OwnerID != requesterID {
		return errors.New("access denied: only owner can share this note")
	}

	if permission != "read" && permission != "write" {
		return errors.New("invalid permission level")
	}

	return s.repo.shareNote(&models.NoteShare{
		NoteID:           noteID,
		SharedWithUserID: sharedWithUserID,
		PermissionLevel:  permission,
	})
}

func (s *assetServiceImpl) RemoveNoteShare(requesterID, noteID, sharedWithUserID int64) error {
	note, err := s.repo.getNoteByID(noteID)
	if err != nil {
		return err
	}

	// Only owner can remove share
	if note.OwnerID != requesterID {
		return errors.New("access denied: only owner can modify shares")
	}

	return s.repo.removeNoteShare(noteID, sharedWithUserID)
}

func (s *assetServiceImpl) GetNoteShares(requesterID, noteID int64) ([]*models.NoteShare, error) {
	note, err := s.repo.getNoteByID(noteID)
	if err != nil {
		return nil, err
	}

	// Only owner can view shares
	if note.OwnerID != requesterID {
		return nil, errors.New("access denied")
	}

	return s.repo.getNoteShares(noteID)
}

func (s *assetServiceImpl) CreateFolder(requesterID int64, name string) (*models.Folder, error) {
	folder := &models.Folder{
		OwnerID: requesterID,
		Name:    name,
	}
	return s.repo.createFolder(folder)
}

func (s *assetServiceImpl) GetUserFolders(requesterID int64) ([]*models.Folder, error) {
	return s.repo.getUserFolders(requesterID)
}

func (s *assetServiceImpl) GetSharedNotes(requesterID int64) ([]*models.Note, error) {
	return s.repo.getSharedNotes(requesterID)
}

func (s *assetServiceImpl) DeleteFolder(requesterID, folderID int64) error {
	folder, err := s.repo.getFolderByID(folderID)
	if err != nil {
		return err
	}

	// Only owner can delete
	if folder.OwnerID != requesterID {
		return errors.New("access denied: only owner can delete this folder")
	}

	return s.repo.deleteFolder(folderID)
}

package asset

import (
	"errors"
	"team-management/internal/models"
)

type AssetService interface {
	GetNoteByID(requesterID, noteID int64) (*models.Note, error)
	UpdateNote(requesterID, noteID int64, title, content string) (*models.Note, error)
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
	folderShareLevel, err := s.repo.GetFolderShareLevel(note.FolderID, requesterID)
	if err != nil {
		return nil, err
	}
	if folderShareLevel == "write" || folderShareLevel == "read" {
		return note, nil
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

func (s *assetServiceImpl) UpdateNote(requesterID, noteID int64, title, content string) (*models.Note, error) {
	note, err := s.repo.getNoteByID(noteID)
	if err != nil {
		return nil, err
	}

	// Check if requester is the owner of the note
	if note.OwnerID == requesterID {
		note.Title = title
		note.Content = content
		err = s.repo.updateNote(note)
		if err != nil {
			return nil, err
		}
		return note, nil
	}

	// Check folder share level
	folderShareLevel, err := s.repo.GetFolderShareLevel(note.FolderID, requesterID)
	if err != nil {
		return nil, err
	}
	if folderShareLevel == "write" {
		note.Title = title
		note.Content = content
		err = s.repo.updateNote(note)
		if err != nil {
			return nil, err
		}
		return note, nil
	}

	// Check note share level
	noteShareLevel, err := s.repo.GetNoteShareLevel(note.ID, requesterID)
	if err != nil {
		return nil, err
	}
	if noteShareLevel == "write" {
		note.Title = title
		note.Content = content
		err = s.repo.updateNote(note)
		if err != nil {
			return nil, err
		}
		return note, nil
	}

	return nil, errors.New("access denied: you do not have permission to edit this note")
}

package asset

import (
	"context"
	"team-management/internal/models"
	apperrors "team-management/internal/utils"
	"time"

	"team-management/internal/audit"

	"github.com/redis/go-redis/v9"
)

type AssetService interface {
	GetNoteByID(ctx context.Context, requesterID, noteID int64) (*models.Note, error)
	UpdateNote(ctx context.Context, requesterID, noteID int64, title, content string, folderID *int64) (*models.Note, error)
	CreateNote(ctx context.Context, requesterID int64, title, content string, folderID *int64) (*models.Note, error)
	GetUserNotes(ctx context.Context, requesterID int64) ([]*models.Note, error)
	DeleteNote(ctx context.Context, requesterID, noteID int64) error
	ShareNote(ctx context.Context, requesterID, noteID, sharedWithUserID int64, permission string) error
	RemoveNoteShare(ctx context.Context, requesterID, noteID, sharedWithUserID int64) error
	GetNoteShares(ctx context.Context, requesterID, noteID int64) ([]*models.NoteShare, error)
	CreateFolder(ctx context.Context, requesterID int64, name string) (*models.Folder, error)
	GetUserFolders(ctx context.Context, requesterID int64) ([]*models.Folder, error)
	GetSharedNotes(ctx context.Context, requesterID int64) ([]*models.Note, error)
	DeleteFolder(ctx context.Context, requesterID, folderID int64) error
	GetNoteAccess(ctx context.Context, requesterID, noteID int64) ([]map[string]interface{}, error)
}

type assetServiceImpl struct {
	repo        AssetRepository
	redisClient *redis.Client
}

func NewAssetService(repo AssetRepository, redisClient *redis.Client) AssetService {
	return &assetServiceImpl{repo: repo, redisClient: redisClient}
}

func (s *assetServiceImpl) GetNoteByID(ctx context.Context, requesterID, noteID int64) (*models.Note, error) {
	note, err := s.repo.GetNoteByID(ctx, noteID)
	if err != nil {
		return nil, err
	}

	// Check if requester is the owner of the note
	if note.OwnerID == requesterID {
		return note, nil
	}

	// Check folder share level
	if note.FolderID > 0 {
		folderShareLevel, err := s.repo.GetFolderShareLevel(ctx, note.FolderID, requesterID)
		if err != nil {
			return nil, err
		}
		if folderShareLevel == "write" || folderShareLevel == "read" {
			return note, nil
		}
	}

	// Check note share level
	noteShareLevel, err := s.repo.GetNoteShareLevel(ctx, note.ID, requesterID)
	if err != nil {
		return nil, err
	}
	if noteShareLevel == "write" || noteShareLevel == "read" {
		return note, nil
	}

	// Check if requester is a manager of the owner
	isManager, err := s.repo.IsManagerOfOwner(ctx, requesterID, note.OwnerID)
	if err != nil {
		return nil, err
	}
	if isManager {
		return note, nil
	}

	return nil, apperrors.NewForbiddenError("you do not have permission to view this note")
}

func (s *assetServiceImpl) UpdateNote(ctx context.Context, requesterID, noteID int64, title, content string, folderID *int64) (*models.Note, error) {
	note, err := s.repo.GetNoteByID(ctx, noteID)
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
		err = s.repo.UpdateNote(ctx, note)
		if err != nil {
			return nil, err
		}
		return note, nil
	}

	// Check folder share level
	if note.FolderID > 0 {
		folderShareLevel, err := s.repo.GetFolderShareLevel(ctx, note.FolderID, requesterID)
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
			err = s.repo.UpdateNote(ctx, note)
			if err != nil {
				return nil, err
			}
			return note, nil
		}
	}

	// Check note share level
	noteShareLevel, err := s.repo.GetNoteShareLevel(ctx, note.ID, requesterID)
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
		err = s.repo.UpdateNote(ctx, note)
		if err != nil {
			return nil, err
		}
		return note, nil
	}

	audit.PublishEvent(s.redisClient, &requesterID, "UNAUTHORIZED_NOTE_UPDATE_ATTEMPT", "note", &noteID, nil)

	return nil, apperrors.NewForbiddenError("you do not have permission to edit this note")
}

func (s *assetServiceImpl) CreateNote(ctx context.Context, requesterID int64, title, content string, folderID *int64) (*models.Note, error) {
	note := &models.Note{
		OwnerID: requesterID,
		Title:   title,
		Content: content,
	}
	if folderID != nil {
		note.FolderID = *folderID
	}

	return s.repo.CreateNote(ctx, note)
}

func (s *assetServiceImpl) GetUserNotes(ctx context.Context, requesterID int64) ([]*models.Note, error) {
	return s.repo.GetUserNotes(ctx, requesterID)
}

func (s *assetServiceImpl) DeleteNote(ctx context.Context, requesterID, noteID int64) error {
	note, err := s.repo.GetNoteByID(ctx, noteID)
	if err != nil {
		return err
	}

	// Only owner can delete
	if note.OwnerID != requesterID {
		return apperrors.NewForbiddenError("only owner can delete this note")
	}

	audit.PublishEvent(s.redisClient, &requesterID, "NOTE_DELETED", "note", &noteID, nil)

	return s.repo.DeleteNote(ctx, noteID)
}

func (s *assetServiceImpl) ShareNote(ctx context.Context, requesterID, noteID, sharedWithUserID int64, permission string) error {
	note, err := s.repo.GetNoteByID(ctx, noteID)
	if err != nil {
		return err
	}

	// Only owner can share
	if note.OwnerID != requesterID {
		return apperrors.NewForbiddenError("only owner can share this note")
	}

	if permission != "read" && permission != "write" {
		return apperrors.NewValidationError("permission", "invalid permission level")
	}

	audit.PublishEvent(s.redisClient, &requesterID, "NOTE_SHARED", "note", &noteID, map[string]any{"shared_with_user_id": sharedWithUserID, "permission": permission})

	return s.repo.ShareNote(ctx, &models.NoteShare{
		NoteID:           noteID,
		SharedWithUserID: sharedWithUserID,
		PermissionLevel:  permission,
	})
}

func (s *assetServiceImpl) RemoveNoteShare(ctx context.Context, requesterID, noteID, sharedWithUserID int64) error {
	note, err := s.repo.GetNoteByID(ctx, noteID)
	if err != nil {
		return err
	}

	// Only owner can remove share
	if note.OwnerID != requesterID {
		return apperrors.NewForbiddenError("only owner can modify shares")
	}

	audit.PublishEvent(s.redisClient, &requesterID, "NOTE_SHARE_REMOVED", "note", &noteID, map[string]any{"shared_with_user_id": sharedWithUserID})
	return s.repo.RemoveNoteShare(ctx, noteID, sharedWithUserID)
}

func (s *assetServiceImpl) GetNoteShares(ctx context.Context, requesterID, noteID int64) ([]*models.NoteShare, error) {
	note, err := s.repo.GetNoteByID(ctx, noteID)
	if err != nil {
		return nil, err
	}

	// Only owner can view shares
	if note.OwnerID != requesterID {
		return nil, apperrors.NewForbiddenError("only owner can view shares")
	}

	return s.repo.GetNoteShares(ctx, noteID)
}

func (s *assetServiceImpl) CreateFolder(ctx context.Context, requesterID int64, name string) (*models.Folder, error) {
	folder := &models.Folder{
		OwnerID: requesterID,
		Name:    name,
	}

	audit.PublishEvent(s.redisClient, &requesterID, "FOLDER_CREATED", "folder", nil, map[string]any{"folder_name": name})

	return s.repo.CreateFolder(ctx, folder)
}

func (s *assetServiceImpl) GetUserFolders(ctx context.Context, requesterID int64) ([]*models.Folder, error) {
	return s.repo.GetUserFolders(ctx, requesterID)
}

func (s *assetServiceImpl) GetSharedNotes(ctx context.Context, requesterID int64) ([]*models.Note, error) {
	return s.repo.GetSharedNotes(ctx, requesterID)
}

func (s *assetServiceImpl) DeleteFolder(ctx context.Context, requesterID, folderID int64) error {
	folder, err := s.repo.GetFolderByID(ctx, folderID)
	if err != nil {
		return err
	}

	// Only owner can delete
	if folder.OwnerID != requesterID {
		return apperrors.NewForbiddenError("only owner can delete this folder")
	}

	audit.PublishEvent(s.redisClient, &requesterID, "FOLDER_DELETED", "folder", &folderID, nil)
	return s.repo.DeleteFolder(ctx, folderID)
}

func permissionPriority(p string) int {
	switch p {
	case "owner":
		return 3
	case "write":
		return 2
	case "read":
		return 1
	default:
		return 0
	}
}

func (s *assetServiceImpl) GetNoteAccess(ctx context.Context, requesterID, noteID int64) ([]map[string]interface{}, error) {
	note, err := s.repo.GetNoteByID(ctx, noteID)
	if err != nil {
		return nil, err
	}

	// Only owner may view the full access list
	if note.OwnerID != requesterID {
		return nil, apperrors.NewForbiddenError("only owner can view access list")
	}

	accessMap := make(map[int64]string)
	// Owner
	accessMap[note.OwnerID] = "owner"

	// Note shares
	noteShares, err := s.repo.GetNoteShares(ctx, noteID)
	if err == nil {
		for _, ns := range noteShares {
			if ns == nil {
				continue
			}
			cur, ok := accessMap[ns.SharedWithUserID]
			if !ok || permissionPriority(ns.PermissionLevel) > permissionPriority(cur) {
				accessMap[ns.SharedWithUserID] = ns.PermissionLevel
			}
		}
	}

	// Folder shares
	if note.FolderID > 0 {
		folderShares, err := s.repo.GetFolderShares(ctx, note.FolderID)
		if err == nil {
			for _, fs := range folderShares {
				if fs == nil {
					continue
				}
				cur, ok := accessMap[fs.SharedWithUserID]
				if !ok || permissionPriority(fs.PermissionLevel) > permissionPriority(cur) {
					accessMap[fs.SharedWithUserID] = fs.PermissionLevel
				}
			}
		}
	}

	// Managers of owner get write access
	managers, err := s.repo.GetManagersOfOwner(ctx, note.OwnerID)
	if err == nil {
		for _, m := range managers {
			cur, ok := accessMap[m]
			if !ok || permissionPriority("write") > permissionPriority(cur) {
				accessMap[m] = "write"
			}
		}
	}

	// Build result slice
	var res []map[string]interface{}
	for uid, perm := range accessMap {
		res = append(res, map[string]interface{}{
			"user_id":    uid,
			"permission": perm,
		})
	}
	return res, nil
}

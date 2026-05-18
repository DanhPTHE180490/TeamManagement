package asset

import (
	"database/sql"
	apperrors "team-management/internal/errors"
	"team-management/internal/models"
	"time"
)

type AssetRepository interface {
	GetNoteByID(id int64) (*models.Note, error)
	GetFolderShareLevel(folderID, userID int64) (string, error)
	GetNoteShareLevel(noteID, userID int64) (string, error)
	IsManagerOfOwner(requesterID, ownerID int64) (bool, error)
	UpdateNote(note *models.Note) error
	CreateNote(note *models.Note) (*models.Note, error)
	GetUserNotes(userID int64) ([]*models.Note, error)
	DeleteNote(noteID int64) error
	ShareNote(noteShare *models.NoteShare) error
	RemoveNoteShare(noteID, userID int64) error
	GetNoteShares(noteID int64) ([]*models.NoteShare, error)
	CreateFolder(folder *models.Folder) (*models.Folder, error)
	GetUserFolders(userID int64) ([]*models.Folder, error)
	GetFolderByID(folderID int64) (*models.Folder, error)
	DeleteFolder(folderID int64) error
	ShareFolder(folderShare *models.FolderShare) error
	RemoveFolderShare(folderID, userID int64) error
	GetSharedNotes(userID int64) ([]*models.Note, error)
	GetFolderShares(folderID int64) ([]*models.FolderShare, error)
	GetManagersOfOwner(ownerID int64) ([]int64, error)
}

type assetRepositoryImpl struct {
	db *sql.DB
}

func NewAssetRepository(db *sql.DB) AssetRepository {
	return &assetRepositoryImpl{db: db}
}

func (r *assetRepositoryImpl) GetNoteByID(id int64) (*models.Note, error) {
	var note models.Note
	var content sql.NullString
	err := r.db.QueryRow("SELECT id, folder_id, owner_id, title, content, created_at, updated_at FROM notes WHERE id = ?", id).
		Scan(&note.ID, &note.FolderID, &note.OwnerID, &note.Title, &content, &note.CreatedAt, &note.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.NewNotFoundError("note")
		}
		return nil, apperrors.NewInternalError("failed to query note", err)
	}
	if content.Valid {
		note.Content = content.String
	}
	return &note, nil
}

func (r *assetRepositoryImpl) CreateNote(note *models.Note) (*models.Note, error) {
	now := time.Now()
	result, err := r.db.Exec(
		"INSERT INTO notes (folder_id, owner_id, title, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		note.FolderID, note.OwnerID, note.Title, note.Content, now, now,
	)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to create note", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, apperrors.NewInternalError("failed to fetch user notes", err)
	}

	note.ID = id
	note.CreatedAt = now
	note.UpdatedAt = now
	return note, nil
}

func (r *assetRepositoryImpl) GetUserNotes(userID int64) ([]*models.Note, error) {
	rows, err := r.db.Query("SELECT id, folder_id, owner_id, title, content, created_at, updated_at FROM notes WHERE owner_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []*models.Note
	for rows.Next() {
		var note models.Note
		var content sql.NullString
		if err := rows.Scan(&note.ID, &note.FolderID, &note.OwnerID, &note.Title, &content, &note.CreatedAt, &note.UpdatedAt); err != nil {
			return nil, err
		}
		if content.Valid {
			note.Content = content.String
		}
		notes = append(notes, &note)
	}

	return notes, rows.Err()
}

func (r *assetRepositoryImpl) DeleteNote(noteID int64) error {
	_, err := r.db.Exec("DELETE FROM notes WHERE id = ?", noteID)
	if err != nil {
		return apperrors.NewInternalError("failed to delete note", err)
	}
	return nil
}

func (r *assetRepositoryImpl) ShareNote(noteShare *models.NoteShare) error {
	_, err := r.db.Exec(
		"INSERT INTO note_shares (note_id, shared_with_user_id, permission_level) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE permission_level = ?",
		noteShare.NoteID, noteShare.SharedWithUserID, noteShare.PermissionLevel, noteShare.PermissionLevel,
	)
	if err != nil {
		return apperrors.NewInternalError("failed to share note", err)
	}
	return nil
}

func (r *assetRepositoryImpl) RemoveNoteShare(noteID, userID int64) error {
	_, err := r.db.Exec("DELETE FROM note_shares WHERE note_id = ? AND shared_with_user_id = ?", noteID, userID)
	if err != nil {
		return apperrors.NewInternalError("failed to remove note share", err)
	}
	return nil
}

func (r *assetRepositoryImpl) GetNoteShares(noteID int64) ([]*models.NoteShare, error) {
	rows, err := r.db.Query("SELECT note_id, shared_with_user_id, permission_level FROM note_shares WHERE note_id = ?", noteID)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to query note shares", err)
	}
	defer rows.Close()

	var shares []*models.NoteShare
	for rows.Next() {
		var share models.NoteShare
		if err := rows.Scan(&share.NoteID, &share.SharedWithUserID, &share.PermissionLevel); err != nil {
			return nil, apperrors.NewInternalError("failed to scan note share", err)
		}
		shares = append(shares, &share)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.NewInternalError("note shares rows iteration error", err)
	}
	return shares, nil
}

func (r *assetRepositoryImpl) GetFolderShares(folderID int64) ([]*models.FolderShare, error) {
	rows, err := r.db.Query("SELECT folder_id, shared_with_user_id, permission_level FROM folder_shares WHERE folder_id = ?", folderID)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to query folder shares", err)
	}
	defer rows.Close()

	var shares []*models.FolderShare
	for rows.Next() {
		var s models.FolderShare
		if err := rows.Scan(&s.FolderID, &s.SharedWithUserID, &s.PermissionLevel); err != nil {
			return nil, apperrors.NewInternalError("failed to scan folder share", err)
		}
		shares = append(shares, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.NewInternalError("folder shares rows iteration error", err)
	}
	return shares, nil
}

func (r *assetRepositoryImpl) GetManagersOfOwner(ownerID int64) ([]int64, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT req_tm.user_id
		FROM team_members req_tm
		JOIN team_members owner_tm ON req_tm.team_id = owner_tm.team_id
		WHERE owner_tm.user_id = ?
		  AND req_tm.team_role IN ('manager','main_manager')
	`, ownerID)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to query managers", err)
	}
	defer rows.Close()

	var managers []int64
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil {
			return nil, apperrors.NewInternalError("failed to scan manager id", err)
		}
		managers = append(managers, uid)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.NewInternalError("managers rows iteration error", err)
	}
	return managers, nil
}

func (r *assetRepositoryImpl) CreateFolder(folder *models.Folder) (*models.Folder, error) {
	now := time.Now()
	result, err := r.db.Exec(
		"INSERT INTO folders (name, owner_id, created_at, updated_at) VALUES (?, ?, ?, ?)",
		folder.Name, folder.OwnerID, now, now,
	)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to create folder", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get user folders", err)
	}

	folder.ID = id
	folder.CreatedAt = now
	folder.UpdatedAt = now
	return folder, nil
}

func (r *assetRepositoryImpl) GetUserFolders(userID int64) ([]*models.Folder, error) {
	rows, err := r.db.Query("SELECT id, name, owner_id, created_at, updated_at FROM folders WHERE owner_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []*models.Folder
	for rows.Next() {
		var folder models.Folder
		if err := rows.Scan(&folder.ID, &folder.Name, &folder.OwnerID, &folder.CreatedAt, &folder.UpdatedAt); err != nil {
			return nil, err
		}
		folders = append(folders, &folder)
	}

	return folders, rows.Err()
}

func (r *assetRepositoryImpl) GetFolderByID(folderID int64) (*models.Folder, error) {
	var folder models.Folder
	err := r.db.QueryRow("SELECT id, name, owner_id, created_at, updated_at FROM folders WHERE id = ?", folderID).
		Scan(&folder.ID, &folder.Name, &folder.OwnerID, &folder.CreatedAt, &folder.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.NewNotFoundError("folder")
		}
		return nil, apperrors.NewInternalError("failed to query folder", err)
	}
	return &folder, nil
}

func (r *assetRepositoryImpl) DeleteFolder(folderID int64) error {
	_, err := r.db.Exec("DELETE FROM folders WHERE id = ?", folderID)
	if err != nil {
		return apperrors.NewInternalError("failed to delete folder", err)
	}
	return nil
}

func (r *assetRepositoryImpl) ShareFolder(folderShare *models.FolderShare) error {
	_, err := r.db.Exec(
		"INSERT INTO folder_shares (folder_id, shared_with_user_id, permission_level) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE permission_level = ?",
		folderShare.FolderID, folderShare.SharedWithUserID, folderShare.PermissionLevel, folderShare.PermissionLevel,
	)
	if err != nil {
		return apperrors.NewInternalError("failed to share folder", err)
	}
	return nil
}

func (r *assetRepositoryImpl) RemoveFolderShare(folderID, userID int64) error {
	_, err := r.db.Exec("DELETE FROM folder_shares WHERE folder_id = ? AND shared_with_user_id = ?", folderID, userID)
	if err != nil {
		return apperrors.NewInternalError("failed to remove folder share", err)
	}
	return nil
}

func (r *assetRepositoryImpl) GetSharedNotes(userID int64) ([]*models.Note, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT n.id, n.folder_id, n.owner_id, n.title, n.content, n.created_at, n.updated_at
		FROM notes n
		LEFT JOIN note_shares ns ON n.id = ns.note_id
		LEFT JOIN folder_shares fs ON n.folder_id = fs.folder_id
		WHERE (ns.shared_with_user_id = ? OR fs.shared_with_user_id = ?) AND n.owner_id != ?
		ORDER BY n.created_at DESC
	`, userID, userID, userID)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to query shared notes", err)
	}
	defer rows.Close()

	var notes []*models.Note
	for rows.Next() {
		var note models.Note
		if err := rows.Scan(&note.ID, &note.FolderID, &note.OwnerID, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt); err != nil {
			return nil, apperrors.NewInternalError("failed to scan shared note", err)
		}
		notes = append(notes, &note)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.NewInternalError("shared notes rows iteration error", err)
	}
	return notes, nil
}

func (r *assetRepositoryImpl) GetFolderShareLevel(folderID, userID int64) (string, error) {
	var shareLevel string
	err := r.db.QueryRow("SELECT permission_level FROM folder_shares WHERE folder_id = ? AND shared_with_user_id = ?", folderID, userID).
		Scan(&shareLevel)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No share level found
		}
		return "", apperrors.NewInternalError("failed to query folder share level", err)
	}
	return shareLevel, nil
}

func (r *assetRepositoryImpl) GetNoteShareLevel(noteID, userID int64) (string, error) {
	var shareLevel string
	err := r.db.QueryRow("SELECT permission_level FROM note_shares WHERE note_id = ? AND shared_with_user_id = ?", noteID, userID).
		Scan(&shareLevel)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No share level found
		}
		return "", apperrors.NewInternalError("failed to query note share level", err)
	}
	return shareLevel, nil
}

func (r *assetRepositoryImpl) IsManagerOfOwner(requesterID, ownerID int64) (bool, error) {
	var count int
	query := `
		SELECT COUNT(1)
		FROM team_members req_tm
		JOIN team_members owner_tm ON req_tm.team_id = owner_tm.team_id
		WHERE req_tm.user_id = ? 
		  AND req_tm.team_role IN ('manager', 'main_manager')
		  AND owner_tm.user_id = ?
	`
	err := r.db.QueryRow(query, requesterID, ownerID).Scan(&count)
	if err != nil {
		return false, apperrors.NewInternalError("failed to query manager relationship", err)
	}
	return count > 0, nil
}

func (r *assetRepositoryImpl) UpdateNote(note *models.Note) error {
	_, err := r.db.Exec("UPDATE notes SET title = ?, content = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", note.Title, note.Content, note.ID)
	if err != nil {
		return apperrors.NewInternalError("failed to update note", err)
	}
	return nil
}

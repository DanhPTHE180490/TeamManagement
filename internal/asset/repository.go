package asset

import (
	"database/sql"
	"team-management/internal/models"
	"time"
)

type AssetRepository interface {
	getNoteByID(id int64) (*models.Note, error)
	GetFolderShareLevel(folderID, userID int64) (string, error)
	GetNoteShareLevel(noteID, userID int64) (string, error)
	IsManagerOfOwner(requesterID, ownerID int64) (bool, error)
	updateNote(note *models.Note) error
	createNote(note *models.Note) (*models.Note, error)
	getUserNotes(userID int64) ([]*models.Note, error)
	deleteNote(noteID int64) error
	shareNote(noteShare *models.NoteShare) error
	removeNoteShare(noteID, userID int64) error
	getNoteShares(noteID int64) ([]*models.NoteShare, error)
	createFolder(folder *models.Folder) (*models.Folder, error)
	getUserFolders(userID int64) ([]*models.Folder, error)
	getFolderByID(folderID int64) (*models.Folder, error)
	deleteFolder(folderID int64) error
	shareFolder(folderShare *models.FolderShare) error
	removeFolderShare(folderID, userID int64) error
	getSharedNotes(userID int64) ([]*models.Note, error)
	getFolderShares(folderID int64) ([]*models.FolderShare, error)
	GetManagersOfOwner(ownerID int64) ([]int64, error)
}

type assetRepositoryImpl struct {
	db *sql.DB
}

func NewAssetRepository(db *sql.DB) AssetRepository {
	return &assetRepositoryImpl{db: db}
}

func (r *assetRepositoryImpl) getNoteByID(id int64) (*models.Note, error) {
	var note models.Note
	err := r.db.QueryRow("SELECT id, folder_id, owner_id, title, content, created_at, updated_at FROM notes WHERE id = ?", id).
		Scan(&note.ID, &note.FolderID, &note.OwnerID, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &note, nil
}

func (r *assetRepositoryImpl) createNote(note *models.Note) (*models.Note, error) {
	now := time.Now()
	result, err := r.db.Exec(
		"INSERT INTO notes (folder_id, owner_id, title, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		note.FolderID, note.OwnerID, note.Title, note.Content, now, now,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	note.ID = id
	note.CreatedAt = now
	note.UpdatedAt = now
	return note, nil
}

func (r *assetRepositoryImpl) getUserNotes(userID int64) ([]*models.Note, error) {
	rows, err := r.db.Query("SELECT id, folder_id, owner_id, title, content, created_at, updated_at FROM notes WHERE owner_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []*models.Note
	for rows.Next() {
		var note models.Note
		if err := rows.Scan(&note.ID, &note.FolderID, &note.OwnerID, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, &note)
	}

	return notes, rows.Err()
}

func (r *assetRepositoryImpl) deleteNote(noteID int64) error {
	_, err := r.db.Exec("DELETE FROM notes WHERE id = ?", noteID)
	return err
}

func (r *assetRepositoryImpl) shareNote(noteShare *models.NoteShare) error {
	_, err := r.db.Exec(
		"INSERT INTO note_shares (note_id, shared_with_user_id, permission_level) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE permission_level = ?",
		noteShare.NoteID, noteShare.SharedWithUserID, noteShare.PermissionLevel, noteShare.PermissionLevel,
	)
	return err
}

func (r *assetRepositoryImpl) removeNoteShare(noteID, userID int64) error {
	_, err := r.db.Exec("DELETE FROM note_shares WHERE note_id = ? AND shared_with_user_id = ?", noteID, userID)
	return err
}

func (r *assetRepositoryImpl) getNoteShares(noteID int64) ([]*models.NoteShare, error) {
	rows, err := r.db.Query("SELECT note_id, shared_with_user_id, permission_level FROM note_shares WHERE note_id = ?", noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []*models.NoteShare
	for rows.Next() {
		var share models.NoteShare
		if err := rows.Scan(&share.NoteID, &share.SharedWithUserID, &share.PermissionLevel); err != nil {
			return nil, err
		}
		shares = append(shares, &share)
	}

	return shares, rows.Err()
}

func (r *assetRepositoryImpl) getFolderShares(folderID int64) ([]*models.FolderShare, error) {
	rows, err := r.db.Query("SELECT folder_id, shared_with_user_id, permission_level FROM folder_shares WHERE folder_id = ?", folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []*models.FolderShare
	for rows.Next() {
		var s models.FolderShare
		if err := rows.Scan(&s.FolderID, &s.SharedWithUserID, &s.PermissionLevel); err != nil {
			return nil, err
		}
		shares = append(shares, &s)
	}
	return shares, rows.Err()
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
		return nil, err
	}
	defer rows.Close()

	var managers []int64
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		managers = append(managers, uid)
	}
	return managers, rows.Err()
}

func (r *assetRepositoryImpl) createFolder(folder *models.Folder) (*models.Folder, error) {
	now := time.Now()
	result, err := r.db.Exec(
		"INSERT INTO folders (name, owner_id, created_at, updated_at) VALUES (?, ?, ?, ?)",
		folder.Name, folder.OwnerID, now, now,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	folder.ID = id
	folder.CreatedAt = now
	folder.UpdatedAt = now
	return folder, nil
}

func (r *assetRepositoryImpl) getUserFolders(userID int64) ([]*models.Folder, error) {
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

func (r *assetRepositoryImpl) getFolderByID(folderID int64) (*models.Folder, error) {
	var folder models.Folder
	err := r.db.QueryRow("SELECT id, name, owner_id, created_at, updated_at FROM folders WHERE id = ?", folderID).
		Scan(&folder.ID, &folder.Name, &folder.OwnerID, &folder.CreatedAt, &folder.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &folder, nil
}

func (r *assetRepositoryImpl) deleteFolder(folderID int64) error {
	_, err := r.db.Exec("DELETE FROM folders WHERE id = ?", folderID)
	return err
}

func (r *assetRepositoryImpl) shareFolder(folderShare *models.FolderShare) error {
	_, err := r.db.Exec(
		"INSERT INTO folder_shares (folder_id, shared_with_user_id, permission_level) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE permission_level = ?",
		folderShare.FolderID, folderShare.SharedWithUserID, folderShare.PermissionLevel, folderShare.PermissionLevel,
	)
	return err
}

func (r *assetRepositoryImpl) removeFolderShare(folderID, userID int64) error {
	_, err := r.db.Exec("DELETE FROM folder_shares WHERE folder_id = ? AND shared_with_user_id = ?", folderID, userID)
	return err
}

func (r *assetRepositoryImpl) getSharedNotes(userID int64) ([]*models.Note, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT n.id, n.folder_id, n.owner_id, n.title, n.content, n.created_at, n.updated_at
		FROM notes n
		LEFT JOIN note_shares ns ON n.id = ns.note_id
		LEFT JOIN folder_shares fs ON n.folder_id = fs.folder_id
		WHERE (ns.shared_with_user_id = ? OR fs.shared_with_user_id = ?) AND n.owner_id != ?
		ORDER BY n.created_at DESC
	`, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []*models.Note
	for rows.Next() {
		var note models.Note
		if err := rows.Scan(&note.ID, &note.FolderID, &note.OwnerID, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, &note)
	}

	return notes, rows.Err()
}

func (r *assetRepositoryImpl) GetFolderShareLevel(folderID, userID int64) (string, error) {
	var shareLevel string
	err := r.db.QueryRow("SELECT permission_level FROM folder_shares WHERE folder_id = ? AND shared_with_user_id = ?", folderID, userID).
		Scan(&shareLevel)
	if err != nil {
		if err == sql.ErrNoRows {
			return "Not found.", nil // No share level found
		}
		return "", err
	}
	return shareLevel, nil
}

func (r *assetRepositoryImpl) GetNoteShareLevel(noteID, userID int64) (string, error) {
	var shareLevel string
	err := r.db.QueryRow("SELECT permission_level FROM note_shares WHERE note_id = ? AND shared_with_user_id = ?", noteID, userID).
		Scan(&shareLevel)
	if err != nil {
		if err == sql.ErrNoRows {
			return "Not found.", nil // No share level found
		}
		return "", err
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
		return false, err
	}
	return count > 0, nil
}

func (r *assetRepositoryImpl) updateNote(note *models.Note) error {
	_, err := r.db.Exec("UPDATE notes SET title = ?, content = ?, updated_at = ? WHERE id = ?", note.Title, note.Content, note.UpdatedAt, note.ID)
	return err
}

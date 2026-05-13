package asset

import (
	"database/sql"
	"team-management/internal/models"
)

type AssetRepository interface {
	getNoteByID(id int64) (*models.Note, error)
	GetFolderShareLevel(folderID, userID int64) (string, error)
	GetNoteShareLevel(noteID, userID int64) (string, error)
	IsManagerOfOwner(requesterID, ownerID int64) (bool, error)
	updateNote(note *models.Note) error
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

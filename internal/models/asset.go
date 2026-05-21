package models

import "time"

type Folder struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	OwnerID   int64     `json:"owner_id" db:"owner_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	Notes     []Note    `json:"notes,omitempty" db:"-"`
}

type Note struct {
	ID        int64     `json:"id" db:"id"`
	FolderID  int64     `json:"folder_id" db:"folder_id"`
	OwnerID   int64     `json:"owner_id" db:"owner_id"`
	Title     string    `json:"title" db:"title"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type FolderShare struct {
	FolderID         int64  `json:"folder_id" db:"folder_id"`
	SharedWithUserID int64  `json:"shared_with_user_id" db:"shared_with_user_id"`
	PermissionLevel  string `json:"permission_level" db:"permission_level"` // "read" or "write"
}

type NoteShare struct {
	NoteID           int64  `json:"note_id" db:"note_id"`
	SharedWithUserID int64  `json:"shared_with_user_id" db:"shared_with_user_id"`
	PermissionLevel  string `json:"permission_level" db:"permission_level"` // "read" or "write"
}

type NoteCreateRequest struct {
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
	FolderID *int64 `json:"folder_id"`
}

type NoteShareRequest struct {
	UserID     int64  `json:"user_id" binding:"required"`
	Permission string `json:"permission" binding:"required"`
}

type FolderCreateRequest struct {
	Name string `json:"name" binding:"required"`
}

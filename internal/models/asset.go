package models

import "time"

type Folder struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	OwnerID   int64     `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Note struct {
	ID        int64     `json:"id"`
	FolderID  int64     `json:"folder_id"`
	OwnerID   int64     `json:"owner_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FolderShare struct {
	FolderID         int64  `json:"folder_id"`
	SharedWithUserID int64  `json:"shared_with_user_id"`
	PermissionLevel  string `json:"permission_level"` // "read" or "write"
}

type NoteShare struct {
	NoteID           int64  `json:"note_id"`
	SharedWithUserID int64  `json:"shared_with_user_id"`
	PermissionLevel  string `json:"permission_level"` // "read" or "write"
}

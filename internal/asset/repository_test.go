package asset

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"team-management/internal/models"
	apperrors "team-management/internal/utils"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAssetRepository_GetNoteByID(t *testing.T) {
	tests := []struct {
		name     string
		noteID   int64
		setupDB  func(sqlmock.Sqlmock)
		wantNote *models.Note
		wantErr  bool
		wantType apperrors.ErrorType
	}{
		{
			name:   "success",
			noteID: 1,
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, folder_id, owner_id, title, content, created_at, updated_at FROM notes WHERE id = \\?").
					WithArgs(int64(1)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "folder_id", "owner_id", "title", "content", "created_at", "updated_at"}).
						AddRow(1, 5, 10, "Test Note", "Content", time.Now(), time.Now()))
			},
			wantNote: &models.Note{ID: 1, FolderID: 5, OwnerID: 10, Title: "Test Note", Content: "Content"},
		},
		{
			name:   "note not found",
			noteID: 999,
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, folder_id, owner_id, title, content, created_at, updated_at FROM notes WHERE id = \\?").
					WithArgs(int64(999)).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr:  true,
			wantType: apperrors.ErrTypeNotFound,
		},
		{
			name:   "null content",
			noteID: 2,
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, folder_id, owner_id, title, content, created_at, updated_at FROM notes WHERE id = \\?").
					WithArgs(int64(2)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "folder_id", "owner_id", "title", "content", "created_at", "updated_at"}).
						AddRow(2, 0, 10, "Title Only", nil, time.Now(), time.Now()))
			},
			wantNote: &models.Note{ID: 2, FolderID: 0, OwnerID: 10, Title: "Title Only", Content: ""},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock db: %v", err)
			}
			defer db.Close()

			tc.setupDB(mock)
			repo := &assetRepositoryImpl{db: db}

			note, err := repo.GetNoteByID(context.Background(), tc.noteID)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !apperrors.IsErrorType(err, tc.wantType) {
					t.Fatalf("expected error type %s, got %v", tc.wantType, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if note.ID != tc.wantNote.ID || note.OwnerID != tc.wantNote.OwnerID {
				t.Fatalf("expected note %v, got %v", tc.wantNote, note)
			}
		})
	}
}

func TestAssetRepository_CreateNote(t *testing.T) {
	tests := []struct {
		name    string
		note    *models.Note
		setupDB func(sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name: "success",
			note: &models.Note{OwnerID: 10, FolderID: 5, Title: "New Note", Content: "New Content"},
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO notes").
					WithArgs(int64(5), int64(10), "New Note", "New Content", sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(42, 1))
			},
		},
		{
			name: "database error",
			note: &models.Note{OwnerID: 10, Title: "Note", Content: "Content"},
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO notes").
					WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock db: %v", err)
			}
			defer db.Close()

			tc.setupDB(mock)
			repo := &assetRepositoryImpl{db: db}

			result, err := repo.CreateNote(context.Background(), tc.note)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected note, got nil")
			}
			if result.ID != 42 {
				t.Fatalf("expected ID 42, got %d", result.ID)
			}
		})
	}
}

func TestAssetRepository_GetUserNotes(t *testing.T) {
	tests := []struct {
		name    string
		userID  int64
		setupDB func(sqlmock.Sqlmock)
		wantLen int
		wantErr bool
	}{
		{
			name:   "success with notes",
			userID: 10,
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, folder_id, owner_id, title, content, created_at, updated_at FROM notes WHERE owner_id = \\?").
					WithArgs(int64(10)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "folder_id", "owner_id", "title", "content", "created_at", "updated_at"}).
						AddRow(1, 5, 10, "Note 1", "Content 1", time.Now(), time.Now()).
						AddRow(2, 5, 10, "Note 2", "Content 2", time.Now(), time.Now()))
			},
			wantLen: 2,
		},
		{
			name:   "empty notes",
			userID: 10,
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, folder_id, owner_id, title, content, created_at, updated_at FROM notes WHERE owner_id = \\?").
					WithArgs(int64(10)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "folder_id", "owner_id", "title", "content", "created_at", "updated_at"}))
			},
			wantLen: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock db: %v", err)
			}
			defer db.Close()

			tc.setupDB(mock)
			repo := &assetRepositoryImpl{db: db}

			notes, err := repo.GetUserNotes(context.Background(), tc.userID)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(notes) != tc.wantLen {
				t.Fatalf("expected %d notes, got %d", tc.wantLen, len(notes))
			}
		})
	}
}

func TestAssetRepository_DeleteNote(t *testing.T) {
	tests := []struct {
		name    string
		noteID  int64
		setupDB func(sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name:   "success",
			noteID: 1,
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("DELETE FROM notes WHERE id = \\?").
					WithArgs(int64(1)).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name:   "database error",
			noteID: 1,
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("DELETE FROM notes WHERE id = \\?").
					WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock db: %v", err)
			}
			defer db.Close()

			tc.setupDB(mock)
			repo := &assetRepositoryImpl{db: db}

			err = repo.DeleteNote(context.Background(), tc.noteID)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAssetRepository_ShareNote(t *testing.T) {
	tests := []struct {
		name    string
		share   *models.NoteShare
		setupDB func(sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name:  "share with read permission",
			share: &models.NoteShare{NoteID: 1, SharedWithUserID: 20, PermissionLevel: "read"},
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO note_shares").
					WithArgs(int64(1), int64(20), "read", "read").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name:  "database error",
			share: &models.NoteShare{NoteID: 1, SharedWithUserID: 20, PermissionLevel: "write"},
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO note_shares").
					WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock db: %v", err)
			}
			defer db.Close()

			tc.setupDB(mock)
			repo := &assetRepositoryImpl{db: db}

			err = repo.ShareNote(context.Background(), tc.share)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAssetRepository_GetNoteShares(t *testing.T) {
	tests := []struct {
		name    string
		noteID  int64
		setupDB func(sqlmock.Sqlmock)
		wantLen int
		wantErr bool
	}{
		{
			name:   "success with shares",
			noteID: 1,
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT note_id, shared_with_user_id, permission_level FROM note_shares WHERE note_id = \\?").
					WithArgs(int64(1)).
					WillReturnRows(sqlmock.NewRows([]string{"note_id", "shared_with_user_id", "permission_level"}).
						AddRow(1, 20, "read").
						AddRow(1, 30, "write"))
			},
			wantLen: 2,
		},
		{
			name:   "no shares",
			noteID: 1,
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT note_id, shared_with_user_id, permission_level FROM note_shares WHERE note_id = \\?").
					WithArgs(int64(1)).
					WillReturnRows(sqlmock.NewRows([]string{"note_id", "shared_with_user_id", "permission_level"}))
			},
			wantLen: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock db: %v", err)
			}
			defer db.Close()

			tc.setupDB(mock)
			repo := &assetRepositoryImpl{db: db}

			shares, err := repo.GetNoteShares(context.Background(), tc.noteID)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(shares) != tc.wantLen {
				t.Fatalf("expected %d shares, got %d", tc.wantLen, len(shares))
			}
		})
	}
}

func TestAssetRepository_CreateFolder(t *testing.T) {
	tests := []struct {
		name    string
		folder  *models.Folder
		setupDB func(sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name:   "success",
			folder: &models.Folder{OwnerID: 10, Name: "My Folder"},
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO folders").
					WithArgs("My Folder", int64(10), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(5, 1))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock db: %v", err)
			}
			defer db.Close()

			tc.setupDB(mock)
			repo := &assetRepositoryImpl{db: db}

			result, err := repo.CreateFolder(context.Background(), tc.folder)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.wantErr && result.ID != 5 {
				t.Fatalf("expected ID 5, got %d", result.ID)
			}
		})
	}
}

func TestAssetRepository_GetUserFolders(t *testing.T) {
	tests := []struct {
		name    string
		userID  int64
		setupDB func(sqlmock.Sqlmock)
		wantLen int
		wantErr bool
	}{
		{
			name:   "success with folders",
			userID: 10,
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, name, owner_id, created_at, updated_at FROM folders WHERE owner_id = \\?").
					WithArgs(int64(10)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "name", "owner_id", "created_at", "updated_at"}).
						AddRow(1, "Folder 1", 10, time.Now(), time.Now()).
						AddRow(2, "Folder 2", 10, time.Now(), time.Now()))
			},
			wantLen: 2,
		},
		{
			name:   "no folders",
			userID: 10,
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, name, owner_id, created_at, updated_at FROM folders WHERE owner_id = \\?").
					WithArgs(int64(10)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "name", "owner_id", "created_at", "updated_at"}))
			},
			wantLen: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock db: %v", err)
			}
			defer db.Close()

			tc.setupDB(mock)
			repo := &assetRepositoryImpl{db: db}

			folders, err := repo.GetUserFolders(context.Background(), tc.userID)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(folders) != tc.wantLen {
				t.Fatalf("expected %d folders, got %d", tc.wantLen, len(folders))
			}
		})
	}
}

func TestAssetRepository_GetFolderByID(t *testing.T) {
	tests := []struct {
		name     string
		folderID int64
		setupDB  func(sqlmock.Sqlmock)
		wantName string
		wantErr  bool
		wantType apperrors.ErrorType
	}{
		{
			name:     "success",
			folderID: 1,
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, name, owner_id, created_at, updated_at FROM folders WHERE id = \\?").
					WithArgs(int64(1)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "name", "owner_id", "created_at", "updated_at"}).
						AddRow(1, "My Folder", 10, time.Now(), time.Now()))
			},
			wantName: "My Folder",
		},
		{
			name:     "folder not found",
			folderID: 999,
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, name, owner_id, created_at, updated_at FROM folders WHERE id = \\?").
					WithArgs(int64(999)).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr:  true,
			wantType: apperrors.ErrTypeNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock db: %v", err)
			}
			defer db.Close()

			tc.setupDB(mock)
			repo := &assetRepositoryImpl{db: db}

			folder, err := repo.GetFolderByID(context.Background(), tc.folderID)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !apperrors.IsErrorType(err, tc.wantType) {
					t.Fatalf("expected error type %s, got %v", tc.wantType, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if folder.Name != tc.wantName {
				t.Fatalf("expected folder name %q, got %q", tc.wantName, folder.Name)
			}
		})
	}
}

func TestAssetRepository_IsManagerOfOwner(t *testing.T) {
	tests := []struct {
		name        string
		requesterID int64
		ownerID     int64
		setupDB     func(sqlmock.Sqlmock)
		wantResult  bool
		wantErr     bool
	}{
		{
			name:        "requester is manager",
			requesterID: 20,
			ownerID:     10,
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT COUNT\\(1\\)").
					WithArgs(int64(20), int64(10)).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			},
			wantResult: true,
		},
		{
			name:        "requester is not manager",
			requesterID: 20,
			ownerID:     10,
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT COUNT\\(1\\)").
					WithArgs(int64(20), int64(10)).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			},
			wantResult: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock db: %v", err)
			}
			defer db.Close()

			tc.setupDB(mock)
			repo := &assetRepositoryImpl{db: db}

			result, err := repo.IsManagerOfOwner(context.Background(), tc.requesterID, tc.ownerID)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tc.wantResult {
				t.Fatalf("expected %v, got %v", tc.wantResult, result)
			}
		})
	}
}

func TestAssetRepository_GetShareLevel(t *testing.T) {
	tests := []struct {
		name      string
		setupDB   func(sqlmock.Sqlmock)
		wantLevel string
		wantErr   bool
	}{
		{
			name: "share exists",
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT permission_level FROM folder_shares WHERE folder_id = \\? AND shared_with_user_id = \\?").
					WithArgs(int64(1), int64(20)).
					WillReturnRows(sqlmock.NewRows([]string{"permission_level"}).AddRow("write"))
			},
			wantLevel: "write",
		},
		{
			name: "no share found",
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT permission_level FROM folder_shares WHERE folder_id = \\? AND shared_with_user_id = \\?").
					WithArgs(int64(1), int64(20)).
					WillReturnError(sql.ErrNoRows)
			},
			wantLevel: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock db: %v", err)
			}
			defer db.Close()

			tc.setupDB(mock)
			repo := &assetRepositoryImpl{db: db}

			level, err := repo.GetFolderShareLevel(context.Background(), 1, 20)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if level != tc.wantLevel {
				t.Fatalf("expected level %q, got %q", tc.wantLevel, level)
			}
		})
	}
}

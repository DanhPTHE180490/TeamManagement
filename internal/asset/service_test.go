package asset

import (
	"testing"
	"time"

	apperrors "team-management/internal/errors"
	"team-management/internal/models"
)

type mockAssetRepo struct {
	getNoteByIDFn         func(int64) (*models.Note, error)
	getFolderShareLevelFn func(int64, int64) (string, error)
	getNoteShareLevelFn   func(int64, int64) (string, error)
	isManagerOfOwnerFn    func(int64, int64) (bool, error)
	updateNoteFn          func(*models.Note) error
	createNoteFn          func(*models.Note) (*models.Note, error)
	getUserNotesFn        func(int64) ([]*models.Note, error)
	deleteNoteFn          func(int64) error
	shareNoteFn           func(*models.NoteShare) error
	removeNoteShareFn     func(int64, int64) error
	getNoteSharesFn       func(int64) ([]*models.NoteShare, error)
	createFolderFn        func(*models.Folder) (*models.Folder, error)
	getUserFoldersFn      func(int64) ([]*models.Folder, error)
	getFolderByIDFn       func(int64) (*models.Folder, error)
	deleteFolderFn        func(int64) error
	shareFolderFn         func(*models.FolderShare) error
	removeFolderShareFn   func(int64, int64) error
	getSharedNotesFn      func(int64) ([]*models.Note, error)
	getFolderSharesFn     func(int64) ([]*models.FolderShare, error)
	getManagersOfOwnerFn  func(int64) ([]int64, error)
}

func (m *mockAssetRepo) GetNoteByID(id int64) (*models.Note, error) {
	if m.getNoteByIDFn != nil {
		return m.getNoteByIDFn(id)
	}
	return &models.Note{ID: id}, nil
}

func (m *mockAssetRepo) GetFolderShareLevel(folderID, userID int64) (string, error) {
	if m.getFolderShareLevelFn != nil {
		return m.getFolderShareLevelFn(folderID, userID)
	}
	return "", nil
}

func (m *mockAssetRepo) GetNoteShareLevel(noteID, userID int64) (string, error) {
	if m.getNoteShareLevelFn != nil {
		return m.getNoteShareLevelFn(noteID, userID)
	}
	return "", nil
}

func (m *mockAssetRepo) IsManagerOfOwner(requesterID, ownerID int64) (bool, error) {
	if m.isManagerOfOwnerFn != nil {
		return m.isManagerOfOwnerFn(requesterID, ownerID)
	}
	return false, nil
}

func (m *mockAssetRepo) UpdateNote(note *models.Note) error {
	if m.updateNoteFn != nil {
		return m.updateNoteFn(note)
	}
	return nil
}

func (m *mockAssetRepo) CreateNote(note *models.Note) (*models.Note, error) {
	if m.createNoteFn != nil {
		return m.createNoteFn(note)
	}
	note.ID = 1
	return note, nil
}

func (m *mockAssetRepo) GetUserNotes(userID int64) ([]*models.Note, error) {
	if m.getUserNotesFn != nil {
		return m.getUserNotesFn(userID)
	}
	return []*models.Note{}, nil
}

func (m *mockAssetRepo) DeleteNote(noteID int64) error {
	if m.deleteNoteFn != nil {
		return m.deleteNoteFn(noteID)
	}
	return nil
}

func (m *mockAssetRepo) ShareNote(noteShare *models.NoteShare) error {
	if m.shareNoteFn != nil {
		return m.shareNoteFn(noteShare)
	}
	return nil
}

func (m *mockAssetRepo) RemoveNoteShare(noteID, userID int64) error {
	if m.removeNoteShareFn != nil {
		return m.removeNoteShareFn(noteID, userID)
	}
	return nil
}

func (m *mockAssetRepo) GetNoteShares(noteID int64) ([]*models.NoteShare, error) {
	if m.getNoteSharesFn != nil {
		return m.getNoteSharesFn(noteID)
	}
	return []*models.NoteShare{}, nil
}

func (m *mockAssetRepo) CreateFolder(folder *models.Folder) (*models.Folder, error) {
	if m.createFolderFn != nil {
		return m.createFolderFn(folder)
	}
	folder.ID = 1
	return folder, nil
}

func (m *mockAssetRepo) GetUserFolders(userID int64) ([]*models.Folder, error) {
	if m.getUserFoldersFn != nil {
		return m.getUserFoldersFn(userID)
	}
	return []*models.Folder{}, nil
}

func (m *mockAssetRepo) GetFolderByID(folderID int64) (*models.Folder, error) {
	if m.getFolderByIDFn != nil {
		return m.getFolderByIDFn(folderID)
	}
	return &models.Folder{ID: folderID}, nil
}

func (m *mockAssetRepo) DeleteFolder(folderID int64) error {
	if m.deleteFolderFn != nil {
		return m.deleteFolderFn(folderID)
	}
	return nil
}

func (m *mockAssetRepo) ShareFolder(folderShare *models.FolderShare) error {
	if m.shareFolderFn != nil {
		return m.shareFolderFn(folderShare)
	}
	return nil
}

func (m *mockAssetRepo) RemoveFolderShare(folderID, userID int64) error {
	if m.removeFolderShareFn != nil {
		return m.removeFolderShareFn(folderID, userID)
	}
	return nil
}

func (m *mockAssetRepo) GetSharedNotes(userID int64) ([]*models.Note, error) {
	if m.getSharedNotesFn != nil {
		return m.getSharedNotesFn(userID)
	}
	return []*models.Note{}, nil
}

func (m *mockAssetRepo) GetFolderShares(folderID int64) ([]*models.FolderShare, error) {
	if m.getFolderSharesFn != nil {
		return m.getFolderSharesFn(folderID)
	}
	return []*models.FolderShare{}, nil
}

func (m *mockAssetRepo) GetManagersOfOwner(ownerID int64) ([]int64, error) {
	if m.getManagersOfOwnerFn != nil {
		return m.getManagersOfOwnerFn(ownerID)
	}
	return []int64{}, nil
}

// Tests

func TestAssetService_CreateNote(t *testing.T) {
	tests := []struct {
		name        string
		requesterID int64
		title       string
		content     string
		folderID    *int64
		wantErr     bool
	}{
		{
			name:        "create note without folder",
			requesterID: 10,
			title:       "My Note",
			content:     "Note content",
			folderID:    nil,
			wantErr:     false,
		},
		{
			name:        "create note with folder",
			requesterID: 10,
			title:       "Organized Note",
			content:     "Organized content",
			folderID:    int64Ptr(5),
			wantErr:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockAssetRepo{}
			service := NewAssetService(repo)

			result, err := service.CreateNote(tc.requesterID, tc.title, tc.content, tc.folderID)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.wantErr && result.OwnerID != tc.requesterID {
				t.Fatalf("expected owner %d, got %d", tc.requesterID, result.OwnerID)
			}
		})
	}
}

func TestAssetService_GetNoteByID(t *testing.T) {
	testNote := &models.Note{
		ID:        1,
		OwnerID:   10,
		Title:     "Test",
		Content:   "Content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tests := []struct {
		name        string
		requesterID int64
		noteID      int64
		setupRepo   func(*mockAssetRepo)
		wantErr     bool
		wantType    apperrors.ErrorType
	}{
		{
			name:        "owner can view own note",
			requesterID: 10,
			noteID:      1,
			setupRepo: func(repo *mockAssetRepo) {
				repo.getNoteByIDFn = func(id int64) (*models.Note, error) {
					return testNote, nil
				}
			},
			wantErr: false,
		},
		{
			name:        "non-owner cannot view without permission",
			requesterID: 20,
			noteID:      1,
			setupRepo: func(repo *mockAssetRepo) {
				repo.getNoteByIDFn = func(id int64) (*models.Note, error) {
					return testNote, nil
				}
				repo.getFolderShareLevelFn = func(folderID, userID int64) (string, error) {
					return "", nil
				}
				repo.getNoteShareLevelFn = func(noteID, userID int64) (string, error) {
					return "", nil
				}
				repo.isManagerOfOwnerFn = func(requesterID, ownerID int64) (bool, error) {
					return false, nil
				}
			},
			wantErr:  true,
			wantType: apperrors.ErrTypeForbidden,
		},
		{
			name:        "can view with note share read permission",
			requesterID: 20,
			noteID:      1,
			setupRepo: func(repo *mockAssetRepo) {
				repo.getNoteByIDFn = func(id int64) (*models.Note, error) {
					return testNote, nil
				}
				repo.getFolderShareLevelFn = func(folderID, userID int64) (string, error) {
					return "", nil
				}
				repo.getNoteShareLevelFn = func(noteID, userID int64) (string, error) {
					return "read", nil
				}
			},
			wantErr: false,
		},
		{
			name:        "manager can view owner's note",
			requesterID: 30,
			noteID:      1,
			setupRepo: func(repo *mockAssetRepo) {
				repo.getNoteByIDFn = func(id int64) (*models.Note, error) {
					return testNote, nil
				}
				repo.getFolderShareLevelFn = func(folderID, userID int64) (string, error) {
					return "", nil
				}
				repo.getNoteShareLevelFn = func(noteID, userID int64) (string, error) {
					return "", nil
				}
				repo.isManagerOfOwnerFn = func(requesterID, ownerID int64) (bool, error) {
					return true, nil
				}
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockAssetRepo{}
			if tc.setupRepo != nil {
				tc.setupRepo(repo)
			}
			service := NewAssetService(repo)

			note, err := service.GetNoteByID(tc.requesterID, tc.noteID)
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
			if note == nil {
				t.Fatal("expected note, got nil")
			}
		})
	}
}

func TestAssetService_UpdateNote(t *testing.T) {
	testNote := &models.Note{
		ID:      1,
		OwnerID: 10,
		Title:   "Old Title",
		Content: "Old Content",
	}

	tests := []struct {
		name        string
		requesterID int64
		noteID      int64
		title       string
		content     string
		folderID    *int64
		setupRepo   func(*mockAssetRepo)
		wantErr     bool
		wantType    apperrors.ErrorType
	}{
		{
			name:        "owner can update own note",
			requesterID: 10,
			noteID:      1,
			title:       "New Title",
			content:     "New Content",
			setupRepo: func(repo *mockAssetRepo) {
				repo.getNoteByIDFn = func(id int64) (*models.Note, error) {
					return testNote, nil
				}
				repo.updateNoteFn = func(n *models.Note) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:        "non-owner cannot update",
			requesterID: 20,
			noteID:      1,
			title:       "Hacked",
			content:     "Hacked",
			setupRepo: func(repo *mockAssetRepo) {
				repo.getNoteByIDFn = func(id int64) (*models.Note, error) {
					return testNote, nil
				}
				repo.getFolderShareLevelFn = func(folderID, userID int64) (string, error) {
					return "", nil
				}
				repo.getNoteShareLevelFn = func(noteID, userID int64) (string, error) {
					return "read", nil
				}
			},
			wantErr:  true,
			wantType: apperrors.ErrTypeForbidden,
		},
		{
			name:        "user with write share can update",
			requesterID: 20,
			noteID:      1,
			title:       "Updated",
			content:     "Updated content",
			setupRepo: func(repo *mockAssetRepo) {
				repo.getNoteByIDFn = func(id int64) (*models.Note, error) {
					return testNote, nil
				}
				repo.getNoteShareLevelFn = func(noteID, userID int64) (string, error) {
					return "write", nil
				}
				repo.updateNoteFn = func(n *models.Note) error {
					return nil
				}
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockAssetRepo{}
			if tc.setupRepo != nil {
				tc.setupRepo(repo)
			}
			service := NewAssetService(repo)

			note, err := service.UpdateNote(tc.requesterID, tc.noteID, tc.title, tc.content, tc.folderID)
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
			if note == nil {
				t.Fatal("expected note, got nil")
			}
		})
	}
}

func TestAssetService_DeleteNote(t *testing.T) {
	testNote := &models.Note{ID: 1, OwnerID: 10}

	tests := []struct {
		name        string
		requesterID int64
		noteID      int64
		setupRepo   func(*mockAssetRepo)
		wantErr     bool
		wantType    apperrors.ErrorType
	}{
		{
			name:        "owner can delete",
			requesterID: 10,
			noteID:      1,
			setupRepo: func(repo *mockAssetRepo) {
				repo.getNoteByIDFn = func(id int64) (*models.Note, error) {
					return testNote, nil
				}
				repo.deleteNoteFn = func(id int64) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:        "non-owner cannot delete",
			requesterID: 20,
			noteID:      1,
			setupRepo: func(repo *mockAssetRepo) {
				repo.getNoteByIDFn = func(id int64) (*models.Note, error) {
					return testNote, nil
				}
			},
			wantErr:  true,
			wantType: apperrors.ErrTypeForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockAssetRepo{}
			if tc.setupRepo != nil {
				tc.setupRepo(repo)
			}
			service := NewAssetService(repo)

			err := service.DeleteNote(tc.requesterID, tc.noteID)
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
		})
	}
}

func TestAssetService_ShareNote(t *testing.T) {
	testNote := &models.Note{ID: 1, OwnerID: 10}

	tests := []struct {
		name             string
		requesterID      int64
		noteID           int64
		sharedWithUserID int64
		permission       string
		setupRepo        func(*mockAssetRepo)
		wantErr          bool
		wantType         apperrors.ErrorType
	}{
		{
			name:             "owner can share with read",
			requesterID:      10,
			noteID:           1,
			sharedWithUserID: 20,
			permission:       "read",
			setupRepo: func(repo *mockAssetRepo) {
				repo.getNoteByIDFn = func(id int64) (*models.Note, error) {
					return testNote, nil
				}
				repo.shareNoteFn = func(ns *models.NoteShare) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:             "owner can share with write",
			requesterID:      10,
			noteID:           1,
			sharedWithUserID: 20,
			permission:       "write",
			setupRepo: func(repo *mockAssetRepo) {
				repo.getNoteByIDFn = func(id int64) (*models.Note, error) {
					return testNote, nil
				}
				repo.shareNoteFn = func(ns *models.NoteShare) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:             "invalid permission level",
			requesterID:      10,
			noteID:           1,
			sharedWithUserID: 20,
			permission:       "admin",
			setupRepo: func(repo *mockAssetRepo) {
				repo.getNoteByIDFn = func(id int64) (*models.Note, error) {
					return testNote, nil
				}
			},
			wantErr:  true,
			wantType: apperrors.ErrTypeValidation,
		},
		{
			name:             "non-owner cannot share",
			requesterID:      20,
			noteID:           1,
			sharedWithUserID: 30,
			permission:       "read",
			setupRepo: func(repo *mockAssetRepo) {
				repo.getNoteByIDFn = func(id int64) (*models.Note, error) {
					return testNote, nil
				}
			},
			wantErr:  true,
			wantType: apperrors.ErrTypeForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockAssetRepo{}
			if tc.setupRepo != nil {
				tc.setupRepo(repo)
			}
			service := NewAssetService(repo)

			err := service.ShareNote(tc.requesterID, tc.noteID, tc.sharedWithUserID, tc.permission)
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
		})
	}
}

func TestAssetService_GetUserNotes(t *testing.T) {
	userNotes := []*models.Note{
		{ID: 1, OwnerID: 10, Title: "Note 1"},
		{ID: 2, OwnerID: 10, Title: "Note 2"},
	}

	repo := &mockAssetRepo{
		getUserNotesFn: func(userID int64) ([]*models.Note, error) {
			return userNotes, nil
		},
	}

	service := NewAssetService(repo)
	notes, err := service.GetUserNotes(10)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
}

func TestAssetService_CreateFolder(t *testing.T) {
	tests := []struct {
		name        string
		requesterID int64
		folderName  string
		setupRepo   func(*mockAssetRepo)
		wantErr     bool
	}{
		{
			name:        "create folder success",
			requesterID: 10,
			folderName:  "My Folder",
			setupRepo: func(repo *mockAssetRepo) {
				repo.createFolderFn = func(f *models.Folder) (*models.Folder, error) {
					f.ID = 5
					return f, nil
				}
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockAssetRepo{}
			if tc.setupRepo != nil {
				tc.setupRepo(repo)
			}
			service := NewAssetService(repo)

			folder, err := service.CreateFolder(tc.requesterID, tc.folderName)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.wantErr && folder.OwnerID != tc.requesterID {
				t.Fatalf("expected owner %d, got %d", tc.requesterID, folder.OwnerID)
			}
		})
	}
}

func TestAssetService_DeleteFolder(t *testing.T) {
	testFolder := &models.Folder{ID: 1, OwnerID: 10, Name: "Folder"}

	tests := []struct {
		name        string
		requesterID int64
		folderID    int64
		setupRepo   func(*mockAssetRepo)
		wantErr     bool
		wantType    apperrors.ErrorType
	}{
		{
			name:        "owner can delete",
			requesterID: 10,
			folderID:    1,
			setupRepo: func(repo *mockAssetRepo) {
				repo.getFolderByIDFn = func(id int64) (*models.Folder, error) {
					return testFolder, nil
				}
				repo.deleteFolderFn = func(id int64) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:        "non-owner cannot delete",
			requesterID: 20,
			folderID:    1,
			setupRepo: func(repo *mockAssetRepo) {
				repo.getFolderByIDFn = func(id int64) (*models.Folder, error) {
					return testFolder, nil
				}
			},
			wantErr:  true,
			wantType: apperrors.ErrTypeForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockAssetRepo{}
			if tc.setupRepo != nil {
				tc.setupRepo(repo)
			}
			service := NewAssetService(repo)

			err := service.DeleteFolder(tc.requesterID, tc.folderID)
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
		})
	}
}

func TestAssetService_RemoveNoteShare(t *testing.T) {
	testNote := &models.Note{ID: 1, OwnerID: 10}

	tests := []struct {
		name             string
		requesterID      int64
		noteID           int64
		sharedWithUserID int64
		setupRepo        func(*mockAssetRepo)
		wantErr          bool
		wantType         apperrors.ErrorType
	}{
		{
			name:             "owner can remove share",
			requesterID:      10,
			noteID:           1,
			sharedWithUserID: 20,
			setupRepo: func(repo *mockAssetRepo) {
				repo.getNoteByIDFn = func(id int64) (*models.Note, error) {
					return testNote, nil
				}
				repo.removeNoteShareFn = func(noteID, userID int64) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:             "non-owner cannot remove share",
			requesterID:      20,
			noteID:           1,
			sharedWithUserID: 30,
			setupRepo: func(repo *mockAssetRepo) {
				repo.getNoteByIDFn = func(id int64) (*models.Note, error) {
					return testNote, nil
				}
			},
			wantErr:  true,
			wantType: apperrors.ErrTypeForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockAssetRepo{}
			if tc.setupRepo != nil {
				tc.setupRepo(repo)
			}
			service := NewAssetService(repo)

			err := service.RemoveNoteShare(tc.requesterID, tc.noteID, tc.sharedWithUserID)
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
		})
	}
}

func TestAssetService_GetNoteShares(t *testing.T) {
	testNote := &models.Note{ID: 1, OwnerID: 10}
	shares := []*models.NoteShare{
		{NoteID: 1, SharedWithUserID: 20, PermissionLevel: "read"},
		{NoteID: 1, SharedWithUserID: 30, PermissionLevel: "write"},
	}

	tests := []struct {
		name        string
		requesterID int64
		noteID      int64
		setupRepo   func(*mockAssetRepo)
		wantLen     int
		wantErr     bool
		wantType    apperrors.ErrorType
	}{
		{
			name:        "owner can view shares",
			requesterID: 10,
			noteID:      1,
			setupRepo: func(repo *mockAssetRepo) {
				repo.getNoteByIDFn = func(id int64) (*models.Note, error) {
					return testNote, nil
				}
				repo.getNoteSharesFn = func(noteID int64) ([]*models.NoteShare, error) {
					return shares, nil
				}
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name:        "non-owner cannot view shares",
			requesterID: 20,
			noteID:      1,
			setupRepo: func(repo *mockAssetRepo) {
				repo.getNoteByIDFn = func(id int64) (*models.Note, error) {
					return testNote, nil
				}
			},
			wantErr:  true,
			wantType: apperrors.ErrTypeForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockAssetRepo{}
			if tc.setupRepo != nil {
				tc.setupRepo(repo)
			}
			service := NewAssetService(repo)

			shares, err := service.GetNoteShares(tc.requesterID, tc.noteID)
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
			if len(shares) != tc.wantLen {
				t.Fatalf("expected %d shares, got %d", tc.wantLen, len(shares))
			}
		})
	}
}

func TestAssetService_GetNoteAccess(t *testing.T) {
	testNote := &models.Note{
		ID:        1,
		OwnerID:   10,
		FolderID:  5,
		Title:     "Test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tests := []struct {
		name        string
		requesterID int64
		noteID      int64
		setupRepo   func(*mockAssetRepo)
		wantLen     int
		wantErr     bool
		wantType    apperrors.ErrorType
	}{
		{
			name:        "owner can view full access",
			requesterID: 10,
			noteID:      1,
			setupRepo: func(repo *mockAssetRepo) {
				repo.getNoteByIDFn = func(id int64) (*models.Note, error) {
					return testNote, nil
				}
				repo.getNoteSharesFn = func(noteID int64) ([]*models.NoteShare, error) {
					return []*models.NoteShare{
						{NoteID: 1, SharedWithUserID: 20, PermissionLevel: "read"},
					}, nil
				}
				repo.getFolderSharesFn = func(folderID int64) ([]*models.FolderShare, error) {
					return []*models.FolderShare{}, nil
				}
				repo.getManagersOfOwnerFn = func(ownerID int64) ([]int64, error) {
					return []int64{}, nil
				}
			},
			wantLen: 2, // owner + 1 shared user
			wantErr: false,
		},
		{
			name:        "non-owner cannot view access",
			requesterID: 20,
			noteID:      1,
			setupRepo: func(repo *mockAssetRepo) {
				repo.getNoteByIDFn = func(id int64) (*models.Note, error) {
					return testNote, nil
				}
			},
			wantErr:  true,
			wantType: apperrors.ErrTypeForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockAssetRepo{}
			if tc.setupRepo != nil {
				tc.setupRepo(repo)
			}
			service := NewAssetService(repo)

			access, err := service.GetNoteAccess(tc.requesterID, tc.noteID)
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
			if len(access) != tc.wantLen {
				t.Fatalf("expected %d access entries, got %d", tc.wantLen, len(access))
			}
		})
	}
}

// Helper function
func int64Ptr(v int64) *int64 {
	return &v
}

package team

import (
	"context"
	"errors"
	"testing"
	"time"

	customErrors "team-management/internal/errors"
	"team-management/internal/models"
)

type mockTeamRepo struct {
	createTeamFn     func(context.Context, *models.Team, int64, string) (error, int64)
	getTeamByIDFn    func(context.Context, int64) (*models.Team, error)
	getTeamsByUserFn func(context.Context, int64) ([]*models.Team, error)
	userExistsFn     func(context.Context, int64) (bool, error)
	getTeamRoleFn    func(context.Context, int64, int64) (string, error)
	updateTeamFn     func(context.Context, *models.Team) error
	deleteTeamFn     func(context.Context, int64) error
	addMemberFn      func(context.Context, int64, int64, string) error
	removeMemberFn   func(context.Context, int64, int64) error
	updateRoleFn     func(context.Context, int64, int64, string) error

	lastCreatedTeam    *models.Team
	lastCreateUserID   int64
	lastCreateUserRole string
	lastUpdatedTeam    *models.Team
	lastDeletedTeamID  int64
	lastAddTeamID      int64
	lastAddUserID      int64
	lastAddRole        string
	lastRemoveTeamID   int64
	lastRemoveUserID   int64
	lastUpdateRoleTeam int64
	lastUpdateRoleUser int64
	lastUpdateRoleName string
}

func (m *mockTeamRepo) CreateTeam(ctx context.Context, team *models.Team, userID int64, userRole string) (error, int64) {
	m.lastCreatedTeam = team
	m.lastCreateUserID = userID
	m.lastCreateUserRole = userRole
	if m.createTeamFn != nil {
		return m.createTeamFn(ctx, team, userID, userRole)
	}
	return nil, 7
}

func (m *mockTeamRepo) GetTeamByID(ctx context.Context, id int64) (*models.Team, error) {
	if m.getTeamByIDFn != nil {
		return m.getTeamByIDFn(ctx, id)
	}
	return &models.Team{ID: id, Name: "team"}, nil
}

func (m *mockTeamRepo) GetTeamsByUserID(ctx context.Context, userID int64) ([]*models.Team, error) {
	if m.getTeamsByUserFn != nil {
		return m.getTeamsByUserFn(ctx, userID)
	}
	return []*models.Team{{ID: 1, Name: "team"}}, nil
}

func (m *mockTeamRepo) UpdateTeam(ctx context.Context, team *models.Team) error {
	m.lastUpdatedTeam = team
	if m.updateTeamFn != nil {
		return m.updateTeamFn(ctx, team)
	}
	return nil
}

func (m *mockTeamRepo) DeleteTeam(ctx context.Context, id int64) error {
	m.lastDeletedTeamID = id
	if m.deleteTeamFn != nil {
		return m.deleteTeamFn(ctx, id)
	}
	return nil
}

func (m *mockTeamRepo) UserExists(ctx context.Context, userID int64) (bool, error) {
	if m.userExistsFn != nil {
		return m.userExistsFn(ctx, userID)
	}
	return true, nil
}

func (m *mockTeamRepo) GetTeamRole(ctx context.Context, teamID int64, userID int64) (string, error) {
	if m.getTeamRoleFn != nil {
		return m.getTeamRoleFn(ctx, teamID, userID)
	}
	return "main_manager", nil
}

func (m *mockTeamRepo) AddMember(ctx context.Context, teamID int64, userID int64, role string) error {
	m.lastAddTeamID = teamID
	m.lastAddUserID = userID
	m.lastAddRole = role
	if m.addMemberFn != nil {
		return m.addMemberFn(ctx, teamID, userID, role)
	}
	return nil
}

func (m *mockTeamRepo) RemoveMember(ctx context.Context, teamID int64, userID int64) error {
	m.lastRemoveTeamID = teamID
	m.lastRemoveUserID = userID
	if m.removeMemberFn != nil {
		return m.removeMemberFn(ctx, teamID, userID)
	}
	return nil
}

func (m *mockTeamRepo) UpdateMemberRole(ctx context.Context, teamID int64, userID int64, newRole string) error {
	m.lastUpdateRoleTeam = teamID
	m.lastUpdateRoleUser = userID
	m.lastUpdateRoleName = newRole
	if m.updateRoleFn != nil {
		return m.updateRoleFn(ctx, teamID, userID, newRole)
	}
	return nil
}

func TestTeamService_CreateTeam(t *testing.T) {
	tests := []struct {
		name     string
		teamName string
		role     string
		setup    func(*mockTeamRepo)
		wantErr  bool
		wantType customErrors.ErrorType
	}{
		{
			name:     "success",
			teamName: "Platform",
			role:     "manager",
		},
		{
			name:     "member forbidden",
			teamName: "Platform",
			role:     "member",
			wantErr:  true,
			wantType: customErrors.ErrTypeForbidden,
		},
		{
			name:     "validation error",
			teamName: "",
			role:     "manager",
			wantErr:  true,
			wantType: customErrors.ErrTypeValidation,
		},
		{
			name:     "repo error",
			teamName: "Platform",
			role:     "manager",
			setup: func(repo *mockTeamRepo) {
				repo.createTeamFn = func(ctx context.Context, team *models.Team, creatorID int64, role string) (error, int64) {
					return customErrors.NewInternalError("db failed", errors.New("boom")), 0
				}
			},
			wantErr:  true,
			wantType: customErrors.ErrTypeInternal,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockTeamRepo{}
			if tc.setup != nil {
				tc.setup(repo)
			}
			service := NewTeamService(repo)

			team, err := service.CreateTeam(context.Background(), tc.teamName, 11, tc.role)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !customErrors.IsErrorType(err, tc.wantType) {
					t.Fatalf("expected %s, got %v", tc.wantType, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if team == nil || team.ID != 7 {
				t.Fatalf("unexpected team result: %#v", team)
			}
			if team.Name != tc.teamName {
				t.Fatalf("expected team name %q, got %q", tc.teamName, team.Name)
			}
		})
	}
}

func TestTeamService_UpdateAndDeletePermissions(t *testing.T) {
	repo := &mockTeamRepo{
		getTeamByIDFn: func(_ context.Context, id int64) (*models.Team, error) {
			return &models.Team{ID: id, Name: "Old", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
		getTeamRoleFn: func(_ context.Context, _ int64, userID int64) (string, error) {
			if userID == 1 {
				return "member", nil
			}
			return "main_manager", nil
		},
	}
	service := NewTeamService(repo)

	if _, err := service.UpdateTeam(context.Background(), 5, "New Name", 1); !customErrors.IsErrorType(err, customErrors.ErrTypeForbidden) {
		t.Fatalf("expected forbidden update for member, got %v", err)
	}

	updated, err := service.UpdateTeam(context.Background(), 5, "New Name", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "New Name" {
		t.Fatalf("expected updated name, got %q", updated.Name)
	}
	if repo.lastUpdatedTeam == nil || repo.lastUpdatedTeam.Name != "New Name" {
		t.Fatal("expected repo UpdateTeam to be called with new name")
	}

	if err := service.DeleteTeam(context.Background(), 5, 1); !customErrors.IsErrorType(err, customErrors.ErrTypeForbidden) {
		t.Fatalf("expected forbidden delete for member, got %v", err)
	}

	if err := service.DeleteTeam(context.Background(), 5, 2); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if repo.lastDeletedTeamID != 5 {
		t.Fatalf("expected delete call for team 5, got %d", repo.lastDeletedTeamID)
	}
}

func TestTeamService_MemberManagement(t *testing.T) {
	repo := &mockTeamRepo{
		getTeamByIDFn: func(_ context.Context, id int64) (*models.Team, error) {
			return &models.Team{ID: id, Name: "Team"}, nil
		},
		getTeamRoleFn: func(_ context.Context, _ int64, userID int64) (string, error) {
			switch userID {
			case 1:
				return "member", nil
			case 2:
				return "main_manager", nil
			case 3:
				return "member", nil
			default:
				return "member", nil
			}
		},
		addMemberFn: func(_ context.Context, _ int64, _ int64, _ string) error {
			return customErrors.NewConflictError("user is already a member of this team", errors.New("duplicate"))
		},
		userExistsFn: func(_ context.Context, userID int64) (bool, error) {
			if userID == 999 {
				return false, nil
			}
			return true, nil
		},
	}
	service := NewTeamService(repo)

	if err := service.AddMemberToTeam(context.Background(), 10, 20, 1); !customErrors.IsErrorType(err, customErrors.ErrTypeForbidden) {
		t.Fatalf("expected forbidden add by member, got %v", err)
	}

	if err := service.AddMemberToTeam(context.Background(), 10, 20, 2); !customErrors.IsErrorType(err, customErrors.ErrTypeConflict) {
		t.Fatalf("expected conflict on duplicate member, got %v", err)
	}

	if err := service.AddMemberToTeam(context.Background(), 10, 999, 2); !customErrors.IsErrorType(err, customErrors.ErrTypeNotFound) {
		t.Fatalf("expected not found for missing target user, got %v", err)
	}

	if err := service.RemoveMemberFromTeam(context.Background(), 10, 20, 1); !customErrors.IsErrorType(err, customErrors.ErrTypeForbidden) {
		t.Fatalf("expected forbidden remove by member, got %v", err)
	}

	if err := service.RemoveMemberFromTeam(context.Background(), 10, 20, 2); err != nil {
		t.Fatalf("unexpected remove error: %v", err)
	}
	if repo.lastRemoveTeamID != 10 || repo.lastRemoveUserID != 20 {
		t.Fatalf("expected remove call for team 10 user 20, got %d %d", repo.lastRemoveTeamID, repo.lastRemoveUserID)
	}

	if err := service.UpdateMemberRole(context.Background(), 10, 20, "invalid", 2); !customErrors.IsErrorType(err, customErrors.ErrTypeValidation) {
		t.Fatalf("expected validation error for invalid role, got %v", err)
	}

	if err := service.UpdateMemberRole(context.Background(), 10, 20, "manager", 2); err != nil {
		t.Fatalf("unexpected update role error: %v", err)
	}
	if repo.lastUpdateRoleTeam != 10 || repo.lastUpdateRoleUser != 20 || repo.lastUpdateRoleName != "manager" {
		t.Fatalf("expected update role call to be recorded, got %d %d %q", repo.lastUpdateRoleTeam, repo.lastUpdateRoleUser, repo.lastUpdateRoleName)
	}
}

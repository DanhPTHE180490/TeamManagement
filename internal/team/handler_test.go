package team

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	customErrors "team-management/internal/errors"
	"team-management/internal/models"

	"github.com/gin-gonic/gin"
)

type mockTeamService struct {
	createTeamFn       func(context.Context, string, int64, string) (*models.Team, error)
	getTeamByIDFn      func(context.Context, int64) (*models.Team, error)
	getTeamsByUserIDFn func(context.Context, int64) ([]*models.Team, error)
	updateTeamFn       func(context.Context, int64, string, int64) (*models.Team, error)
	deleteTeamFn       func(context.Context, int64, int64) error
	addMemberFn        func(context.Context, int64, int64, int64) error
	removeMemberFn     func(context.Context, int64, int64, int64) error
	updateMemberRoleFn func(context.Context, int64, int64, string, int64) error

	createTeamCalls       int
	getTeamByIDCalls      int
	getTeamsByUserIDCalls int
	updateTeamCalls       int
	deleteTeamCalls       int
	addMemberCalls        int
	removeMemberCalls     int
	updateMemberRoleCalls int

	lastCreateTeamName      string
	lastCreateTeamUserID    int64
	lastCreateTeamRole      string
	lastUpdateTeamID        int64
	lastUpdateTeamName      string
	lastUpdateTeamRequester int64
	lastDeleteTeamID        int64
	lastDeleteTeamRequester int64
	lastAddTeamID           int64
	lastAddTargetID         int64
	lastAddRequesterID      int64
	lastRemoveTeamID        int64
	lastRemoveTargetID      int64
	lastRemoveRequesterID   int64
	lastUpdateRoleTeamID    int64
	lastUpdateRoleTargetID  int64
	lastUpdateRoleName      string
	lastUpdateRoleRequester int64
}

func (m *mockTeamService) CreateTeam(_ context.Context, name string, userID int64, userRole string) (*models.Team, error) {
	m.createTeamCalls++
	m.lastCreateTeamName = name
	m.lastCreateTeamUserID = userID
	m.lastCreateTeamRole = userRole
	if m.createTeamFn != nil {
		return m.createTeamFn(context.Background(), name, userID, userRole)
	}
	return &models.Team{ID: 1, Name: name}, nil
}

func (m *mockTeamService) GetTeamByID(_ context.Context, id int64) (*models.Team, error) {
	m.getTeamByIDCalls++
	if m.getTeamByIDFn != nil {
		return m.getTeamByIDFn(context.Background(), id)
	}
	return &models.Team{ID: id, Name: "team"}, nil
}

func (m *mockTeamService) GetTeamsByUserID(_ context.Context, userID int64) ([]*models.Team, error) {
	m.getTeamsByUserIDCalls++
	if m.getTeamsByUserIDFn != nil {
		return m.getTeamsByUserIDFn(context.Background(), userID)
	}
	return []*models.Team{{ID: 1, Name: "team"}}, nil
}

func (m *mockTeamService) UpdateTeam(_ context.Context, id int64, name string, requesterID int64) (*models.Team, error) {
	m.updateTeamCalls++
	m.lastUpdateTeamID = id
	m.lastUpdateTeamName = name
	m.lastUpdateTeamRequester = requesterID
	if m.updateTeamFn != nil {
		return m.updateTeamFn(context.Background(), id, name, requesterID)
	}
	return &models.Team{ID: id, Name: name}, nil
}

func (m *mockTeamService) DeleteTeam(_ context.Context, id int64, requesterID int64) error {
	m.deleteTeamCalls++
	m.lastDeleteTeamID = id
	m.lastDeleteTeamRequester = requesterID
	if m.deleteTeamFn != nil {
		return m.deleteTeamFn(context.Background(), id, requesterID)
	}
	return nil
}

func (m *mockTeamService) AddMemberToTeam(_ context.Context, teamID int64, targetID int64, requesterID int64) error {
	m.addMemberCalls++
	m.lastAddTeamID = teamID
	m.lastAddTargetID = targetID
	m.lastAddRequesterID = requesterID
	if m.addMemberFn != nil {
		return m.addMemberFn(context.Background(), teamID, targetID, requesterID)
	}
	return nil
}

func (m *mockTeamService) RemoveMemberFromTeam(_ context.Context, teamID int64, targetID int64, requesterID int64) error {
	m.removeMemberCalls++
	m.lastRemoveTeamID = teamID
	m.lastRemoveTargetID = targetID
	m.lastRemoveRequesterID = requesterID
	if m.removeMemberFn != nil {
		return m.removeMemberFn(context.Background(), teamID, targetID, requesterID)
	}
	return nil
}

func (m *mockTeamService) UpdateMemberRole(_ context.Context, teamID int64, targetID int64, newRole string, requesterID int64) error {
	m.updateMemberRoleCalls++
	m.lastUpdateRoleTeamID = teamID
	m.lastUpdateRoleTargetID = targetID
	m.lastUpdateRoleName = newRole
	m.lastUpdateRoleRequester = requesterID
	if m.updateMemberRoleFn != nil {
		return m.updateMemberRoleFn(context.Background(), teamID, targetID, newRole, requesterID)
	}
	return nil
}

func newTeamRouterWithContext(service TeamService, userID any, userRole any) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if userID != nil {
			c.Set(ContextUserIDKey, userID)
		}
		if userRole != nil {
			c.Set(ContextUserRoleKey, userRole)
		}
		c.Next()
	})
	NewTeamHandler(service).RegisterRoutes(router.Group(""))
	return router
}

func TestTeamHandler_CreateTeam(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		userID       any
		userRole     any
		service      *mockTeamService
		expectedCode int
		expectedBody string
		callCount    int
	}{
		{
			name:         "success",
			body:         `{"name":"Platform"}`,
			userID:       float64(10),
			userRole:     "manager",
			service:      &mockTeamService{},
			expectedCode: http.StatusCreated,
			expectedBody: "Team created successfully",
			callCount:    1,
		},
		{
			name:         "member forbidden",
			body:         `{"name":"Platform"}`,
			userID:       float64(10),
			userRole:     "member",
			service:      &mockTeamService{},
			expectedCode: http.StatusForbidden,
			expectedBody: ForbiddenError,
			callCount:    0,
		},
		{
			name:         "validation error",
			body:         `{}`,
			userID:       float64(10),
			userRole:     "manager",
			service:      &mockTeamService{},
			expectedCode: http.StatusBadRequest,
			expectedBody: InvalidInput,
			callCount:    0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := newTeamRouterWithContext(tc.service, tc.userID, tc.userRole)
			req := httptest.NewRequest(http.MethodPost, "/teams/", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tc.expectedCode {
				t.Fatalf("expected status %d, got %d", tc.expectedCode, w.Code)
			}
			if !bytes.Contains(w.Body.Bytes(), []byte(tc.expectedBody)) {
				t.Fatalf("expected body to contain %q, got %s", tc.expectedBody, w.Body.String())
			}
			if tc.service.createTeamCalls != tc.callCount {
				t.Fatalf("expected CreateTeam to be called %d times, got %d", tc.callCount, tc.service.createTeamCalls)
			}
		})
	}
}

func TestTeamHandler_GetTeamByID(t *testing.T) {
	service := &mockTeamService{getTeamByIDFn: func(_ context.Context, id int64) (*models.Team, error) {
		if id == 99 {
			return nil, customErrors.NewNotFoundError("team")
		}
		return &models.Team{ID: id, Name: "platform"}, nil
	}}
	router := newTeamRouterWithContext(service, float64(1), "manager")

	tests := []struct {
		name         string
		path         string
		expectedCode int
		expectedBody string
	}{
		{name: "success", path: "/teams/1", expectedCode: http.StatusOK, expectedBody: "platform"},
		{name: "not found", path: "/teams/99", expectedCode: http.StatusNotFound, expectedBody: TeamNotFoundError},
		{name: "bad id", path: "/teams/not-a-number", expectedCode: http.StatusBadRequest, expectedBody: InvalidTeamIDError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tc.expectedCode {
				t.Fatalf("expected status %d, got %d", tc.expectedCode, w.Code)
			}
			if !bytes.Contains(w.Body.Bytes(), []byte(tc.expectedBody)) {
				t.Fatalf("expected body to contain %q, got %s", tc.expectedBody, w.Body.String())
			}
		})
	}
}

func TestTeamHandler_UpdateMemberRole_UsesCorrectServiceMethod(t *testing.T) {
	service := &mockTeamService{}
	router := newTeamRouterWithContext(service, float64(1), "manager")

	req := httptest.NewRequest(http.MethodPut, "/teams/10/members/20/role", bytes.NewBufferString(`{"role":"manager"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if service.updateMemberRoleCalls != 1 {
		t.Fatalf("expected UpdateMemberRole to be called once, got %d", service.updateMemberRoleCalls)
	}
	if service.removeMemberCalls != 0 {
		t.Fatalf("expected RemoveMemberFromTeam not to be called, got %d", service.removeMemberCalls)
	}
}

func TestTeamHandler_AddMemberToTeam_Conflict(t *testing.T) {
	service := &mockTeamService{addMemberFn: func(_ context.Context, _ int64, _ int64, _ int64) error {
		return customErrors.NewConflictError("user is already a member of this team", errors.New("duplicate"))
	}}
	router := newTeamRouterWithContext(service, float64(1), "manager")

	req := httptest.NewRequest(http.MethodPost, "/teams/10/members", bytes.NewBufferString(`{"user_id":20}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("already a member")) {
		t.Fatalf("expected conflict message, got %s", w.Body.String())
	}
}

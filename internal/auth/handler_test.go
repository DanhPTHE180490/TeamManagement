package auth

import (
	"bytes"
	stdErrors "errors"
	"net/http"
	"net/http/httptest"
	"testing"

	customErrors "team-management/internal/errors"
	"team-management/internal/models"

	"github.com/gin-gonic/gin"
)

type mockAuthService struct {
	registerUser  *models.User
	registerErr   error
	loginToken    string
	loginErr      error
	registerInput []string
	loginInput    []string
}

func (m *mockAuthService) Register(username, email, password, role string) (*models.User, error) {
	m.registerInput = []string{username, email, password, role}
	return m.registerUser, m.registerErr
}

func (m *mockAuthService) Login(email, password string) (string, error) {
	m.loginInput = []string{email, password}
	return m.loginToken, m.loginErr
}

func newAuthTestRouter(service AuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewAuthHandler(service).RegisterRoutes(router)
	return router
}

func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		service        *mockAuthService
		expectedCode   int
		expectedBody   string
		expectRegister bool
	}{
		{
			name:           "success",
			body:           `{"username":"alice","email":"alice@example.com","password":"password123","role":"manager"}`,
			service:        &mockAuthService{registerUser: &models.User{ID: 1, Username: "alice", Email: "alice@example.com", SystemRole: "manager"}},
			expectedCode:   http.StatusCreated,
			expectedBody:   "User registered successfully",
			expectRegister: true,
		},
		{
			name:           "duplicate email",
			body:           `{"username":"alice","email":"alice@example.com","password":"password123","role":"manager"}`,
			service:        &mockAuthService{registerErr: customErrors.NewDuplicateError("email", stdErrors.New("duplicate"))},
			expectedCode:   http.StatusConflict,
			expectedBody:   "Email already in use",
			expectRegister: true,
		},
		{
			name:           "validation error",
			body:           `{"username":"alice","email":"alice@example.com","password":"password123","role":"manager"}`,
			service:        &mockAuthService{registerErr: customErrors.NewValidationError("role", "must be manager or member")},
			expectedCode:   http.StatusBadRequest,
			expectedBody:   "Invalid role",
			expectRegister: true,
		},
		{
			name:         "invalid json",
			body:         `{"username":"alice"`,
			service:      &mockAuthService{},
			expectedCode: http.StatusBadRequest,
			expectedBody: "Invalid input",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := newAuthTestRouter(tc.service)
			req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tc.expectedCode {
				t.Fatalf("expected status %d, got %d", tc.expectedCode, w.Code)
			}
			if !bytes.Contains(w.Body.Bytes(), []byte(tc.expectedBody)) {
				t.Fatalf("expected response body to contain %q, got %s", tc.expectedBody, w.Body.String())
			}
			if tc.expectRegister && tc.service.registerInput == nil {
				t.Fatal("expected Register to be called")
			}
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		service      *mockAuthService
		expectedCode int
		expectedBody string
	}{
		{
			name:         "success",
			body:         `{"email":"alice@example.com","password":"password123"}`,
			service:      &mockAuthService{loginToken: "fake-token"},
			expectedCode: http.StatusOK,
			expectedBody: "Login successful",
		},
		{
			name:         "invalid credentials",
			body:         `{"email":"alice@example.com","password":"password123"}`,
			service:      &mockAuthService{loginErr: customErrors.NewUnauthorizedError("invalid email or password")},
			expectedCode: http.StatusUnauthorized,
			expectedBody: "Invalid credentials",
		},
		{
			name:         "server error",
			body:         `{"email":"alice@example.com","password":"password123"}`,
			service:      &mockAuthService{loginErr: customErrors.NewInternalError("boom", stdErrors.New("db down"))},
			expectedCode: http.StatusInternalServerError,
			expectedBody: "Failed to login",
		},
		{
			name:         "invalid json",
			body:         `{"email":"alice@example.com"`,
			service:      &mockAuthService{},
			expectedCode: http.StatusBadRequest,
			expectedBody: "Invalid input",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := newAuthTestRouter(tc.service)
			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tc.expectedCode {
				t.Fatalf("expected status %d, got %d", tc.expectedCode, w.Code)
			}
			if !bytes.Contains(w.Body.Bytes(), []byte(tc.expectedBody)) {
				t.Fatalf("expected response body to contain %q, got %s", tc.expectedBody, w.Body.String())
			}
		})
	}
}

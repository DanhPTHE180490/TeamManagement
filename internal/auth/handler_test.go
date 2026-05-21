package auth

import (
	"bytes"
	"context"
	stdErrors "errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"team-management/internal/models"
	customErrors "team-management/internal/utils"

	"github.com/gin-gonic/gin"
)

type mockAuthService struct {
	registerUser  *models.User
	registerErr   error
	loginToken    string
	loginErr      error
	bulkSummary   *BulkImportSummary
	bulkErr       error
	bulkCalled    bool
	registerInput []string
	loginInput    []string
	bulkInput     string
}

func (m *mockAuthService) Register(_ context.Context, username, email, password, role string) (*models.User, error) {
	m.registerInput = []string{username, email, password, role}
	return m.registerUser, m.registerErr
}

func (m *mockAuthService) Login(_ context.Context, email, password string) (string, error) {
	m.loginInput = []string{email, password}
	return m.loginToken, m.loginErr
}

func (m *mockAuthService) BulkImportUsersFromCSV(_ context.Context, reader io.Reader) (*BulkImportSummary, error) {
	m.bulkCalled = true
	data, _ := io.ReadAll(reader)
	m.bulkInput = string(data)
	return m.bulkSummary, m.bulkErr
}

func newAuthTestRouter(service AuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewAuthHandler(service).RegisterRoutes(router)
	return router
}

func newAuthProtectedTestRouter(service AuthService, userRole any) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if userRole != nil {
			c.Set("userRole", userRole)
		}
		c.Next()
	})
	NewAuthHandler(service).RegisterProtectedRoutes(router.Group("/api"))
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
			expectedBody:   "email already exists",
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
			expectedBody: "invalid email or password",
		},
		{
			name:         "server error",
			body:         `{"email":"alice@example.com","password":"password123"}`,
			service:      &mockAuthService{loginErr: customErrors.NewInternalError("Failed to login", stdErrors.New("db down"))},
			expectedCode: http.StatusInternalServerError,
			expectedBody: "Internal Server Error",
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

func TestAuthHandler_BulkImportUsers(t *testing.T) {
	tests := []struct {
		name         string
		userRole     any
		fileContent  string
		service      *mockAuthService
		expectedCode int
		expectedBody string
		expectInput  bool
	}{
		{
			name:         "forbidden for member",
			userRole:     "member",
			service:      &mockAuthService{},
			expectedCode: http.StatusForbidden,
			expectedBody: "forbidden: only managers can bulk import users",
		},
		{
			name:         "missing file",
			userRole:     "manager",
			service:      &mockAuthService{},
			expectedCode: http.StatusBadRequest,
			expectedBody: "failed to get file from request",
		},
		{
			name:        "success",
			userRole:    "manager",
			fileContent: "alice,alice@example.com,password123,manager\n",
			service: &mockAuthService{
				bulkSummary: &BulkImportSummary{TotalProcessed: 1, Succeeded: 1, Failed: 0},
			},
			expectedCode: http.StatusOK,
			expectedBody: "Import complete",
			expectInput:  true,
		},
		{
			name:         "service error",
			userRole:     "main_manager",
			fileContent:  "alice,alice@example.com,password123,manager\n",
			service:      &mockAuthService{bulkErr: stdErrors.New("boom")},
			expectedCode: http.StatusInternalServerError,
			expectedBody: "Internal Server Error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := newAuthProtectedTestRouter(tc.service, tc.userRole)

			var req *http.Request
			if tc.fileContent == "" && tc.name == "missing file" {
				req = httptest.NewRequest(http.MethodPost, "/api/auth/import-users", nil)
			} else {
				var body bytes.Buffer
				writer := multipart.NewWriter(&body)
				part, err := writer.CreateFormFile("file", "users.csv")
				if err != nil {
					t.Fatalf("failed to create multipart file: %v", err)
				}
				if _, err := io.Copy(part, bytes.NewBufferString(tc.fileContent)); err != nil {
					t.Fatalf("failed to write multipart body: %v", err)
				}
				if err := writer.Close(); err != nil {
					t.Fatalf("failed to close multipart writer: %v", err)
				}
				req = httptest.NewRequest(http.MethodPost, "/api/auth/import-users", &body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.expectedCode {
				t.Fatalf("expected status %d, got %d", tc.expectedCode, w.Code)
			}
			if !bytes.Contains(w.Body.Bytes(), []byte(tc.expectedBody)) {
				t.Fatalf("expected response body to contain %q, got %s", tc.expectedBody, w.Body.String())
			}
			if tc.expectInput && !bytes.Contains([]byte(tc.service.bulkInput), []byte("alice@example.com")) {
				t.Fatalf("expected uploaded CSV content to be passed to service, got %q", tc.service.bulkInput)
			}
		})
	}
}

func TestAuthHandler_BulkImportUsers_RejectsLargeUpload(t *testing.T) {
	previousLimit := maxBulkImportUploadBytes
	maxBulkImportUploadBytes = 64
	t.Cleanup(func() {
		maxBulkImportUploadBytes = previousLimit
	})

	repo := &mockAuthService{}
	router := newAuthProtectedTestRouter(repo, "manager")

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "users.csv")
	if err != nil {
		t.Fatalf("failed to create multipart file: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewBufferString("username,email,password,role\n"+strings.Repeat("a", 256))); err != nil {
		t.Fatalf("failed to write multipart body: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/import-users", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("file too large")) {
		t.Fatalf("expected response body to mention file too large, got %s", w.Body.String())
	}
	if repo.bulkCalled {
		t.Fatal("expected bulk import service not to be called")
	}
}

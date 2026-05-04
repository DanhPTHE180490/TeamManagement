package auth

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"team-management/internal/models"

	"github.com/gin-gonic/gin"
)

type mockAuthService struct{}

func (m *mockAuthService) Register(username, email, password, role string) (*models.User, error) {
	if email == "fail@example.com" {
		return nil, errors.New("simulated service error")
	}
	// Return a fake successful user
	return &models.User{
		ID:         1,
		Username:   username,
		Email:      email,
		SystemRole: role,
	}, nil
}

func (m *mockAuthService) Login(email, password string) (string, error) {
	if email == "fail@example.com" {
		return "", errors.New("invalid credentials")
	}
	return "fake-jwt-token", nil
}

func TestRegister_Handler(t *testing.T) {
	// Put Gin into Test Mode so it doesn't spam your console with debug logs
	gin.SetMode(gin.TestMode)

	// Setup our layers
	mockSvc := &mockAuthService{}
	handler := NewAuthHandler(mockSvc)

	// Setup a temporary Gin router just for this test
	router := gin.Default()
	handler.RegisterRoutes(router)

	// --- TEST CASE 1: Happy Path ---
	t.Run("Valid Registration", func(t *testing.T) {
		// The JSON body Postman would send
		body := []byte(`{
			"username": "TestUser",
			"email": "test@example.com",
			"password": "password123",
			"role": "manager"
		}`)

		// Create the fake request
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		// Create the fake response recorder
		w := httptest.NewRecorder()

		// Fire the request into the router!
		router.ServeHTTP(w, req)

		// Assert the Status Code
		if w.Code != http.StatusCreated { // 201
			t.Errorf("Expected status 201, got %d", w.Code)
		}
	})

	// --- TEST CASE 2: Bad JSON Input ---
	t.Run("Invalid JSON Missing Password", func(t *testing.T) {
		// Missing the required password field
		body := []byte(`{
			"username": "TestUser",
			"email": "test@example.com"
		}`)

		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// The handler's `ShouldBindJSON` should catch this and return a 400!
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

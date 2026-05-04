package auth

import (
	"errors"
	"testing"

	"team-management/internal/models"
)

type mockAuthRepo struct {
	users map[string]*models.User
}

func (m *mockAuthRepo) CreateUser(user *models.User) error {
	if _, exists := m.users[user.Email]; exists {
		return errors.New("Duplicate entry") // Simulate MySQL duplicate error
	}
	// "Save" the user to our memory map
	m.users[user.Email] = user
	user.ID = len(m.users) // Fake an auto-increment ID
	return nil
}

func (m *mockAuthRepo) GetUserByEmail(email string) (*models.User, error) {
	if user, exists := m.users[email]; exists {
		return user, nil
	}
	return nil, errors.New("not found")
}

func TestRegister_Service(t *testing.T) {
	// Setup our fake database and the service we are actually testing
	fakeRepo := &mockAuthRepo{users: make(map[string]*models.User)}
	authService := NewAuthService(fakeRepo)

	// 2. Define the Test Table
	tests := []struct {
		name          string
		inputUsername string
		inputEmail    string
		inputPassword string
		inputRole     string
		expectError   bool
		expectedRole  string
	}{
		{
			name:          "Happy Path - Valid Manager",
			inputUsername: "Alice",
			inputEmail:    "alice@example.com",
			inputPassword: "password123",
			inputRole:     "manager",
			expectError:   false,
			expectedRole:  "manager", // Should keep the manager role
		},
		{
			name:          "Role Fallback - Invalid Role Gets Member",
			inputUsername: "Bob",
			inputEmail:    "bob@example.com",
			inputPassword: "password123",
			inputRole:     "superadmin", // Invalid!
			expectError:   false,
			expectedRole:  "member", // The service logic should fix this
		},
		{
			name:          "Duplicate Email Fails",
			inputUsername: "Alice 2",
			inputEmail:    "alice@example.com", // Already used in test 1!
			inputPassword: "password123",
			inputRole:     "member",
			expectError:   true,
		},
	}

	// 3. Execute the Table
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			user, err := authService.Register(tc.inputUsername, tc.inputEmail, tc.inputPassword, tc.inputRole)

			// Assert Error State
			if tc.expectError && err == nil {
				t.Errorf("Expected an error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Did not expect error, got: %v", err)
			}

			// Assert Business Logic (Did the invalid role get corrected?)
			if !tc.expectError && user.SystemRole != tc.expectedRole {
				t.Errorf("Expected role %s, got %s", tc.expectedRole, user.SystemRole)
			}
		})
	}
}

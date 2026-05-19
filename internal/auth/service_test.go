package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	customErrors "team-management/internal/errors"
	"team-management/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type mockAuthRepo struct {
	createUserFn     func(*models.User) error
	getUserByEmailFn func(context.Context, string) (*models.User, error)
	createdUsers     []*models.User
	lookedUpEmails   []string
}

func (m *mockAuthRepo) CreateUser(_ context.Context, user *models.User) error {
	m.createdUsers = append(m.createdUsers, user)
	if m.createUserFn != nil {
		return m.createUserFn(user)
	}
	user.ID = len(m.createdUsers)
	return nil
}

func (m *mockAuthRepo) GetUserByEmail(_ context.Context, email string) (*models.User, error) {
	m.lookedUpEmails = append(m.lookedUpEmails, email)
	if m.getUserByEmailFn != nil {
		return m.getUserByEmailFn(context.Background(), email)
	}
	return nil, customErrors.NewNotFoundError("user")
}

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		email     string
		password  string
		role      string
		setupRepo func(*mockAuthRepo)
		wantRole  string
		wantErr   bool
		wantType  customErrors.ErrorType
	}{
		{
			name:     "success manager",
			username: "alice",
			email:    "alice@example.com",
			password: "password123",
			role:     "manager",
			wantRole: "manager",
		},
		{
			name:     "invalid role",
			username: "alice",
			email:    "alice@example.com",
			password: "password123",
			role:     "superadmin",
			wantErr:  true,
			wantType: customErrors.ErrTypeValidation,
		},
		{
			name:     "duplicate email",
			username: "alice",
			email:    "alice@example.com",
			password: "password123",
			role:     "member",
			setupRepo: func(repo *mockAuthRepo) {
				repo.createUserFn = func(*models.User) error {
					return customErrors.NewDuplicateError("email", errors.New("duplicate key"))
				}
			},
			wantErr:  true,
			wantType: customErrors.ErrTypeDuplicate,
		},
		{
			name:     "empty username",
			username: "",
			email:    "alice@example.com",
			password: "password123",
			role:     "member",
			wantErr:  true,
			wantType: customErrors.ErrTypeValidation,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockAuthRepo{}
			if tc.setupRepo != nil {
				tc.setupRepo(repo)
			}
			service := NewAuthService(repo)

			user, err := service.Register(context.Background(), tc.username, tc.email, tc.password, tc.role)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !customErrors.IsErrorType(err, tc.wantType) {
					t.Fatalf("expected error type %s, got %v", tc.wantType, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if user == nil {
				t.Fatal("expected user, got nil")
			}
			if user.SystemRole != tc.wantRole {
				t.Fatalf("expected role %q, got %q", tc.wantRole, user.SystemRole)
			}
			if user.PasswordHash == tc.password {
				t.Fatal("password was not hashed")
			}
			if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(tc.password)); err != nil {
				t.Fatalf("hashed password does not match input: %v", err)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash setup failed: %v", err)
	}

	baseUser := &models.User{
		ID:           42,
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: string(hashedPassword),
		SystemRole:   "manager",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	tests := []struct {
		name      string
		email     string
		password  string
		setupRepo func(*mockAuthRepo)
		wantErr   bool
		wantType  customErrors.ErrorType
	}{
		{
			name:     "success",
			email:    "alice@example.com",
			password: "password123",
			setupRepo: func(repo *mockAuthRepo) {
				repo.getUserByEmailFn = func(context.Context, string) (*models.User, error) {
					return baseUser, nil
				}
			},
		},
		{
			name:     "wrong password",
			email:    "alice@example.com",
			password: "wrong-password",
			setupRepo: func(repo *mockAuthRepo) {
				repo.getUserByEmailFn = func(context.Context, string) (*models.User, error) {
					return baseUser, nil
				}
			},
			wantErr:  true,
			wantType: customErrors.ErrTypeUnauthorized,
		},
		{
			name:     "missing user",
			email:    "missing@example.com",
			password: "password123",
			setupRepo: func(repo *mockAuthRepo) {
				repo.getUserByEmailFn = func(context.Context, string) (*models.User, error) {
					return nil, customErrors.NewNotFoundError("user")
				}
			},
			wantErr:  true,
			wantType: customErrors.ErrTypeUnauthorized,
		},
		{
			name:     "repository error",
			email:    "alice@example.com",
			password: "password123",
			setupRepo: func(repo *mockAuthRepo) {
				repo.getUserByEmailFn = func(context.Context, string) (*models.User, error) {
					return nil, customErrors.NewInternalError("db unavailable", errors.New("boom"))
				}
			},
			wantErr:  true,
			wantType: customErrors.ErrTypeInternal,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockAuthRepo{}
			if tc.setupRepo != nil {
				tc.setupRepo(repo)
			}
			service := NewAuthService(repo)

			token, err := service.Login(context.Background(), tc.email, tc.password)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !customErrors.IsErrorType(err, tc.wantType) {
					t.Fatalf("expected error type %s, got %v", tc.wantType, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if token == "" {
				t.Fatal("expected token, got empty string")
			}

			parsed, err := jwt.Parse(token, func(token *jwt.Token) (any, error) {
				return jwtSecret, nil
			})
			if err != nil || !parsed.Valid {
				t.Fatalf("token should be valid: %v", err)
			}
			claims, ok := parsed.Claims.(jwt.MapClaims)
			if !ok {
				t.Fatal("expected jwt.MapClaims")
			}
			if claims["user_id"] != float64(42) {
				t.Fatalf("expected user_id 42, got %v", claims["user_id"])
			}
			if claims["system_role"] != "manager" {
				t.Fatalf("expected system_role manager, got %v", claims["system_role"])
			}
			if _, ok := claims["exp"]; !ok {
				t.Fatal("expected exp claim to exist")
			}
		})
	}
}

func TestAuthService_BulkImportUsersFromCSV(t *testing.T) {
	repo := &mockAuthRepo{}
	service := NewAuthService(repo)

	summary, err := service.BulkImportUsersFromCSV(context.Background(), strings.NewReader("username,email,password,role\nalice,alice@example.com,password123,manager\nbad-row-only\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.TotalProcessed != 2 {
		t.Fatalf("expected 2 processed rows, got %d", summary.TotalProcessed)
	}
	if summary.Succeeded != 1 {
		t.Fatalf("expected 1 success, got %d", summary.Succeeded)
	}
	if summary.Failed != 1 {
		t.Fatalf("expected 1 failure, got %d", summary.Failed)
	}
	if len(summary.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(summary.Errors))
	}
	if !strings.Contains(summary.Errors[0], "Row 3: CSV read error:") {
		t.Fatalf("unexpected error message: %q", summary.Errors[0])
	}
	if len(repo.createdUsers) != 1 {
		t.Fatalf("expected 1 created user, got %d", len(repo.createdUsers))
	}
	if repo.createdUsers[0].Email != "alice@example.com" {
		t.Fatalf("unexpected created user email: %s", repo.createdUsers[0].Email)
	}
	if repo.createdUsers[0].SystemRole != "manager" {
		t.Fatalf("unexpected created user role: %s", repo.createdUsers[0].SystemRole)
	}
}

func TestAuthService_BulkImportUsersFromCSV_RespectsRowLimit(t *testing.T) {
	previousLimit := maxBulkImportRows
	maxBulkImportRows = 1
	t.Cleanup(func() {
		maxBulkImportRows = previousLimit
	})

	repo := &mockAuthRepo{}
	service := NewAuthService(repo)

	summary, err := service.BulkImportUsersFromCSV(context.Background(), strings.NewReader("username,email,password,role\nalice,alice@example.com,password123,manager\nbob,bob@example.com,password123,member\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.TotalProcessed != 2 {
		t.Fatalf("expected 2 processed rows, got %d", summary.TotalProcessed)
	}
	if summary.Succeeded != 1 {
		t.Fatalf("expected 1 success, got %d", summary.Succeeded)
	}
	if summary.Failed != 1 {
		t.Fatalf("expected 1 failure, got %d", summary.Failed)
	}
	if len(summary.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(summary.Errors))
	}
	if !strings.Contains(summary.Errors[0], "bulk import limit exceeded") {
		t.Fatalf("unexpected error message: %q", summary.Errors[0])
	}
	if len(repo.createdUsers) != 1 {
		t.Fatalf("expected 1 created user, got %d", len(repo.createdUsers))
	}
}

package auth

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"team-management/internal/errors"
	"team-management/internal/models"
)

type AuthRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
}

type authRepositoryImpl struct {
	db *sql.DB
}

// NewAuthRepository acts as a constructor
func NewAuthRepository(db *sql.DB) AuthRepository {
	return &authRepositoryImpl{db: db}
}

// CreateUser executes the INSERT statement
func (r *authRepositoryImpl) CreateUser(ctx context.Context, user *models.User) error {
	query := `INSERT INTO users (username, email, password_hash, system_role) 
              VALUES (?, ?, ?, ?)`

	result, err := r.db.ExecContext(ctx, query, user.Username, user.Email, user.PasswordHash, user.SystemRole)
	if err != nil {
		// Handle duplicate key error
		if strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "Duplicate entry") {
			log.Printf("Duplicate email attempted: %s", user.Email)
			return errors.NewDuplicateError("email", err)
		}
		log.Printf("Database error creating user %s: %v", user.Email, err)
		return errors.NewInternalError("failed to create user in database", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Printf("Failed to get inserted user ID for email %s: %v", user.Email, err)
		return errors.NewInternalError("failed to retrieve user ID", err)
	}
	user.ID = int(id)

	return nil
}

// GetUserByEmail executes the SELECT statement
func (r *authRepositoryImpl) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, username, email, password_hash, system_role, created_at, updated_at 
          FROM users WHERE email = ?`

	user := &models.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.SystemRole,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("User not found with email: %s", email)
			return nil, errors.NewNotFoundError("user")
		}
		log.Printf("Database error retrieving user by email %s: %v", email, err)
		return nil, errors.NewInternalError("failed to retrieve user", err)
	}

	return user, nil
}

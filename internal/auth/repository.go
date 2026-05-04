package auth

import (
	"database/sql"
	"team-management/internal/models"
)

type AuthRepository interface {
	CreateUser(user *models.User) error
	GetUserByEmail(email string) (*models.User, error)
}

// authRepositoryImpl is the concrete implementation of the interface
type authRepositoryImpl struct {
	db *sql.DB
}

// NewAuthRepository acts as a constructor
func NewAuthRepository(db *sql.DB) AuthRepository {
	return &authRepositoryImpl{db: db}
}

// CreateUser executes the INSERT statement
func (r *authRepositoryImpl) CreateUser(user *models.User) error {
	query := `INSERT INTO users (username, email, password_hash, system_role) 
              VALUES (?, ?, ?, ?)`

	result, err := r.db.Exec(query, user.Username, user.Email, user.PasswordHash, user.SystemRole)
	if err != nil {
		return err
	}

	// Retrieve the auto-generated ID
	id, err := result.LastInsertId()
	if err == nil {
		user.ID = int(id)
	}

	return nil
}

// GetUserByEmail executes the SELECT statement
func (r *authRepositoryImpl) GetUserByEmail(email string) (*models.User, error) {
	query := `SELECT id, username, email, password_hash, system_role, created_at, updated_at 
	          FROM users WHERE email = ?`

	user := &models.User{}
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.SystemRole,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err // Returns sql.ErrNoRows if user is not found
	}

	return user, nil
}

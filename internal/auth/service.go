package auth

import (
	"errors"
	"team-management/internal/models"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("super-secret-capstone-key")

// AuthService defines the business logic interface
type AuthService interface {
	Register(username, email, password, role string) (*models.User, error)
	Login(email, password string) (string, error)
}

type authServiceImpl struct {
	repo AuthRepository
}

// NewAuthService acts as a constructor
func NewAuthService(repo AuthRepository) AuthService {
	return &authServiceImpl{repo: repo}
}

func (s *authServiceImpl) Register(username, email, password, role string) (*models.User, error) {
	// 1. Hash the password (DefaultCost is 10)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 2. Validate role strictly (default to 'member' if they try to pass garbage)
	if role != "manager" {
		role = "member"
	}

	// 3. Construct the user model
	user := &models.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		SystemRole:   role,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 4. Save via the Repository layer
	err = s.repo.CreateUser(user)
	if err != nil {
		return nil, err // This will bubble up if the email already exists
	}

	return user, nil
}

func (s *authServiceImpl) Login(email, password string) (string, error) {
	// 1. Fetch user by email via Repository
	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		return "", errors.New("invalid email or password") // Keep errors generic for security
	}

	// 2. Compare the stored hash with the provided password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", errors.New("invalid email or password")
	}

	// 3. Generate JWT Token (Embed the ID and Role so we don't have to query the DB on every request)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":     user.ID,
		"system_role": user.SystemRole,
		"exp":         time.Now().Add(time.Hour * 72).Unix(), // Token expires in 72 hours
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

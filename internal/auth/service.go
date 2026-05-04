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
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	if role != "manager" {
		role = "member"
	}

	user := &models.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		SystemRole:   role,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = s.repo.CreateUser(user)
	if err != nil {
		return nil, err // This will bubble up if the email already exists
	}

	return user, nil
}

func (s *authServiceImpl) Login(email, password string) (string, error) {
	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		return "", errors.New("invalid email or password") // Keep errors generic for security
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", errors.New("invalid email or password")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":     user.ID,
		"system_role": user.SystemRole,
		"exp":         time.Now().Add(time.Hour * 72).Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

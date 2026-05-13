package auth

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"sync"
	"team-management/internal/errors"
	"team-management/internal/models"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("super-secret-capstone-key")

type UserImportJob struct {
	RowNumber int
	Username  string
	Email     string
	Password  string
	Role      string
}

type UserImportResult struct {
	RowNumber int
	Success   bool
	ErrorMsg  string
}

type BulkImportSummary struct {
	TotalProcessed int      `json:"total_processed"`
	Succeeded      int      `json:"succeeded"`
	Failed         int      `json:"failed"`
	Errors         []string `json:"errors"`
}

// AuthService defines the business logic interface
type AuthService interface {
	Register(username, email, password, role string) (*models.User, error)
	Login(email, password string) (string, error)
	BulkImportUsersFromCSV(reader io.Reader) (*BulkImportSummary, error)
}

type authServiceImpl struct {
	repo AuthRepository
}

// NewAuthService acts as a constructor
func NewAuthService(repo AuthRepository) AuthService {
	return &authServiceImpl{repo: repo}
}

func (s *authServiceImpl) Register(username, email, password, role string) (*models.User, error) {
	// Validate input
	if len(username) == 0 || len(username) > 50 {
		return nil, errors.NewValidationError("username", "must be between 1 and 50 characters")
	}

	if len(email) == 0 {
		return nil, errors.NewValidationError("email", "cannot be empty")
	}

	if len(password) < 6 {
		return nil, errors.NewValidationError("password", "must be at least 6 characters")
	}

	// Validate role
	if role != "manager" && role != "member" && role != "admin" {
		return nil, errors.NewValidationError("role", "must be 'manager', 'member', or 'admin'")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash password for user %s: %v", username, err)
		return nil, errors.NewInternalError("Failed to process password", err)
	}

	// Enforce role rules: only allow manager and admin, default others to member
	if role != "manager" && role != "admin" {
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
		if errors.IsErrorType(err, errors.ErrTypeDuplicate) {
			return nil, err // Already wrapped as duplicate error
		}
		log.Printf("Failed to create user %s: %v", email, err)
		return nil, errors.NewInternalError("Failed to create user", err)
	}

	log.Printf("User registered successfully: %s (email: %s, role: %s)", username, email, role)
	return user, nil
}

func (s *authServiceImpl) Login(email, password string) (string, error) {
	// Validate input
	if len(email) == 0 {
		return "", errors.NewValidationError("email", "cannot be empty")
	}
	if len(password) == 0 {
		return "", errors.NewValidationError("password", "cannot be empty")
	}

	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		if errors.IsErrorType(err, errors.ErrTypeNotFound) {
			log.Printf("Login attempt for non-existent user: %s", email)
			return "", errors.NewUnauthorizedError("invalid email or password")
		}
		log.Printf("Database error during login for email %s: %v", email, err)
		return "", errors.NewInternalError("Failed to authenticate user", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		log.Printf("Failed password attempt for user: %s", email)
		return "", errors.NewUnauthorizedError("invalid email or password")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":     user.ID,
		"system_role": user.SystemRole,
		"exp":         time.Now().Add(time.Hour * 72).Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		log.Printf("Failed to generate JWT token for user %d: %v", user.ID, err)
		return "", errors.NewInternalError("Failed to generate authentication token", err)
	}

	log.Printf("User logged in successfully: %s (ID: %d)", email, user.ID)
	return tokenString, nil
}

func (s *authServiceImpl) BulkImportUsersFromCSV(reader io.Reader) (*BulkImportSummary, error) {
	csvReader := csv.NewReader(reader)
	var wg sync.WaitGroup
	jobChan := make(chan UserImportJob)
	resultChan := make(chan UserImportResult)

	numWorkers := 5
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go s.processImportJobs(&wg, jobChan, resultChan)
	}

	go func() {
		defer close(jobChan)
		if _, err := csvReader.Read(); err != nil {
			return
		}

		rowNumber := 2
		for {
			record, err := csvReader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("Error reading CSV at row %d: %v", rowNumber, err)
				resultChan <- UserImportResult{RowNumber: rowNumber, Success: false, ErrorMsg: fmt.Sprintf("CSV read error: %v", err)}
				rowNumber++
				continue
			}

			if len(record) < 4 {
				log.Printf("Invalid CSV format at row %d: expected 4 fields, got %d", rowNumber, len(record))
				resultChan <- UserImportResult{RowNumber: rowNumber, Success: false, ErrorMsg: "Invalid CSV format: expected 4 fields"}
				rowNumber++
				continue
			}

			jobChan <- UserImportJob{
				RowNumber: rowNumber,
				Username:  record[0],
				Email:     record[1],
				Password:  record[2],
				Role:      record[3],
			}
			rowNumber++
		}
	}()

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	summary := &BulkImportSummary{}
	for result := range resultChan {
		summary.TotalProcessed++
		if result.Success {
			summary.Succeeded++
		} else {
			summary.Failed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("Row %d: %s", result.RowNumber, result.ErrorMsg))
		}
	}

	log.Printf("Bulk import completed: %d processed, %d succeeded, %d failed", summary.TotalProcessed, summary.Succeeded, summary.Failed)
	return summary, nil
}

func (s *authServiceImpl) processImportJobs(wg *sync.WaitGroup, jobChan <-chan UserImportJob, resultChan chan<- UserImportResult) {
	defer wg.Done()
	for job := range jobChan {
		_, err := s.Register(job.Username, job.Email, job.Password, job.Role)
		if err != nil {
			resultChan <- UserImportResult{RowNumber: job.RowNumber, Success: false, ErrorMsg: err.Error()}
		} else {
			resultChan <- UserImportResult{RowNumber: job.RowNumber, Success: true}
		}
	}
}

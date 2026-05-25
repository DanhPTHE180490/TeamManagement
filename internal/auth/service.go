package auth

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"team-management/internal/models"
	apperrors "team-management/internal/utils"
	"time"

	"team-management/internal/audit"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

var maxBulkImportRows = 1000

type UserImportJob struct {
	RowNumber int
	Username  string
	Email     string
	Password  string
	Role      string
	ErrorMsg  string
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
	Register(ctx context.Context, username, email, password, role string) (*models.User, error)
	Login(ctx context.Context, email, password string) (string, error)
	BulkImportUsersFromCSV(ctx context.Context, requesterID int64, reader io.Reader) (*BulkImportSummary, error)
}

type authServiceImpl struct {
	repo        AuthRepository
	redisClient *redis.Client
}

type contextAwareReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextAwareReader) Read(p []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
	}
	return r.reader.Read(p)
}
func newContextAwareReader(ctx context.Context, reader io.Reader) io.Reader {
	return &contextAwareReader{
		ctx:    ctx,
		reader: reader,
	}
}

// NewAuthService acts as a constructor
func NewAuthService(repo AuthRepository, redisClient *redis.Client) AuthService {
	return &authServiceImpl{repo: repo, redisClient: redisClient}
}

func (s *authServiceImpl) Register(ctx context.Context, username, email, password, role string) (*models.User, error) {
	// Validate input
	if len(username) == 0 || len(username) > 50 {
		return nil, apperrors.NewValidationError("username", "must be between 1 and 50 characters")
	}

	if len(email) == 0 {
		return nil, apperrors.NewValidationError("email", "cannot be empty")
	}

	if len(password) < 6 {
		return nil, apperrors.NewValidationError("password", "must be at least 6 characters")
	}

	// Validate role
	if role != "manager" && role != "member" && role != "main_manager" {
		return nil, apperrors.NewValidationError("role", "must be 'manager', 'member', or 'main_manager'")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash password for user %s: %v", username, err)
		return nil, apperrors.NewInternalError("Failed to process password", err)
	}

	// Enforce role rules: only allow manager, default others to member
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

	err = s.repo.CreateUser(ctx, user)
	if err != nil {
		if apperrors.IsErrorType(err, apperrors.ErrTypeDuplicate) {
			return nil, err // Already wrapped as duplicate error
		}
		log.Printf("Failed to create user %s: %v", email, err)
		return nil, apperrors.NewInternalError("Failed to create user", err)
	}

	userID := int64(user.ID)
	entityType := "user"

	audit.PublishEvent(
		s.redisClient,
		&userID,
		"USER_REGISTERED",
		entityType,
		&userID,
		map[string]any{"role": role, "email": email},
	)

	log.Printf("User registered successfully: %s (email: %s, role: %s)", username, email, role)
	return user, nil
}

func (s *authServiceImpl) Login(ctx context.Context, email, password string) (string, error) {
	// Validate input
	if len(email) == 0 {
		return "", apperrors.NewValidationError("email", "cannot be empty")
	}
	if len(password) == 0 {
		return "", apperrors.NewValidationError("password", "cannot be empty")
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if apperrors.IsErrorType(err, apperrors.ErrTypeNotFound) {
			log.Printf("Login attempt for non-existent user: %s", email)
			return "", apperrors.NewUnauthorizedError("invalid email or password")
		}
		log.Printf("Database error during login for email %s: %v", email, err)
		return "", apperrors.NewInternalError("Failed to authenticate user", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		log.Printf("Failed password attempt for user: %s", email)
		return "", apperrors.NewUnauthorizedError("invalid email or password")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":     user.ID,
		"system_role": user.SystemRole,
		"exp":         time.Now().Add(time.Hour * 72).Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		log.Printf("Failed to generate JWT token for user %d: %v", user.ID, err)
		return "", apperrors.NewInternalError("Failed to generate authentication token", err)
	}

	userID := int64(user.ID)
	entityType := "user"

	audit.PublishEvent(
		s.redisClient,
		&userID,
		"USER_LOGGED_IN",
		entityType,
		&userID,
		map[string]any{"email": email},
	)

	log.Printf("User logged in successfully: %s (ID: %d)", email, user.ID)
	return tokenString, nil
}

func (s *authServiceImpl) BulkImportUsersFromCSV(ctx context.Context, requesterID int64, reader io.Reader) (*BulkImportSummary, error) {
	csvReader := csv.NewReader(newContextAwareReader(ctx, reader))
	var wg sync.WaitGroup
	jobChan := make(chan UserImportJob)
	resultChan := make(chan UserImportResult)

	numWorkers := 5
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go s.processImportJobs(ctx, &wg, jobChan, resultChan)
	}

	go func() {
		defer close(jobChan)
		if _, err := csvReader.Read(); err != nil {
			if err == io.EOF {
				return
			}
			jobChan <- UserImportJob{RowNumber: 1, ErrorMsg: fmt.Sprintf("CSV header read error: %v", err)}
			return
		}

		rowNumber := 2
		rowsSeen := 0
		for {
			if rowsSeen >= maxBulkImportRows {
				jobChan <- UserImportJob{RowNumber: rowNumber, ErrorMsg: fmt.Sprintf("bulk import limit exceeded: maximum %d rows", maxBulkImportRows)}
				return
			}

			record, err := csvReader.Read()
			if err == io.EOF {
				break
			}
			rowsSeen++
			if err != nil {
				log.Printf("Error reading CSV at row %d: %v", rowNumber, err)
				jobChan <- UserImportJob{RowNumber: rowNumber, ErrorMsg: fmt.Sprintf("CSV read error: %v", err)}
				rowNumber++
				continue
			}

			if len(record) < 4 {
				log.Printf("Invalid CSV format at row %d: expected 4 fields, got %d", rowNumber, len(record))
				jobChan <- UserImportJob{RowNumber: rowNumber, ErrorMsg: "Invalid CSV format: expected 4 fields"}
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

	audit.PublishEvent(
		s.redisClient,
		&requesterID,
		"BULK_IMPORT_COMPLETED",
		"system",
		nil,
		map[string]any{
			"total_processed": summary.TotalProcessed,
			"succeeded":       summary.Succeeded,
			"failed":          summary.Failed,
		},
	)

	log.Printf("Bulk import completed: %d processed, %d succeeded, %d failed", summary.TotalProcessed, summary.Succeeded, summary.Failed)
	return summary, nil
}

func (s *authServiceImpl) processImportJobs(ctx context.Context, wg *sync.WaitGroup, jobChan <-chan UserImportJob, resultChan chan<- UserImportResult) {
	defer wg.Done()
	for job := range jobChan {
		if job.ErrorMsg != "" {
			resultChan <- UserImportResult{RowNumber: job.RowNumber, Success: false, ErrorMsg: job.ErrorMsg}
			continue
		}
		_, err := s.Register(ctx, job.Username, job.Email, job.Password, job.Role)
		if err != nil {
			log.Printf("Row %d import failed for user %s: %v", job.RowNumber, job.Username, err)

			safeError := "failed to register user"

			if strings.Contains(err.Error(), "must be") || strings.Contains(err.Error(), "cannot be empty") || strings.Contains(err.Error(), "duplicate") {
				safeError = err.Error()
			}

			resultChan <- UserImportResult{RowNumber: job.RowNumber, Success: false, ErrorMsg: safeError}
		} else {
			resultChan <- UserImportResult{RowNumber: job.RowNumber, Success: true}
		}
	}
}

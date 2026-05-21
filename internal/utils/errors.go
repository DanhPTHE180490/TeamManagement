package utils

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// Error types for consistent error handling
type ErrorType string

const (
	ErrTypeDuplicate      ErrorType = "Duplicate"
	ErrTypeNotFound       ErrorType = "Not Found"
	ErrTypeUnauthorized   ErrorType = "Unauthorized"
	ErrTypeForbidden      ErrorType = "Forbidden"
	ErrTypeValidation     ErrorType = "Validation"
	ErrTypeInternal       ErrorType = "Internal"
	ErrTypeConflict       ErrorType = "Conflict"
	ErrTypeInvalidId      ErrorType = "Invalid ID"
	ErrTypeInternalServer ErrorType = "Internal Server Error"
)

// AppError wraps errors with type and message information
type AppError struct {
	Type    ErrorType
	Message string
	Details []string
	Err     error
}

// Error implements the error interface
func (e *AppError) Error() string {
	base := fmt.Sprintf("[%s] %s", e.Type, e.Message)
	if len(e.Details) > 0 {
		base += fmt.Sprintf(" (details: %v)", e.Details)
	}
	if e.Err != nil {
		base += fmt.Sprintf(": %v", e.Err)
	}
	return base
}

// Unwrap returns the wrapped error
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewError creates a new AppError
func NewError(errType ErrorType, message string, err error) *AppError {
	return &AppError{
		Type:    errType,
		Message: message,
		Err:     err,
	}
}

func FormatValidationError(err error) *AppError {
	var details []string
	message := "Invalid input"

	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		message = "Validation failed"
		for _, e := range validationErrs {
			details = append(details, e.Field()+" failed validation: "+e.Tag())
		}
	}

	customErr := NewError(ErrTypeValidation, message, err)
	customErr.Details = details

	return customErr
}

func MapErrorToHTTPStatus(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		switch appErr.Type {
		case ErrTypeValidation:
			return http.StatusBadRequest
		case ErrTypeDuplicate, ErrTypeConflict:
			return http.StatusConflict
		case ErrTypeNotFound:
			return http.StatusNotFound
		case ErrTypeUnauthorized:
			return http.StatusUnauthorized
		case ErrTypeForbidden:
			return http.StatusForbidden
		case ErrTypeInvalidId:
			return http.StatusBadRequest
		default:
			return http.StatusInternalServerError
		}
	}
	// If it's not our custom error, default to 500
	return http.StatusInternalServerError
}

// NewDuplicateError creates a duplicate entry error
func NewDuplicateError(resource string, err error) *AppError {
	return NewError(ErrTypeDuplicate, fmt.Sprintf("%s already exists", resource), err)
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string) *AppError {
	return NewError(ErrTypeNotFound, fmt.Sprintf("%s not found", resource), nil)
}

// NewUnauthorizedError creates an unauthorized error
func NewUnauthorizedError(message string) *AppError {
	return NewError(ErrTypeUnauthorized, message, nil)
}

// NewForbiddenError creates a forbidden error
func NewForbiddenError(message string) *AppError {
	return NewError(ErrTypeForbidden, message, nil)
}

// NewValidationError creates a validation error
func NewValidationError(field string, reason string) *AppError {
	return NewError(ErrTypeValidation, fmt.Sprintf("Invalid %s: %s", field, reason), nil)
}

// NewInternalError creates an internal server error
func NewInternalError(message string, err error) *AppError {
	// Use a stable, client-safe message for internal errors to avoid leaking
	// low-level implementation details to API clients. Preserve the original
	// message in Details for logging/diagnostics.
	appErr := NewError(ErrTypeInternal, "Internal server error", err)
	if message != "" {
		appErr.Details = []string{message}
	}
	return appErr
}

// NewConflictError creates a conflict error
func NewConflictError(message string, err error) *AppError {
	return NewError(ErrTypeConflict, message, err)
}

// NewInvalidIdError creates an invalid ID error
func NewInvalidIdError(message string) *AppError {
	return NewError(ErrTypeInvalidId, message, nil)
}

// IsErrorType checks if an error is of a specific type
func IsErrorType(err error, errType ErrorType) bool {
	var appErr *AppError
	// errors.As unwraps the error chain and populates appErr if a match is found
	if errors.As(err, &appErr) {
		return appErr.Type == errType
	}
	return false
}

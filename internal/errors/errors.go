package errors

import "fmt"

// Error types for consistent error handling
type ErrorType string

const (
	ErrTypeDuplicate    ErrorType = "duplicate"
	ErrTypeNotFound     ErrorType = "not_found"
	ErrTypeUnauthorized ErrorType = "unauthorized"
	ErrTypeForbidden    ErrorType = "forbidden"
	ErrTypeValidation   ErrorType = "validation"
	ErrTypeInternal     ErrorType = "internal"
	ErrTypeConflict     ErrorType = "conflict"
)

// CustomError wraps errors with type and message information
type CustomError struct {
	Type    ErrorType
	Message string
	Err     error
}

// Error implements the error interface
func (e *CustomError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap returns the wrapped error
func (e *CustomError) Unwrap() error {
	return e.Err
}

// NewError creates a new CustomError
func NewError(errType ErrorType, message string, err error) *CustomError {
	return &CustomError{
		Type:    errType,
		Message: message,
		Err:     err,
	}
}

// NewDuplicateError creates a duplicate entry error
func NewDuplicateError(resource string, err error) *CustomError {
	return NewError(ErrTypeDuplicate, fmt.Sprintf("%s already exists", resource), err)
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string) *CustomError {
	return NewError(ErrTypeNotFound, fmt.Sprintf("%s not found", resource), nil)
}

// NewUnauthorizedError creates an unauthorized error
func NewUnauthorizedError(message string) *CustomError {
	return NewError(ErrTypeUnauthorized, message, nil)
}

// NewForbiddenError creates a forbidden error
func NewForbiddenError(message string) *CustomError {
	return NewError(ErrTypeForbidden, message, nil)
}

// NewValidationError creates a validation error
func NewValidationError(field string, reason string) *CustomError {
	return NewError(ErrTypeValidation, fmt.Sprintf("Invalid %s: %s", field, reason), nil)
}

// NewInternalError creates an internal server error
func NewInternalError(message string, err error) *CustomError {
	return NewError(ErrTypeInternal, message, err)
}

// NewConflictError creates a conflict error
func NewConflictError(message string, err error) *CustomError {
	return NewError(ErrTypeConflict, message, err)
}

// IsErrorType checks if an error is of a specific type
func IsErrorType(err error, errType ErrorType) bool {
	if customErr, ok := err.(*CustomError); ok {
		return customErr.Type == errType
	}
	return false
}

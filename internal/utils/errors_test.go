package utils

import (
	"fmt"
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestFormatValidationErrorHandlesWrappedValidationErrors(t *testing.T) {
	type input struct {
		Name string `validate:"required"`
	}

	validate := validator.New()
	err := validate.Struct(input{})
	if err == nil {
		t.Fatal("expected validation error")
	}

	wrappedErr := fmt.Errorf("binding failed: %w", err)
	appErr := FormatValidationError(wrappedErr)

	if appErr.Message != "Validation failed" {
		t.Fatalf("expected message %q, got %q", "Validation failed", appErr.Message)
	}

	if len(appErr.Details) != 1 {
		t.Fatalf("expected 1 detail, got %d", len(appErr.Details))
	}

	if appErr.Details[0] != "Name failed validation: required" {
		t.Fatalf("unexpected detail: %q", appErr.Details[0])
	}
}

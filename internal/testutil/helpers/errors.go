package helpers

import (
	"errors"
	"fmt"
)

// Common test errors
var (
	// ErrTest is a generic test error
	ErrTest = errors.New("test error")

	// ErrNotFound represents a not found error
	ErrNotFound = errors.New("not found")

	// ErrTimeout represents a timeout error
	ErrTimeout = errors.New("timeout")

	// ErrConnection represents a connection error
	ErrConnection = errors.New("connection error")

	// ErrPermission represents a permission error
	ErrPermission = errors.New("permission denied")

	// ErrAPILimit represents an API rate limit error
	ErrAPILimit = errors.New("API rate limit exceeded")
)

// NewTestError creates a test error with a custom message.
func NewTestError(message string) error {
	return fmt.Errorf("test error: %s", message)
}

// NewTestErrorf creates a test error with a formatted message.
func NewTestErrorf(format string, args ...interface{}) error {
	return fmt.Errorf("test error: "+format, args...)
}

// WrapTestError wraps an error with a test error message.
func WrapTestError(err error, message string) error {
	return fmt.Errorf("test error: %s: %w", message, err)
}

// ErrorContains checks if an error contains a specific substring.
// This is useful for testing error messages without exact matching.
func ErrorContains(err error, substr string) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), substr)
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsAt(s, substr)
}

// containsAt checks if s contains substr at any position.
func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ErrorIs checks if an error matches a target error.
// This is a wrapper around errors.Is for consistency.
func ErrorIs(err, target error) bool {
	return errors.Is(err, target)
}

// ErrorAs checks if an error can be assigned to a target type.
// This is a wrapper around errors.As for consistency.
func ErrorAs(err error, target interface{}) bool {
	return errors.As(err, target)
}

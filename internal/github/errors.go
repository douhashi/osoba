package github

import (
	"errors"
	"fmt"
	"time"
)

// GitHubErrorType represents the type of GitHub API error
type GitHubErrorType int

const (
	// ErrorTypeRateLimit indicates rate limit exceeded
	ErrorTypeRateLimit GitHubErrorType = iota
	// ErrorTypeNetworkTimeout indicates network timeout
	ErrorTypeNetworkTimeout
	// ErrorTypeAuthentication indicates authentication failure
	ErrorTypeAuthentication
	// ErrorTypeNotFound indicates resource not found
	ErrorTypeNotFound
	// ErrorTypeServerError indicates server error (5xx)
	ErrorTypeServerError
	// ErrorTypeUnknown indicates unknown error type
	ErrorTypeUnknown
)

// String returns the string representation of the error type
func (t GitHubErrorType) String() string {
	switch t {
	case ErrorTypeRateLimit:
		return "RateLimit"
	case ErrorTypeNetworkTimeout:
		return "NetworkTimeout"
	case ErrorTypeAuthentication:
		return "Authentication"
	case ErrorTypeNotFound:
		return "NotFound"
	case ErrorTypeServerError:
		return "ServerError"
	default:
		return "Unknown"
	}
}

// GitHubError represents a structured GitHub API error
type GitHubError struct {
	Type        GitHubErrorType
	StatusCode  int
	Message     string
	RetryAfter  time.Duration
	OriginalErr error
}

// Error implements the error interface
func (e *GitHubError) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("GitHub API error [%s]: %s (original: %v)", e.Type, e.Message, e.OriginalErr)
	}
	return fmt.Sprintf("GitHub API error [%s]: %s", e.Type, e.Message)
}

// Unwrap returns the original error
func (e *GitHubError) Unwrap() error {
	return e.OriginalErr
}

// IsRetryable returns true if the error is retryable
func (e *GitHubError) IsRetryable() bool {
	switch e.Type {
	case ErrorTypeRateLimit, ErrorTypeNetworkTimeout, ErrorTypeServerError:
		return true
	default:
		return false
	}
}

// IsRateLimitError checks if the error is a rate limit error
func IsRateLimitError(err error) bool {
	var ghErr *GitHubError
	if errors.As(err, &ghErr) {
		return ghErr.Type == ErrorTypeRateLimit
	}
	return false
}

// IsNotFoundError checks if the error is a not found error
func IsNotFoundError(err error) bool {
	var ghErr *GitHubError
	if errors.As(err, &ghErr) {
		return ghErr.Type == ErrorTypeNotFound
	}
	return false
}

// IsAuthenticationError checks if the error is an authentication error
func IsAuthenticationError(err error) bool {
	var ghErr *GitHubError
	if errors.As(err, &ghErr) {
		return ghErr.Type == ErrorTypeAuthentication
	}
	return false
}

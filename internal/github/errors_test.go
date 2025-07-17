package github

import (
	"errors"
	"testing"
	"time"
)

func TestGitHubErrorType_String(t *testing.T) {
	tests := []struct {
		name     string
		errType  GitHubErrorType
		expected string
	}{
		{
			name:     "RateLimit",
			errType:  ErrorTypeRateLimit,
			expected: "RateLimit",
		},
		{
			name:     "NetworkTimeout",
			errType:  ErrorTypeNetworkTimeout,
			expected: "NetworkTimeout",
		},
		{
			name:     "Authentication",
			errType:  ErrorTypeAuthentication,
			expected: "Authentication",
		},
		{
			name:     "NotFound",
			errType:  ErrorTypeNotFound,
			expected: "NotFound",
		},
		{
			name:     "ServerError",
			errType:  ErrorTypeServerError,
			expected: "ServerError",
		},
		{
			name:     "Unknown",
			errType:  ErrorTypeUnknown,
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.errType.String(); got != tt.expected {
				t.Errorf("GitHubErrorType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGitHubError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      GitHubError
		expected string
	}{
		{
			name: "with original error",
			err: GitHubError{
				Type:        ErrorTypeRateLimit,
				StatusCode:  429,
				Message:     "API rate limit exceeded",
				OriginalErr: errors.New("original error"),
			},
			expected: "GitHub API error [RateLimit]: API rate limit exceeded (original: original error)",
		},
		{
			name: "without original error",
			err: GitHubError{
				Type:       ErrorTypeNotFound,
				StatusCode: 404,
				Message:    "Resource not found",
			},
			expected: "GitHub API error [NotFound]: Resource not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("GitHubError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGitHubError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	ghErr := &GitHubError{
		Type:        ErrorTypeServerError,
		StatusCode:  500,
		Message:     "Internal server error",
		OriginalErr: originalErr,
	}

	if got := ghErr.Unwrap(); got != originalErr {
		t.Errorf("GitHubError.Unwrap() = %v, want %v", got, originalErr)
	}
}

func TestGitHubError_IsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      GitHubError
		expected bool
	}{
		{
			name: "rate limit error is retryable",
			err: GitHubError{
				Type:       ErrorTypeRateLimit,
				StatusCode: 429,
			},
			expected: true,
		},
		{
			name: "network timeout is retryable",
			err: GitHubError{
				Type: ErrorTypeNetworkTimeout,
			},
			expected: true,
		},
		{
			name: "server error is retryable",
			err: GitHubError{
				Type:       ErrorTypeServerError,
				StatusCode: 500,
			},
			expected: true,
		},
		{
			name: "not found error is not retryable",
			err: GitHubError{
				Type:       ErrorTypeNotFound,
				StatusCode: 404,
			},
			expected: false,
		},
		{
			name: "authentication error is not retryable",
			err: GitHubError{
				Type:       ErrorTypeAuthentication,
				StatusCode: 401,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.IsRetryable(); got != tt.expected {
				t.Errorf("GitHubError.IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "rate limit error",
			err: &GitHubError{
				Type:       ErrorTypeRateLimit,
				StatusCode: 429,
			},
			expected: true,
		},
		{
			name: "not rate limit error",
			err: &GitHubError{
				Type:       ErrorTypeNotFound,
				StatusCode: 404,
			},
			expected: false,
		},
		{
			name:     "non-GitHubError",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRateLimitError(tt.err); got != tt.expected {
				t.Errorf("IsRateLimitError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "not found error",
			err: &GitHubError{
				Type:       ErrorTypeNotFound,
				StatusCode: 404,
			},
			expected: true,
		},
		{
			name: "not a not found error",
			err: &GitHubError{
				Type:       ErrorTypeRateLimit,
				StatusCode: 429,
			},
			expected: false,
		},
		{
			name:     "non-GitHubError",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFoundError(tt.err); got != tt.expected {
				t.Errorf("IsNotFoundError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsAuthenticationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "authentication error",
			err: &GitHubError{
				Type:       ErrorTypeAuthentication,
				StatusCode: 401,
			},
			expected: true,
		},
		{
			name: "not authentication error",
			err: &GitHubError{
				Type:       ErrorTypeServerError,
				StatusCode: 500,
			},
			expected: false,
		},
		{
			name:     "non-GitHubError",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAuthenticationError(tt.err); got != tt.expected {
				t.Errorf("IsAuthenticationError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGitHubError_RetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		err      GitHubError
		expected time.Duration
	}{
		{
			name: "with retry after",
			err: GitHubError{
				Type:       ErrorTypeRateLimit,
				StatusCode: 429,
				RetryAfter: 60 * time.Second,
			},
			expected: 60 * time.Second,
		},
		{
			name: "without retry after",
			err: GitHubError{
				Type:       ErrorTypeServerError,
				StatusCode: 500,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.RetryAfter; got != tt.expected {
				t.Errorf("GitHubError.RetryAfter = %v, want %v", got, tt.expected)
			}
		})
	}
}

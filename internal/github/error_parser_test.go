package github

import (
	"errors"
	"testing"
	"time"
)

func TestParseGHError(t *testing.T) {
	tests := []struct {
		name           string
		errOutput      string
		err            error
		expectedType   GitHubErrorType
		expectedStatus int
		expectRetry    bool
	}{
		{
			name:           "rate limit error",
			errOutput:      "gh: API rate limit exceeded for user ID 12345",
			err:            errors.New("exit status 1"),
			expectedType:   ErrorTypeRateLimit,
			expectedStatus: 429,
			expectRetry:    false,
		},
		{
			name:           "rate limit with retry after",
			errOutput:      "gh: You have exceeded a secondary rate limit. Retry after: 60",
			err:            errors.New("exit status 1"),
			expectedType:   ErrorTypeRateLimit,
			expectedStatus: 429,
			expectRetry:    true,
		},
		{
			name:           "not found error",
			errOutput:      "gh: could not resolve to a Repository with the name 'owner/repo'",
			err:            errors.New("exit status 1"),
			expectedType:   ErrorTypeNotFound,
			expectedStatus: 404,
			expectRetry:    false,
		},
		{
			name:           "label not found",
			errOutput:      "gh: issue does not have the label 'status:ready'",
			err:            errors.New("exit status 1"),
			expectedType:   ErrorTypeNotFound,
			expectedStatus: 404,
			expectRetry:    false,
		},
		{
			name:           "authentication error",
			errOutput:      "gh: authentication required",
			err:            errors.New("exit status 1"),
			expectedType:   ErrorTypeAuthentication,
			expectedStatus: 401,
			expectRetry:    false,
		},
		{
			name:           "owner is required error",
			errOutput:      "owner is required",
			err:            errors.New("exit status 1"),
			expectedType:   ErrorTypeAuthentication,
			expectedStatus: 401,
			expectRetry:    false,
		},
		{
			name:           "network timeout",
			errOutput:      "dial tcp: lookup api.github.com: i/o timeout",
			err:            errors.New("exit status 1"),
			expectedType:   ErrorTypeNetworkTimeout,
			expectedStatus: 0,
			expectRetry:    false,
		},
		{
			name:           "server error 500",
			errOutput:      "gh: Internal Server Error (HTTP 500)",
			err:            errors.New("exit status 1"),
			expectedType:   ErrorTypeServerError,
			expectedStatus: 500,
			expectRetry:    false,
		},
		{
			name:           "server error 502",
			errOutput:      "gh: Bad Gateway (HTTP 502)",
			err:            errors.New("exit status 1"),
			expectedType:   ErrorTypeServerError,
			expectedStatus: 502,
			expectRetry:    false,
		},
		{
			name:           "unknown error",
			errOutput:      "some random error",
			err:            errors.New("exit status 1"),
			expectedType:   ErrorTypeUnknown,
			expectedStatus: 0,
			expectRetry:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ghErr := ParseGHError(tt.errOutput, tt.err)

			if ghErr.Type != tt.expectedType {
				t.Errorf("ParseGHError() Type = %v, want %v", ghErr.Type, tt.expectedType)
			}

			if ghErr.StatusCode != tt.expectedStatus {
				t.Errorf("ParseGHError() StatusCode = %v, want %v", ghErr.StatusCode, tt.expectedStatus)
			}

			if tt.expectRetry && ghErr.RetryAfter == 0 {
				t.Errorf("ParseGHError() expected RetryAfter to be set")
			}

			if ghErr.OriginalErr != tt.err {
				t.Errorf("ParseGHError() OriginalErr = %v, want %v", ghErr.OriginalErr, tt.err)
			}
		})
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedType GitHubErrorType
	}{
		{
			name:         "nil error",
			err:          nil,
			expectedType: ErrorTypeUnknown,
		},
		{
			name: "already GitHubError",
			err: &GitHubError{
				Type:    ErrorTypeRateLimit,
				Message: "rate limited",
			},
			expectedType: ErrorTypeRateLimit,
		},
		{
			name:         "rate limit in error message",
			err:          errors.New("API rate limit exceeded"),
			expectedType: ErrorTypeRateLimit,
		},
		{
			name:         "not found in error message",
			err:          errors.New("repository not found"),
			expectedType: ErrorTypeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.err)

			if tt.err == nil {
				if result != nil {
					t.Errorf("ClassifyError() = %v, want nil", result)
				}
				return
			}

			ghErr, ok := result.(*GitHubError)
			if !ok && tt.err != nil {
				t.Errorf("ClassifyError() did not return *GitHubError")
				return
			}

			if ghErr != nil && ghErr.Type != tt.expectedType {
				t.Errorf("ClassifyError() Type = %v, want %v", ghErr.Type, tt.expectedType)
			}
		})
	}
}

func TestWrapWithRetryInfo(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		retryAfter    time.Duration
		expectWrapped bool
	}{
		{
			name:          "nil error",
			err:           nil,
			retryAfter:    60 * time.Second,
			expectWrapped: false,
		},
		{
			name:          "wrap plain error",
			err:           errors.New("some error"),
			retryAfter:    30 * time.Second,
			expectWrapped: true,
		},
		{
			name: "wrap existing GitHubError",
			err: &GitHubError{
				Type:    ErrorTypeServerError,
				Message: "server error",
			},
			retryAfter:    45 * time.Second,
			expectWrapped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapWithRetryInfo(tt.err, tt.retryAfter)

			if !tt.expectWrapped {
				if result != nil {
					t.Errorf("WrapWithRetryInfo() = %v, want nil", result)
				}
				return
			}

			ghErr, ok := result.(*GitHubError)
			if !ok {
				t.Errorf("WrapWithRetryInfo() did not return *GitHubError")
				return
			}

			if ghErr.RetryAfter != tt.retryAfter {
				t.Errorf("WrapWithRetryInfo() RetryAfter = %v, want %v", ghErr.RetryAfter, tt.retryAfter)
			}
		})
	}
}

func TestErrorParsing_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		errOutput string
		expected  GitHubErrorType
	}{
		{
			name:      "mixed case rate limit",
			errOutput: "GH: api RATE LIMIT Exceeded",
			expected:  ErrorTypeRateLimit,
		},
		{
			name:      "HTTP status in middle of text",
			errOutput: "Error occurred HTTP 503 Service Unavailable",
			expected:  ErrorTypeServerError,
		},
		{
			name:      "multiple error indicators",
			errOutput: "Authentication failed: rate limit exceeded",
			expected:  ErrorTypeRateLimit, // Rate limit takes precedence
		},
		{
			name:      "empty error output",
			errOutput: "",
			expected:  ErrorTypeUnknown,
		},
		{
			name:      "whitespace only",
			errOutput: "   \n\t   ",
			expected:  ErrorTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ghErr := ParseGHError(tt.errOutput, errors.New("test error"))
			if ghErr.Type != tt.expected {
				t.Errorf("ParseGHError() Type = %v, want %v", ghErr.Type, tt.expected)
			}
		})
	}
}

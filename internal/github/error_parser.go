package github

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// Regular expressions for parsing gh command errors
	rateLimitRegex   = regexp.MustCompile(`(?i)(rate limit|API rate limit exceeded|You have exceeded a secondary rate limit)`)
	notFoundRegex    = regexp.MustCompile(`(?i)(not found|could not resolve to|does not have the label)`)
	authRegex        = regexp.MustCompile(`(?i)(authentication|unauthorized|bad credentials|requires authentication|owner is required)`)
	networkRegex     = regexp.MustCompile(`(?i)(timeout|connection refused|network|dial tcp)`)
	serverErrorRegex = regexp.MustCompile(`(?i)(internal server error|server error|502|503|504)`)
	httpStatusRegex  = regexp.MustCompile(`HTTP (\d{3})`)
	retryAfterRegex  = regexp.MustCompile(`(?i)retry.?after:\s*(\d+)`)
)

// ParseGHError parses error output from gh command and returns a structured GitHubError
func ParseGHError(errOutput string, err error) *GitHubError {
	ghErr := &GitHubError{
		Message:     strings.TrimSpace(errOutput),
		OriginalErr: err,
	}

	// Extract HTTP status code if present
	if matches := httpStatusRegex.FindStringSubmatch(errOutput); len(matches) > 1 {
		if statusCode, err := strconv.Atoi(matches[1]); err == nil {
			ghErr.StatusCode = statusCode
		}
	}

	// Determine error type based on content
	switch {
	case rateLimitRegex.MatchString(errOutput):
		ghErr.Type = ErrorTypeRateLimit
		if ghErr.StatusCode == 0 {
			ghErr.StatusCode = 429
		}
		// Try to extract retry-after duration
		if matches := retryAfterRegex.FindStringSubmatch(errOutput); len(matches) > 1 {
			if seconds, err := strconv.Atoi(matches[1]); err == nil {
				ghErr.RetryAfter = time.Duration(seconds) * time.Second
			}
		}

	case authRegex.MatchString(errOutput):
		ghErr.Type = ErrorTypeAuthentication
		if ghErr.StatusCode == 0 {
			ghErr.StatusCode = 401
		}

	case notFoundRegex.MatchString(errOutput):
		ghErr.Type = ErrorTypeNotFound
		if ghErr.StatusCode == 0 {
			ghErr.StatusCode = 404
		}

	case networkRegex.MatchString(errOutput):
		ghErr.Type = ErrorTypeNetworkTimeout

	case serverErrorRegex.MatchString(errOutput):
		ghErr.Type = ErrorTypeServerError
		if ghErr.StatusCode == 0 && strings.Contains(errOutput, "502") {
			ghErr.StatusCode = 502
		} else if ghErr.StatusCode == 0 && strings.Contains(errOutput, "503") {
			ghErr.StatusCode = 503
		} else if ghErr.StatusCode == 0 && strings.Contains(errOutput, "504") {
			ghErr.StatusCode = 504
		} else if ghErr.StatusCode == 0 {
			ghErr.StatusCode = 500
		}

	default:
		ghErr.Type = ErrorTypeUnknown
		// Check if it's a 5xx error based on status code
		if ghErr.StatusCode >= 500 && ghErr.StatusCode < 600 {
			ghErr.Type = ErrorTypeServerError
		}
	}

	return ghErr
}

// ClassifyError takes an error and attempts to classify it as a GitHubError
func ClassifyError(err error) error {
	if err == nil {
		return nil
	}

	// If it's already a GitHubError, return as is
	var ghErr *GitHubError
	if errors.As(err, &ghErr) {
		return err
	}

	// Try to parse the error message
	errMsg := err.Error()
	return ParseGHError(errMsg, err)
}

// WrapWithRetryInfo wraps an error with retry information
func WrapWithRetryInfo(err error, retryAfter time.Duration) error {
	if err == nil {
		return nil
	}

	ghErr, ok := err.(*GitHubError)
	if !ok {
		// Create a new GitHubError
		ghErr = &GitHubError{
			Type:        ErrorTypeUnknown,
			Message:     err.Error(),
			OriginalErr: err,
		}
	}

	ghErr.RetryAfter = retryAfter
	return ghErr
}

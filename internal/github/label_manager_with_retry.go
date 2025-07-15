package github

import (
	"context"
	"fmt"
	"time"
)

// LabelManagerWithRetry wraps LabelManager with retry functionality
type LabelManagerWithRetry struct {
	*LabelManager
	maxRetries int
	retryDelay time.Duration
}

// NewLabelManagerWithRetry creates a new LabelManagerWithRetry instance
func NewLabelManagerWithRetry(client LabelService, maxRetries int, retryDelay time.Duration) *LabelManagerWithRetry {
	return &LabelManagerWithRetry{
		LabelManager: NewLabelManager(client),
		maxRetries:   maxRetries,
		retryDelay:   retryDelay,
	}
}

// TransitionLabelWithRetry performs label transition with retry logic
func (lm *LabelManagerWithRetry) TransitionLabelWithRetry(ctx context.Context, owner, repo string, issueNumber int) (bool, error) {
	var lastErr error

	for attempt := 0; attempt < lm.maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-time.After(lm.retryDelay * time.Duration(attempt)):
			case <-ctx.Done():
				return false, ctx.Err()
			}
		}

		transitioned, err := lm.TransitionLabel(ctx, owner, repo, issueNumber)
		if err == nil {
			return transitioned, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return false, err
		}
	}

	return false, fmt.Errorf("failed after %d attempts: %w", lm.maxRetries, lastErr)
}

// TransitionLabelWithInfoWithRetry performs label transition with retry logic and returns transition info
func (lm *LabelManagerWithRetry) TransitionLabelWithInfoWithRetry(ctx context.Context, owner, repo string, issueNumber int) (bool, *TransitionInfo, error) {
	var lastErr error
	var lastInfo *TransitionInfo

	for attempt := 0; attempt < lm.maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-time.After(lm.retryDelay * time.Duration(attempt)):
			case <-ctx.Done():
				return false, nil, ctx.Err()
			}
		}

		transitioned, info, err := lm.TransitionLabelWithInfo(ctx, owner, repo, issueNumber)
		if err == nil {
			return transitioned, info, nil
		}

		lastErr = err
		lastInfo = info

		// Check if error is retryable
		if !isRetryableError(err) {
			return false, info, err
		}
	}

	return false, lastInfo, fmt.Errorf("failed after %d attempts: %w", lm.maxRetries, lastErr)
}

// EnsureLabelsExistWithRetry ensures labels exist with retry logic
func (lm *LabelManagerWithRetry) EnsureLabelsExistWithRetry(ctx context.Context, owner, repo string) error {
	var lastErr error

	for attempt := 0; attempt < lm.maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-time.After(lm.retryDelay * time.Duration(attempt)):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err := lm.EnsureLabelsExist(ctx, owner, repo)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return err
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", lm.maxRetries, lastErr)
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	// In a real implementation, we would check for specific error types
	// like rate limit errors, temporary network errors, etc.
	// For now, we'll consider all errors as retryable except for context errors

	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	// Check for specific error messages that indicate non-retryable errors
	errMsg := err.Error()
	nonRetryableMessages := []string{
		"not found",
		"permission denied",
		"unauthorized",
		"forbidden",
		"invalid",
	}

	for _, msg := range nonRetryableMessages {
		if contains(errMsg, msg) {
			return false
		}
	}

	return true
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}

	// Simple case-insensitive contains implementation
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if toLower(s[i+j]) != toLower(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}

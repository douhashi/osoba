package github

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// RetryStrategy defines the retry behavior for GitHub API operations
type RetryStrategy struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       bool
}

// DefaultRetryStrategy returns a default retry strategy
func DefaultRetryStrategy() RetryStrategy {
	return RetryStrategy{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}
}

// RateLimitRetryStrategy returns a retry strategy optimized for rate limits
func RateLimitRetryStrategy() RetryStrategy {
	return RetryStrategy{
		MaxAttempts:  5,
		InitialDelay: 5 * time.Second,
		MaxDelay:     5 * time.Minute,
		Multiplier:   2.0,
		Jitter:       false, // Use exact retry-after values when available
	}
}

// NetworkRetryStrategy returns a retry strategy optimized for network issues
func NetworkRetryStrategy() RetryStrategy {
	return RetryStrategy{
		MaxAttempts:  4,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   1.5,
		Jitter:       true,
	}
}

// GetRetryDelay calculates the delay for a given attempt
func (rs *RetryStrategy) GetRetryDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// Calculate exponential delay
	delay := float64(rs.InitialDelay) * math.Pow(rs.Multiplier, float64(attempt-1))

	// Cap at max delay
	if delay > float64(rs.MaxDelay) {
		delay = float64(rs.MaxDelay)
	}

	// Add jitter if enabled
	if rs.Jitter && delay > 0 {
		// Add up to 25% jitter
		jitter := rand.Float64() * 0.25 * delay
		delay += jitter
	}

	return time.Duration(delay)
}

// ShouldRetry determines if an operation should be retried based on the error
func (rs *RetryStrategy) ShouldRetry(err error, attempt int) bool {
	if err == nil || attempt >= rs.MaxAttempts {
		return false
	}

	// Check if error is retryable
	ghErr, ok := err.(*GitHubError)
	if !ok {
		// For non-GitHubError, retry on a limited basis
		return attempt < 2
	}

	return ghErr.IsRetryable()
}

// GetStrategyForError returns the appropriate retry strategy for a given error
func GetStrategyForError(err error) RetryStrategy {
	ghErr, ok := err.(*GitHubError)
	if !ok {
		return DefaultRetryStrategy()
	}

	switch ghErr.Type {
	case ErrorTypeRateLimit:
		return RateLimitRetryStrategy()
	case ErrorTypeNetworkTimeout:
		return NetworkRetryStrategy()
	case ErrorTypeServerError:
		// Server errors get moderate retry
		return RetryStrategy{
			MaxAttempts:  3,
			InitialDelay: 2 * time.Second,
			MaxDelay:     20 * time.Second,
			Multiplier:   2.0,
			Jitter:       true,
		}
	default:
		return DefaultRetryStrategy()
	}
}

// RetryWithStrategy executes a function with retry logic
func RetryWithStrategy(ctx context.Context, strategy RetryStrategy, operation func() error) error {
	var lastErr error

	for attempt := 1; attempt <= strategy.MaxAttempts; attempt++ {
		// Execute the operation
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !strategy.ShouldRetry(err, attempt) {
			return err
		}

		// Check if context is cancelled
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Don't sleep on the last attempt
		if attempt < strategy.MaxAttempts {
			delay := strategy.GetRetryDelay(attempt)

			// If error has specific retry-after, use it
			if ghErr, ok := err.(*GitHubError); ok && ghErr.RetryAfter > 0 {
				delay = ghErr.RetryAfter
			}

			select {
			case <-time.After(delay):
				// Continue to next attempt
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return lastErr
}

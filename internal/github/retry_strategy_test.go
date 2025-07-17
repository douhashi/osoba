package github

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultRetryStrategy(t *testing.T) {
	rs := DefaultRetryStrategy()

	if rs.MaxAttempts != 3 {
		t.Errorf("DefaultRetryStrategy() MaxAttempts = %v, want 3", rs.MaxAttempts)
	}
	if rs.InitialDelay != 1*time.Second {
		t.Errorf("DefaultRetryStrategy() InitialDelay = %v, want 1s", rs.InitialDelay)
	}
	if rs.MaxDelay != 30*time.Second {
		t.Errorf("DefaultRetryStrategy() MaxDelay = %v, want 30s", rs.MaxDelay)
	}
	if rs.Multiplier != 2.0 {
		t.Errorf("DefaultRetryStrategy() Multiplier = %v, want 2.0", rs.Multiplier)
	}
	if !rs.Jitter {
		t.Errorf("DefaultRetryStrategy() Jitter = false, want true")
	}
}

func TestRateLimitRetryStrategy(t *testing.T) {
	rs := RateLimitRetryStrategy()

	if rs.MaxAttempts != 5 {
		t.Errorf("RateLimitRetryStrategy() MaxAttempts = %v, want 5", rs.MaxAttempts)
	}
	if rs.InitialDelay != 5*time.Second {
		t.Errorf("RateLimitRetryStrategy() InitialDelay = %v, want 5s", rs.InitialDelay)
	}
	if rs.MaxDelay != 5*time.Minute {
		t.Errorf("RateLimitRetryStrategy() MaxDelay = %v, want 5m", rs.MaxDelay)
	}
	if rs.Jitter {
		t.Errorf("RateLimitRetryStrategy() Jitter = true, want false")
	}
}

func TestNetworkRetryStrategy(t *testing.T) {
	rs := NetworkRetryStrategy()

	if rs.MaxAttempts != 4 {
		t.Errorf("NetworkRetryStrategy() MaxAttempts = %v, want 4", rs.MaxAttempts)
	}
	if rs.InitialDelay != 500*time.Millisecond {
		t.Errorf("NetworkRetryStrategy() InitialDelay = %v, want 500ms", rs.InitialDelay)
	}
	if rs.MaxDelay != 10*time.Second {
		t.Errorf("NetworkRetryStrategy() MaxDelay = %v, want 10s", rs.MaxDelay)
	}
	if rs.Multiplier != 1.5 {
		t.Errorf("NetworkRetryStrategy() Multiplier = %v, want 1.5", rs.Multiplier)
	}
}

func TestRetryStrategy_GetRetryDelay(t *testing.T) {
	tests := []struct {
		name     string
		strategy RetryStrategy
		attempt  int
		minDelay time.Duration
		maxDelay time.Duration
	}{
		{
			name: "first attempt without jitter",
			strategy: RetryStrategy{
				InitialDelay: 1 * time.Second,
				MaxDelay:     10 * time.Second,
				Multiplier:   2.0,
				Jitter:       false,
			},
			attempt:  1,
			minDelay: 1 * time.Second,
			maxDelay: 1 * time.Second,
		},
		{
			name: "second attempt without jitter",
			strategy: RetryStrategy{
				InitialDelay: 1 * time.Second,
				MaxDelay:     10 * time.Second,
				Multiplier:   2.0,
				Jitter:       false,
			},
			attempt:  2,
			minDelay: 2 * time.Second,
			maxDelay: 2 * time.Second,
		},
		{
			name: "third attempt without jitter",
			strategy: RetryStrategy{
				InitialDelay: 1 * time.Second,
				MaxDelay:     10 * time.Second,
				Multiplier:   2.0,
				Jitter:       false,
			},
			attempt:  3,
			minDelay: 4 * time.Second,
			maxDelay: 4 * time.Second,
		},
		{
			name: "with jitter",
			strategy: RetryStrategy{
				InitialDelay: 1 * time.Second,
				MaxDelay:     10 * time.Second,
				Multiplier:   2.0,
				Jitter:       true,
			},
			attempt:  2,
			minDelay: 2 * time.Second,
			maxDelay: 2500 * time.Millisecond, // 2s + 25% jitter
		},
		{
			name: "capped at max delay",
			strategy: RetryStrategy{
				InitialDelay: 1 * time.Second,
				MaxDelay:     5 * time.Second,
				Multiplier:   10.0,
				Jitter:       false,
			},
			attempt:  3,
			minDelay: 5 * time.Second,
			maxDelay: 5 * time.Second,
		},
		{
			name: "zero attempt",
			strategy: RetryStrategy{
				InitialDelay: 1 * time.Second,
				MaxDelay:     10 * time.Second,
				Multiplier:   2.0,
				Jitter:       false,
			},
			attempt:  0,
			minDelay: 0,
			maxDelay: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := tt.strategy.GetRetryDelay(tt.attempt)
			if delay < tt.minDelay || delay > tt.maxDelay {
				t.Errorf("GetRetryDelay() = %v, want between %v and %v", delay, tt.minDelay, tt.maxDelay)
			}
		})
	}
}

func TestRetryStrategy_ShouldRetry(t *testing.T) {
	tests := []struct {
		name     string
		strategy RetryStrategy
		err      error
		attempt  int
		expected bool
	}{
		{
			name:     "nil error",
			strategy: DefaultRetryStrategy(),
			err:      nil,
			attempt:  1,
			expected: false,
		},
		{
			name:     "exceeded max attempts",
			strategy: RetryStrategy{MaxAttempts: 3},
			err:      errors.New("some error"),
			attempt:  3,
			expected: false,
		},
		{
			name:     "retryable GitHub error",
			strategy: DefaultRetryStrategy(),
			err: &GitHubError{
				Type: ErrorTypeRateLimit,
			},
			attempt:  1,
			expected: true,
		},
		{
			name:     "non-retryable GitHub error",
			strategy: DefaultRetryStrategy(),
			err: &GitHubError{
				Type: ErrorTypeNotFound,
			},
			attempt:  1,
			expected: false,
		},
		{
			name:     "non-GitHub error first attempt",
			strategy: DefaultRetryStrategy(),
			err:      errors.New("generic error"),
			attempt:  1,
			expected: true,
		},
		{
			name:     "non-GitHub error second attempt",
			strategy: DefaultRetryStrategy(),
			err:      errors.New("generic error"),
			attempt:  2,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.strategy.ShouldRetry(tt.err, tt.attempt); got != tt.expected {
				t.Errorf("ShouldRetry() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetStrategyForError(t *testing.T) {
	tests := []struct {
		name                 string
		err                  error
		expectedMaxAttempts  int
		expectedInitialDelay time.Duration
	}{
		{
			name:                 "rate limit error",
			err:                  &GitHubError{Type: ErrorTypeRateLimit},
			expectedMaxAttempts:  5,
			expectedInitialDelay: 5 * time.Second,
		},
		{
			name:                 "network timeout error",
			err:                  &GitHubError{Type: ErrorTypeNetworkTimeout},
			expectedMaxAttempts:  4,
			expectedInitialDelay: 500 * time.Millisecond,
		},
		{
			name:                 "server error",
			err:                  &GitHubError{Type: ErrorTypeServerError},
			expectedMaxAttempts:  3,
			expectedInitialDelay: 2 * time.Second,
		},
		{
			name:                 "unknown error",
			err:                  &GitHubError{Type: ErrorTypeUnknown},
			expectedMaxAttempts:  3,
			expectedInitialDelay: 1 * time.Second,
		},
		{
			name:                 "non-GitHub error",
			err:                  errors.New("generic error"),
			expectedMaxAttempts:  3,
			expectedInitialDelay: 1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := GetStrategyForError(tt.err)
			if strategy.MaxAttempts != tt.expectedMaxAttempts {
				t.Errorf("GetStrategyForError() MaxAttempts = %v, want %v", strategy.MaxAttempts, tt.expectedMaxAttempts)
			}
			if strategy.InitialDelay != tt.expectedInitialDelay {
				t.Errorf("GetStrategyForError() InitialDelay = %v, want %v", strategy.InitialDelay, tt.expectedInitialDelay)
			}
		})
	}
}

func TestRetryWithStrategy(t *testing.T) {
	tests := []struct {
		name           string
		strategy       RetryStrategy
		expectedCalls  int
		expectError    bool
		setupOperation func() (func() error, func() int)
	}{
		{
			name: "success on first attempt",
			strategy: RetryStrategy{
				MaxAttempts:  3,
				InitialDelay: 10 * time.Millisecond,
			},
			expectedCalls: 1,
			expectError:   false,
			setupOperation: func() (func() error, func() int) {
				count := 0
				return func() error {
					count++
					if count == 1 {
						return nil
					}
					return errors.New("should not reach here")
				}, func() int { return count }
			},
		},
		{
			name: "success on second attempt",
			strategy: RetryStrategy{
				MaxAttempts:  3,
				InitialDelay: 10 * time.Millisecond,
			},
			expectedCalls: 2,
			expectError:   false,
			setupOperation: func() (func() error, func() int) {
				count := 0
				return func() error {
					count++
					if count == 2 {
						return nil
					}
					return &GitHubError{Type: ErrorTypeServerError}
				}, func() int { return count }
			},
		},
		{
			name: "all attempts fail",
			strategy: RetryStrategy{
				MaxAttempts:  3,
				InitialDelay: 10 * time.Millisecond,
			},
			expectedCalls: 3,
			expectError:   true,
			setupOperation: func() (func() error, func() int) {
				count := 0
				return func() error {
					count++
					return &GitHubError{Type: ErrorTypeServerError}
				}, func() int { return count }
			},
		},
		{
			name: "non-retryable error",
			strategy: RetryStrategy{
				MaxAttempts:  3,
				InitialDelay: 10 * time.Millisecond,
			},
			expectedCalls: 1,
			expectError:   true,
			setupOperation: func() (func() error, func() int) {
				count := 0
				return func() error {
					count++
					return &GitHubError{Type: ErrorTypeNotFound}
				}, func() int { return count }
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			operation, getCount := tt.setupOperation()

			err := RetryWithStrategy(ctx, tt.strategy, operation)

			// Get actual call count
			callCount := getCount()

			if (err != nil) != tt.expectError {
				t.Errorf("RetryWithStrategy() error = %v, expectError = %v", err, tt.expectError)
			}

			if callCount != tt.expectedCalls {
				t.Errorf("RetryWithStrategy() callCount = %v, want %v", callCount, tt.expectedCalls)
			}
		})
	}
}

func TestRetryWithStrategy_ContextCancellation(t *testing.T) {
	strategy := RetryStrategy{
		MaxAttempts:  5,
		InitialDelay: 1 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	operation := func() error {
		callCount++
		if callCount == 2 {
			cancel() // Cancel context on second attempt
		}
		return &GitHubError{Type: ErrorTypeServerError}
	}

	err := RetryWithStrategy(ctx, strategy, operation)

	if err != context.Canceled {
		t.Errorf("RetryWithStrategy() with cancelled context = %v, want context.Canceled", err)
	}

	// Should have attempted twice before cancellation
	if callCount != 2 {
		t.Errorf("RetryWithStrategy() callCount = %v, want 2", callCount)
	}
}

func TestRetryWithStrategy_RespectRetryAfter(t *testing.T) {
	strategy := RetryStrategy{
		MaxAttempts:  2,
		InitialDelay: 100 * time.Millisecond,
	}

	retryAfter := 50 * time.Millisecond
	callCount := 0
	start := time.Now()

	operation := func() error {
		callCount++
		if callCount == 1 {
			return &GitHubError{
				Type:       ErrorTypeRateLimit,
				RetryAfter: retryAfter,
			}
		}
		return nil
	}

	err := RetryWithStrategy(context.Background(), strategy, operation)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("RetryWithStrategy() error = %v, want nil", err)
	}

	// Should have waited at least retryAfter duration
	if elapsed < retryAfter {
		t.Errorf("RetryWithStrategy() elapsed = %v, want at least %v", elapsed, retryAfter)
	}

	// But not too much longer (allowing some overhead)
	if elapsed > retryAfter+20*time.Millisecond {
		t.Errorf("RetryWithStrategy() elapsed = %v, want close to %v", elapsed, retryAfter)
	}
}

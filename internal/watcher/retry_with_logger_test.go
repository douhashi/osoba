package watcher

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
)

// TestRetryWithBackoffLogger tests the new RetryWithBackoffLogger function
func TestRetryWithBackoffLogger(t *testing.T) {
	tests := []struct {
		name         string
		maxRetries   int
		operation    func() error
		wantErr      bool
		wantAttempts int
		checkLogs    func(t *testing.T, logs []MockLogEntry)
	}{
		{
			name:       "正常系: 初回成功",
			maxRetries: 3,
			operation: func() error {
				return nil
			},
			wantErr:      false,
			wantAttempts: 1,
			checkLogs: func(t *testing.T, logs []MockLogEntry) {
				// 成功時はログなし
				if len(logs) != 0 {
					t.Errorf("Expected no logs, got %d logs", len(logs))
				}
			},
		},
		{
			name:       "正常系: 2回目で成功",
			maxRetries: 3,
			operation: func() func() error {
				var attempt int32
				return func() error {
					currentAttempt := atomic.AddInt32(&attempt, 1)
					if currentAttempt < 2 {
						return &github.RateLimitError{
							Message: "API rate limit exceeded",
							Rate: github.RateLimit{
								Reset: time.Now().Add(time.Second),
							},
						}
					}
					return nil
				}
			}(),
			wantErr:      false,
			wantAttempts: 2,
			checkLogs: func(t *testing.T, logs []MockLogEntry) {
				// リトライログとレート制限ログがあるはず
				if len(logs) < 2 {
					t.Errorf("Expected at least 2 logs, got %d", len(logs))
					return
				}

				// ログレベルを確認
				hasRetryLog := false
				hasRateLimitLog := false
				for _, log := range logs {
					// RetryWithBackoffLoggerではINFOレベルで出力される
					if log.Level == "INFO" && log.Message == "Retrying operation" {
						hasRetryLog = true
					}
					if log.Level == "WARN" && log.Message == "Rate limit hit, waiting until reset" {
						hasRateLimitLog = true
					}
				}

				if !hasRetryLog {
					t.Error("Expected retry log but not found")
				}
				if !hasRateLimitLog {
					t.Error("Expected rate limit warning log but not found")
				}
			},
		},
		{
			name:       "異常系: 最大リトライ回数超過",
			maxRetries: 2,
			operation: func() error {
				return &github.ErrorResponse{
					Message: "Service Unavailable",
				}
			},
			wantErr:      true,
			wantAttempts: 2,
			checkLogs: func(t *testing.T, logs []MockLogEntry) {
				// 各リトライのログとエラーログがあるはず
				infoLogs := 0
				debugLogs := 0
				errorLogs := 0
				for _, log := range logs {
					switch log.Level {
					case "INFO":
						infoLogs++
					case "DEBUG":
						debugLogs++
					case "ERROR":
						errorLogs++
					}
				}

				// Retrying operationのINFOログと、IsRetryableErrorのDEBUGログ
				if infoLogs < 1 {
					t.Errorf("Expected at least 1 INFO log, got %d", infoLogs)
				}
				if debugLogs < 1 {
					t.Errorf("Expected at least 1 DEBUG log, got %d", debugLogs)
				}
				if errorLogs != 1 {
					t.Errorf("Expected 1 ERROR log, got %d", errorLogs)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 各テストケースで新しいモックロガーを使用
			testLogger := NewMockLogger()

			attempts := 0
			countingOperation := func() error {
				attempts++
				return tt.operation()
			}

			err := RetryWithBackoffLogger(context.Background(), testLogger, tt.maxRetries, 10*time.Millisecond, countingOperation)
			if (err != nil) != tt.wantErr {
				t.Errorf("RetryWithBackoffLogger() error = %v, wantErr %v", err, tt.wantErr)
			}

			if attempts != tt.wantAttempts {
				t.Errorf("RetryWithBackoffLogger() attempts = %v, want %v", attempts, tt.wantAttempts)
			}

			// ログの検証
			if mockLog, ok := testLogger.(*mockLogger); ok {
				tt.checkLogs(t, mockLog.GetLogs())
			}
		})
	}
}

// TestIsRetryableErrorLogger tests the new IsRetryableErrorLogger function
func TestIsRetryableErrorLogger(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		want      bool
		checkLogs func(t *testing.T, logs []MockLogEntry)
	}{
		{
			name: "GitHub APIレート制限エラー",
			err: &github.RateLimitError{
				Message: "API rate limit exceeded",
			},
			want: true,
			checkLogs: func(t *testing.T, logs []MockLogEntry) {
				if len(logs) != 1 {
					t.Errorf("Expected 1 log, got %d", len(logs))
					return
				}
				if logs[0].Level != "DEBUG" {
					t.Errorf("Expected DEBUG log, got %s", logs[0].Level)
				}
				if logs[0].Message != "Error is retryable" {
					t.Errorf("Expected retryable message, got %s", logs[0].Message)
				}
			},
		},
		{
			name: "リトライ不可能なエラー",
			err:  errors.New("not found"),
			want: false,
			checkLogs: func(t *testing.T, logs []MockLogEntry) {
				if len(logs) != 1 {
					t.Errorf("Expected 1 log, got %d", len(logs))
					return
				}
				if logs[0].Level != "DEBUG" {
					t.Errorf("Expected DEBUG log, got %s", logs[0].Level)
				}
				if logs[0].Message != "Error is not retryable" {
					t.Errorf("Expected not retryable message, got %s", logs[0].Message)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLogger := NewMockLogger()

			got := IsRetryableErrorLogger(testLogger, tt.err)
			if got != tt.want {
				t.Errorf("IsRetryableErrorLogger() = %v, want %v", got, tt.want)
			}

			// ログの検証
			if mockLog, ok := testLogger.(*mockLogger); ok {
				tt.checkLogs(t, mockLog.GetLogs())
			}
		})
	}
}

// TestMigrationCompatibility tests that old function calls still work
func TestMigrationCompatibility(t *testing.T) {
	t.Run("RetryWithBackoff互換性", func(t *testing.T) {
		var attempts int32
		operation := func() error {
			currentAttempt := atomic.AddInt32(&attempts, 1)
			if currentAttempt < 2 {
				return &github.RateLimitError{
					Message: "API rate limit exceeded",
					Rate: github.RateLimit{
						Reset: time.Now().Add(100 * time.Millisecond),
					},
				}
			}
			return nil
		}

		// 既存の関数が引き続き動作することを確認
		err := RetryWithBackoff(context.Background(), 3, 10*time.Millisecond, operation)
		if err != nil {
			t.Errorf("RetryWithBackoff() returned unexpected error: %v", err)
		}

		finalAttempts := atomic.LoadInt32(&attempts)
		if finalAttempts != 2 {
			t.Errorf("Expected 2 attempts, got %d", finalAttempts)
		}
	})

	t.Run("IsRetryableError互換性", func(t *testing.T) {
		err := &github.RateLimitError{Message: "rate limit"}

		// 既存の関数が引き続き動作することを確認
		if !IsRetryableError(err) {
			t.Error("IsRetryableError() should return true for rate limit error")
		}
	})
}

// TestLoggerIntegration tests integration with real logger
func TestLoggerIntegration(t *testing.T) {
	// 実際のロガー実装を使用したテスト
	realLogger, err := logger.New(logger.WithLevel("debug"))
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	var attempts int32
	operation := func() error {
		currentAttempt := atomic.AddInt32(&attempts, 1)
		if currentAttempt < 3 {
			return &github.ErrorResponse{
				Message: "Internal Server Error",
			}
		}
		return nil
	}

	err = RetryWithBackoffLogger(context.Background(), realLogger, 5, 10*time.Millisecond, operation)
	if err != nil {
		t.Errorf("RetryWithBackoffLogger() returned unexpected error: %v", err)
	}

	finalAttempts := atomic.LoadInt32(&attempts)
	if finalAttempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", finalAttempts)
	}
}

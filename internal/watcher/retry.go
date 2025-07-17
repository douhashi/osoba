package watcher

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
)

// defaultLogger はグローバル関数用のデフォルトロガー
var defaultLogger logger.Logger

// init はパッケージ初期化時にデフォルトロガーを設定
func init() {
	// エラーハンドリングを簡略化するため、エラーが発生した場合はnilロガーを使用
	l, err := logger.New(logger.WithLevel("info"))
	if err != nil {
		// エラーが発生した場合は標準ログに出力
		log.Printf("Failed to initialize default logger: %v", err)
		// nilチェックを避けるため、モックロガーを使用
		defaultLogger = &mockLogger{}
	} else {
		defaultLogger = l
	}
}

// SetDefaultLogger はテスト用にデフォルトロガーを設定する
func SetDefaultLogger(l logger.Logger) {
	defaultLogger = l
}

// stdLogCompatLogger は標準ログとの互換性を提供するロガー
type stdLogCompatLogger struct{}

func (s *stdLogCompatLogger) Debug(msg string, keysAndValues ...interface{}) {
	// RetryWithBackoffは標準ログを使用していたが、デバッグログは出力していなかった
}

func (s *stdLogCompatLogger) Info(msg string, keysAndValues ...interface{}) {
	// 既存の動作と同じように標準ログ形式で出力
	if msg == "Retrying operation" {
		// 既存のフォーマットを再現
		var attempt, maxRetries int
		var backoff time.Duration
		var errMsg string
		for i := 0; i < len(keysAndValues)-1; i += 2 {
			key, ok := keysAndValues[i].(string)
			if !ok {
				continue
			}
			switch key {
			case "attempt":
				attempt, _ = keysAndValues[i+1].(int)
			case "maxRetries":
				maxRetries, _ = keysAndValues[i+1].(int)
			case "backoff":
				backoff, _ = keysAndValues[i+1].(time.Duration)
			case "error":
				errMsg, _ = keysAndValues[i+1].(string)
			}
		}
		log.Printf("Retrying after %v (attempt %d/%d): %s", backoff, attempt, maxRetries, errMsg)
	}
}

func (s *stdLogCompatLogger) Warn(msg string, keysAndValues ...interface{}) {
	if msg == "Rate limit hit, waiting until reset" {
		// 既存のフォーマットを再現
		var waitDuration time.Duration
		for i := 0; i < len(keysAndValues)-1; i += 2 {
			key, ok := keysAndValues[i].(string)
			if ok && key == "waitDuration" {
				waitDuration, _ = keysAndValues[i+1].(time.Duration)
				break
			}
		}
		log.Printf("Rate limit hit, waiting until reset: %v", waitDuration)
	}
}

func (s *stdLogCompatLogger) Error(msg string, keysAndValues ...interface{}) {
	// RetryWithBackoffは最終的なエラーをログ出力していなかったため、何もしない
}

func (s *stdLogCompatLogger) WithFields(keysAndValues ...interface{}) logger.Logger {
	return s
}

// RetryWithBackoff は指数バックオフでリトライを実行する
// Deprecated: Use RetryWithBackoffLogger instead. This function will be removed in a future version.
func RetryWithBackoff(ctx context.Context, maxRetries int, baseDelay time.Duration, operation func() error) error {
	// 互換性のため、標準ログ出力を使用する特別なロガーを作成
	compatLogger := &stdLogCompatLogger{}
	return RetryWithBackoffLogger(ctx, compatLogger, maxRetries, baseDelay, operation)
}

// RetryWithBackoffLogger は指数バックオフでリトライを実行する（logger付き）
func RetryWithBackoffLogger(ctx context.Context, logger logger.Logger, maxRetries int, baseDelay time.Duration, operation func() error) error {
	if maxRetries <= 0 {
		maxRetries = 1
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		// コンテキストがキャンセルされているか確認
		select {
		case <-ctx.Done():
			return fmt.Errorf("operation cancelled: %w", ctx.Err())
		default:
		}

		// 操作を実行
		err := operation()
		if err == nil {
			return nil
		}
		lastErr = err

		// リトライ可能なエラーかチェック
		if !IsRetryableErrorLogger(logger, err) {
			return err
		}

		// 最後の試行の場合はリトライしない
		if attempt == maxRetries-1 {
			break
		}

		// バックオフ時間を計算
		backoff := CalculateBackoffLogger(logger, attempt+1, baseDelay)
		errMsg := "unknown error"
		if err != nil {
			// エラーメッセージを安全に取得
			func() {
				defer func() {
					if r := recover(); r != nil {
						errMsg = fmt.Sprintf("error getting error message: %v", r)
					}
				}()
				errMsg = err.Error()
			}()
		}
		// 互換性のために、Retrying operationはInfoレベルで出力
		logger.Info("Retrying operation",
			"attempt", attempt+1,
			"maxRetries", maxRetries,
			"backoff", backoff,
			"error", errMsg)

		// レート制限エラーの場合は特別な処理
		if sleepDuration, ok := HandleRateLimitErrorLogger(logger, err); ok && sleepDuration > 0 {
			backoff = sleepDuration
			logger.Warn("Rate limit hit, waiting until reset",
				"waitDuration", backoff)
		}

		// バックオフ時間待機
		select {
		case <-time.After(backoff):
			// 次の試行へ
		case <-ctx.Done():
			return fmt.Errorf("operation cancelled during backoff: %w", ctx.Err())
		}
	}

	logger.Error("Max retries exceeded",
		"maxRetries", maxRetries,
		"error", lastErr)
	return fmt.Errorf("max retries (%d) exceeded: %w", maxRetries, lastErr)
}

// IsRetryableError はエラーがリトライ可能かどうかを判定する
// Deprecated: Use IsRetryableErrorLogger instead. This function will be removed in a future version.
func IsRetryableError(err error) bool {
	return IsRetryableErrorLogger(defaultLogger, err)
}

// IsRetryableErrorLogger はエラーがリトライ可能かどうかを判定する（logger付き）
func IsRetryableErrorLogger(logger logger.Logger, err error) bool {
	if err == nil {
		return false
	}

	// GitHub APIのレート制限エラー
	var rateLimitErr *github.RateLimitError
	if errors.As(err, &rateLimitErr) {
		logger.Debug("Error is retryable",
			"errorType", "RateLimitError",
			"error", err)
		return true
	}

	// GitHub APIのエラーレスポンス
	var errResp *github.ErrorResponse
	if errors.As(err, &errResp) {
		// エラーメッセージでサーバーエラーやレート制限を判定
		msg := strings.ToLower(errResp.Message)
		if strings.Contains(msg, "server error") || strings.Contains(msg, "internal server error") || strings.Contains(msg, "service unavailable") || strings.Contains(msg, "bad gateway") {
			logger.Debug("Error is retryable",
				"errorType", "ServerError",
				"error", err)
			return true
		}
		if strings.Contains(msg, "rate limit") || strings.Contains(msg, "too many requests") {
			logger.Debug("Error is retryable",
				"errorType", "RateLimit",
				"error", err)
			return true
		}
	}

	// ネットワークタイムアウトエラー
	errStr := err.Error()
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "Client.Timeout exceeded") {
		logger.Debug("Error is retryable",
			"errorType", "Timeout",
			"error", err)
		return true
	}

	// 一時的なネットワークエラー
	if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "no such host") {
		logger.Debug("Error is retryable",
			"errorType", "NetworkError",
			"error", err)
		return true
	}

	logger.Debug("Error is not retryable",
		"error", err)
	return false
}

// CalculateBackoff は指数バックオフの遅延時間を計算する
// Deprecated: Use CalculateBackoffLogger instead. This function will be removed in a future version.
func CalculateBackoff(attempt int, baseDelay time.Duration) time.Duration {
	return CalculateBackoffLogger(defaultLogger, attempt, baseDelay)
}

// CalculateBackoffLogger は指数バックオフの遅延時間を計算する（logger付き）
func CalculateBackoffLogger(logger logger.Logger, attempt int, baseDelay time.Duration) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}

	// 指数バックオフ: baseDelay * 2^(attempt-1)
	delay := float64(baseDelay) * math.Pow(2, float64(attempt-1))

	// ジッターを追加（±20%）
	jitter := delay * 0.2 * (rand.Float64()*2 - 1)
	delay += jitter

	// 最大遅延時間を1分に制限
	maxDelay := float64(60 * time.Second)
	if delay > maxDelay {
		delay = maxDelay
		logger.Debug("Backoff delay capped at max",
			"maxDelay", time.Duration(maxDelay),
			"attempt", attempt)
	} else {
		logger.Debug("Calculated backoff delay",
			"delay", time.Duration(delay),
			"attempt", attempt,
			"baseDelay", baseDelay)
	}

	return time.Duration(delay)
}

// HandleRateLimitError はGitHub APIのレート制限エラーを処理する
// Deprecated: Use HandleRateLimitErrorLogger instead. This function will be removed in a future version.
func HandleRateLimitError(err error) (time.Duration, bool) {
	return HandleRateLimitErrorLogger(defaultLogger, err)
}

// HandleRateLimitErrorLogger はGitHub APIのレート制限エラーを処理する（logger付き）
func HandleRateLimitErrorLogger(logger logger.Logger, err error) (time.Duration, bool) {
	var rateLimitErr *github.RateLimitError
	if !errors.As(err, &rateLimitErr) {
		return 0, false
	}

	// リセット時刻が設定されている場合
	resetTime := rateLimitErr.Rate.Reset
	if !resetTime.IsZero() {
		sleepDuration := time.Until(resetTime)
		if sleepDuration > 0 {
			// 少し余裕を持たせる（1秒追加）
			sleepDuration += time.Second
			logger.Debug("Rate limit reset time calculated",
				"resetTime", resetTime,
				"sleepDuration", sleepDuration)
			return sleepDuration, true
		}
	}

	logger.Debug("Rate limit error has no valid reset time",
		"error", err)
	return 0, false
}

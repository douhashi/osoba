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
)

// RetryWithBackoff は指数バックオフでリトライを実行する
func RetryWithBackoff(ctx context.Context, maxRetries int, baseDelay time.Duration, operation func() error) error {
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
		if !IsRetryableError(err) {
			return err
		}

		// 最後の試行の場合はリトライしない
		if attempt == maxRetries-1 {
			break
		}

		// バックオフ時間を計算
		backoff := CalculateBackoff(attempt+1, baseDelay)
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
		log.Printf("Retrying after %v (attempt %d/%d): %s", backoff, attempt+1, maxRetries, errMsg)

		// レート制限エラーの場合は特別な処理
		if sleepDuration, ok := HandleRateLimitError(err); ok && sleepDuration > 0 {
			backoff = sleepDuration
			log.Printf("Rate limit hit, waiting until reset: %v", backoff)
		}

		// バックオフ時間待機
		select {
		case <-time.After(backoff):
			// 次の試行へ
		case <-ctx.Done():
			return fmt.Errorf("operation cancelled during backoff: %w", ctx.Err())
		}
	}

	return fmt.Errorf("max retries (%d) exceeded: %w", maxRetries, lastErr)
}

// IsRetryableError はエラーがリトライ可能かどうかを判定する
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// GitHub APIのレート制限エラー
	var rateLimitErr *github.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return true
	}

	// GitHub APIのエラーレスポンス
	var errResp *github.ErrorResponse
	if errors.As(err, &errResp) {
		// エラーメッセージでサーバーエラーやレート制限を判定
		msg := strings.ToLower(errResp.Message)
		if strings.Contains(msg, "server error") || strings.Contains(msg, "internal server error") || strings.Contains(msg, "service unavailable") || strings.Contains(msg, "bad gateway") {
			return true
		}
		if strings.Contains(msg, "rate limit") || strings.Contains(msg, "too many requests") {
			return true
		}
	}

	// ネットワークタイムアウトエラー
	errStr := err.Error()
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "Client.Timeout exceeded") {
		return true
	}

	// 一時的なネットワークエラー
	if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "no such host") {
		return true
	}

	return false
}

// CalculateBackoff は指数バックオフの遅延時間を計算する
func CalculateBackoff(attempt int, baseDelay time.Duration) time.Duration {
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
	}

	return time.Duration(delay)
}

// HandleRateLimitError はGitHub APIのレート制限エラーを処理する
func HandleRateLimitError(err error) (time.Duration, bool) {
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
			return sleepDuration, true
		}
	}

	return 0, false
}

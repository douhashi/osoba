package watcher

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-github/v50/github"
)

func TestRetryWithBackoff(t *testing.T) {
	tests := []struct {
		name         string
		maxRetries   int
		operation    func() error
		wantErr      bool
		wantAttempts int
	}{
		{
			name:       "正常系: 初回成功",
			maxRetries: 3,
			operation: func() error {
				return nil
			},
			wantErr:      false,
			wantAttempts: 1,
		},
		{
			name:       "正常系: 2回目で成功",
			maxRetries: 3,
			operation: func() func() error {
				attempt := 0
				return func() error {
					attempt++
					if attempt < 2 {
						// リトライ可能なエラーを返す
						return &github.RateLimitError{
							Message: "API rate limit exceeded",
							Rate: github.Rate{
								Reset: github.Timestamp{Time: time.Now().Add(time.Second)},
							},
						}
					}
					return nil
				}
			}(),
			wantErr:      false,
			wantAttempts: 2,
		},
		{
			name:       "異常系: 最大リトライ回数超過",
			maxRetries: 3,
			operation: func() error {
				// リトライ可能なエラーを返し続ける
				return &github.ErrorResponse{
					Response: &http.Response{
						StatusCode: 503,
						Status:     "503 Service Unavailable",
						Request: &http.Request{
							Method: "GET",
							URL:    &url.URL{Scheme: "https", Host: "api.github.com", Path: "/test"},
						},
					},
					Message: "Service Unavailable",
				}
			},
			wantErr:      true,
			wantAttempts: 3,
		},
		{
			name:       "異常系: maxRetries が 0",
			maxRetries: 0,
			operation: func() error {
				return errors.New("error")
			},
			wantErr:      true,
			wantAttempts: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := 0
			countingOperation := func() error {
				attempts++
				return tt.operation()
			}

			err := RetryWithBackoff(context.Background(), tt.maxRetries, 10*time.Millisecond, countingOperation)
			if (err != nil) != tt.wantErr {
				t.Errorf("RetryWithBackoff() error = %v, wantErr %v", err, tt.wantErr)
			}

			if attempts != tt.wantAttempts {
				t.Errorf("RetryWithBackoff() attempts = %v, want %v", attempts, tt.wantAttempts)
			}
		})
	}
}

func TestRetryWithBackoff_ContextCancellation(t *testing.T) {
	t.Run("コンテキストキャンセルで即座に終了", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // 即座にキャンセル

		attempts := 0
		operation := func() error {
			attempts++
			return errors.New("should not retry")
		}

		err := RetryWithBackoff(ctx, 5, 10*time.Millisecond, operation)
		if err == nil {
			t.Error("RetryWithBackoff() should return error on cancelled context")
		}

		if attempts > 1 {
			t.Errorf("RetryWithBackoff() attempts = %v, want <= 1", attempts)
		}
	})
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "GitHub APIレート制限エラー",
			err: &github.RateLimitError{
				Message: "API rate limit exceeded",
			},
			want: true,
		},
		{
			name: "GitHub APIレスポンスエラー（5xx）",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: 503,
					Status:     "503 Service Unavailable",
					Request: &http.Request{
						Method: "GET",
						URL:    &url.URL{Scheme: "https", Host: "api.github.com", Path: "/test"},
					},
				},
				Message: "Service Unavailable",
			},
			want: true,
		},
		{
			name: "GitHub APIレスポンスエラー（4xx）",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: 404,
					Status:     "404 Not Found",
					Request: &http.Request{
						Method: "GET",
						URL:    &url.URL{Scheme: "https", Host: "api.github.com", Path: "/test"},
					},
				},
				Message: "Not Found",
			},
			want: false,
		},
		{
			name: "ネットワークタイムアウトエラー",
			err:  errors.New("net/http: request canceled (Client.Timeout exceeded)"),
			want: true,
		},
		{
			name: "一般的なエラー",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "nilエラー",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryableError(tt.err)
			if got != tt.want {
				t.Errorf("IsRetryableError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		name      string
		attempt   int
		baseDelay time.Duration
		wantMin   time.Duration
		wantMax   time.Duration
	}{
		{
			name:      "1回目のリトライ",
			attempt:   1,
			baseDelay: time.Second,
			wantMin:   800 * time.Millisecond,  // 1秒 - 20%
			wantMax:   1200 * time.Millisecond, // 1秒 + 20%
		},
		{
			name:      "2回目のリトライ",
			attempt:   2,
			baseDelay: time.Second,
			wantMin:   1600 * time.Millisecond, // 2秒 - 20%
			wantMax:   2400 * time.Millisecond, // 2秒 + 20%
		},
		{
			name:      "3回目のリトライ",
			attempt:   3,
			baseDelay: time.Second,
			wantMin:   3200 * time.Millisecond, // 4秒 - 20%
			wantMax:   4800 * time.Millisecond, // 4秒 + 20%
		},
		{
			name:      "最大遅延時間の制限",
			attempt:   10,
			baseDelay: time.Second,
			wantMin:   60 * time.Second, // 最大1分
			wantMax:   60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateBackoff(tt.attempt, tt.baseDelay)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("CalculateBackoff() = %v, want between %v and %v", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestHandleRateLimitError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantSleep time.Duration
		wantOk    bool
	}{
		{
			name: "レート制限エラーでリセット時刻あり",
			err: &github.RateLimitError{
				Rate: github.Rate{
					Reset: github.Timestamp{Time: time.Now().Add(5 * time.Second)},
				},
			},
			wantSleep: 5 * time.Second,
			wantOk:    true,
		},
		{
			name: "レート制限エラーでリセット時刻なし",
			err: &github.RateLimitError{
				Rate: github.Rate{},
			},
			wantSleep: 0,
			wantOk:    false,
		},
		{
			name:      "レート制限エラー以外",
			err:       errors.New("other error"),
			wantSleep: 0,
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSleep, gotOk := HandleRateLimitError(tt.err)

			// 時間の差が1秒以内であることを確認（時間計算の誤差を考慮）
			if tt.wantOk && gotOk {
				diff := gotSleep - tt.wantSleep
				if diff < -time.Second || diff > time.Second {
					t.Errorf("HandleRateLimitError() sleep = %v, want approximately %v", gotSleep, tt.wantSleep)
				}
			} else if gotSleep != tt.wantSleep {
				t.Errorf("HandleRateLimitError() sleep = %v, want %v", gotSleep, tt.wantSleep)
			}

			if gotOk != tt.wantOk {
				t.Errorf("HandleRateLimitError() ok = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

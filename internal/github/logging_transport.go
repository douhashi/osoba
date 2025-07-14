package github

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/douhashi/osoba/internal/logger"
)

// loggingRoundTripper はHTTPリクエスト/レスポンスをログ出力するラウンドトリッパー
type loggingRoundTripper struct {
	base   http.RoundTripper
	logger logger.Logger
}

// RoundTrip はHTTPリクエストを実行し、リクエスト/レスポンスの詳細をログ出力する
func (rt *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	// リクエストをクローンして、元のリクエストを変更しない
	clonedReq := req.Clone(req.Context())

	// リクエストログ
	rt.logRequest(clonedReq)

	// 実際のリクエスト実行
	resp, err := rt.base.RoundTrip(req)

	duration := time.Since(start)

	if err != nil {
		// エラーログ
		rt.logger.Error("github_api_error",
			"method", req.Method,
			"url", req.URL.String(),
			"duration_ms", duration.Milliseconds(),
			"error", err.Error(),
		)
		return nil, err
	}

	// レスポンスログ
	rt.logResponse(resp, duration)

	return resp, nil
}

// logRequest はHTTPリクエストの詳細をログ出力する
func (rt *loggingRoundTripper) logRequest(req *http.Request) {
	fields := []interface{}{
		"method", req.Method,
		"url", req.URL.String(),
	}

	// 全てのヘッダーをチェック（デバッグ用）
	for key, values := range req.Header {
		if key == "Authorization" {
			// Authorizationヘッダーをマスキング
			if len(values) > 0 && values[0] != "" {
				fields = append(fields, "authorization", rt.maskAuthHeader(values[0]))
			}
		}
	}

	// User-Agentヘッダー
	if ua := req.Header.Get("User-Agent"); ua != "" {
		fields = append(fields, "user_agent", ua)
	}

	rt.logger.Debug("github_api_request", fields...)
}

// logResponse はHTTPレスポンスの詳細をログ出力する
func (rt *loggingRoundTripper) logResponse(resp *http.Response, duration time.Duration) {
	fields := []interface{}{
		"status_code", resp.StatusCode,
		"duration_ms", duration.Milliseconds(),
	}

	// レート制限情報
	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		fields = append(fields, "rate_limit_remaining", remaining)
	}
	if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
		fields = append(fields, "rate_limit_reset", reset)
	}

	// レスポンスボディをプレビュー
	if resp.Body != nil {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			rt.logger.Error("failed_to_read_response_body", "error", err.Error())
		} else {
			// ボディを再設定
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			// プレビュー用にボディの一部を追加
			preview := string(bodyBytes)
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			fields = append(fields, "body_preview", preview)
		}
	}

	rt.logger.Debug("github_api_response", fields...)
}

// maskAuthHeader はAuthorizationヘッダーの値をマスキングする
func (rt *loggingRoundTripper) maskAuthHeader(auth string) string {
	if auth == "" {
		return ""
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) == 2 {
		return fmt.Sprintf("%s [REDACTED]", parts[0])
	}
	return "[REDACTED]"
}

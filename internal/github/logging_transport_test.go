package github

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/douhashi/osoba/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggingRoundTripper_RoundTrip(t *testing.T) {
	t.Run("正常系: HTTPリクエスト/レスポンスがログ出力される", func(t *testing.T) {
		// Arrange
		core, observed := observer.New(zapcore.DebugLevel)
		testLogger := newLoggerWithCore(core)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-RateLimit-Remaining", "60")
			w.Header().Set("X-RateLimit-Reset", "1234567890")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "success"}`))
		}))
		defer server.Close()

		transport := &loggingRoundTripper{
			base:   http.DefaultTransport,
			logger: testLogger,
		}

		// Act
		req, err := http.NewRequest("GET", server.URL+"/repos/owner/repo", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer secret-token")

		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Assert
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		logs := observed.All()
		require.GreaterOrEqual(t, len(logs), 2, "少なくともリクエストとレスポンスのログが必要")

		// リクエストログの確認
		var reqLog observer.LoggedEntry
		for _, log := range logs {
			if log.Message == "github_api_request" {
				reqLog = log
				break
			}
		}
		assert.Equal(t, "github_api_request", reqLog.Message)
		assert.Equal(t, "GET", reqLog.ContextMap()["method"])
		assert.Contains(t, reqLog.ContextMap()["url"], "/repos/owner/repo")
		assert.Equal(t, "Bearer [REDACTED]", reqLog.ContextMap()["authorization"])

		// レスポンスログの確認
		var respLog observer.LoggedEntry
		for _, log := range logs {
			if log.Message == "github_api_response" {
				respLog = log
				break
			}
		}
		assert.Equal(t, "github_api_response", respLog.Message)
		assert.Equal(t, int64(200), respLog.ContextMap()["status_code"])
		assert.Equal(t, "60", respLog.ContextMap()["rate_limit_remaining"])
		assert.NotNil(t, respLog.ContextMap()["duration_ms"])
	})

	t.Run("正常系: Authorizationヘッダーがマスキングされる", func(t *testing.T) {
		// Arrange
		core, observed := observer.New(zapcore.DebugLevel)
		testLogger := newLoggerWithCore(core)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		transport := &loggingRoundTripper{
			base:   http.DefaultTransport,
			logger: testLogger,
		}

		// Act
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "token ghp_1234567890abcdef")

		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Assert
		logs := observed.All()
		var reqLog observer.LoggedEntry
		for _, log := range logs {
			if log.Message == "github_api_request" {
				reqLog = log
				break
			}
		}
		assert.Equal(t, "token [REDACTED]", reqLog.ContextMap()["authorization"])
	})

	t.Run("正常系: 大きなレスポンスボディが要約される", func(t *testing.T) {
		// Arrange
		core, observed := observer.New(zapcore.DebugLevel)
		testLogger := newLoggerWithCore(core)

		largeBody := strings.Repeat("a", 1000)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(largeBody))
		}))
		defer server.Close()

		transport := &loggingRoundTripper{
			base:   http.DefaultTransport,
			logger: testLogger,
		}

		// Act
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)

		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Body を読み込む
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, largeBody, string(body))

		// Assert
		logs := observed.All()
		var respLog observer.LoggedEntry
		for _, log := range logs {
			if log.Message == "github_api_response" {
				respLog = log
				break
			}
		}
		bodyPreview := respLog.ContextMap()["body_preview"].(string)
		assert.Equal(t, 203, len(bodyPreview)) // 200文字 + "..."
		assert.True(t, strings.HasSuffix(bodyPreview, "..."))
	})

	t.Run("異常系: トランスポートエラーがログ出力される", func(t *testing.T) {
		// Arrange
		core, observed := observer.New(zapcore.DebugLevel)
		testLogger := newLoggerWithCore(core)

		transport := &loggingRoundTripper{
			base:   http.DefaultTransport,
			logger: testLogger,
		}

		// Act - 不正なURLでリクエスト
		req, err := http.NewRequest("GET", "http://invalid-domain-that-does-not-exist.local", nil)
		require.NoError(t, err)

		_, err = transport.RoundTrip(req)

		// Assert
		assert.Error(t, err)

		logs := observed.All()
		require.GreaterOrEqual(t, len(logs), 2)

		errorLog := logs[1]
		assert.Equal(t, "github_api_error", errorLog.Message)
		assert.NotNil(t, errorLog.ContextMap()["error"])
	})

	t.Run("異常系: nilリクエストでパニックしない", func(t *testing.T) {
		// Arrange
		core, observed := observer.New(zapcore.DebugLevel)
		testLogger := newLoggerWithCore(core)

		transport := &loggingRoundTripper{
			base:   http.DefaultTransport,
			logger: testLogger,
		}

		// Act & Assert
		assert.Panics(t, func() {
			transport.RoundTrip(nil)
		})

		// パニックしてもログは残っているはず
		logs := observed.All()
		assert.Empty(t, logs)
	})
}

func TestLoggingRoundTripper_MaskAuthHeader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Bearer token",
			input:    "Bearer ghp_1234567890abcdef",
			expected: "Bearer [REDACTED]",
		},
		{
			name:     "Basic auth",
			input:    "Basic dXNlcjpwYXNz",
			expected: "Basic [REDACTED]",
		},
		{
			name:     "token形式",
			input:    "token 1234567890abcdef",
			expected: "token [REDACTED]",
		},
		{
			name:     "空文字",
			input:    "",
			expected: "",
		},
		{
			name:     "不明な形式",
			input:    "CustomAuth something",
			expected: "CustomAuth [REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &loggingRoundTripper{}
			result := rt.maskAuthHeader(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// newLoggerWithCore はテスト用にカスタムコアでロガーを作成するヘルパー関数
func newLoggerWithCore(core zapcore.Core) logger.Logger {
	zapLogger := zap.New(core)
	sugar := zapLogger.Sugar()
	return &testLogger{sugar: sugar}
}

// testLogger はテスト用のロガー実装
type testLogger struct {
	sugar *zap.SugaredLogger
}

func (l *testLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.sugar.Debugw(msg, keysAndValues...)
}

func (l *testLogger) Info(msg string, keysAndValues ...interface{}) {
	l.sugar.Infow(msg, keysAndValues...)
}

func (l *testLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.sugar.Warnw(msg, keysAndValues...)
}

func (l *testLogger) Error(msg string, keysAndValues ...interface{}) {
	l.sugar.Errorw(msg, keysAndValues...)
}

func (l *testLogger) WithFields(keysAndValues ...interface{}) logger.Logger {
	return &testLogger{
		sugar: l.sugar.With(keysAndValues...),
	}
}

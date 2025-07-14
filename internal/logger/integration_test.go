package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestIntegration_LogLevelFiltering(t *testing.T) {
	tests := []struct {
		name        string
		level       string
		shouldDebug bool
		shouldInfo  bool
		shouldWarn  bool
		shouldError bool
	}{
		{
			name:        "debugレベル：すべて出力",
			level:       "debug",
			shouldDebug: true,
			shouldInfo:  true,
			shouldWarn:  true,
			shouldError: true,
		},
		{
			name:        "infoレベル：debug以外出力",
			level:       "info",
			shouldDebug: false,
			shouldInfo:  true,
			shouldWarn:  true,
			shouldError: true,
		},
		{
			name:        "warnレベル：warnとerrorのみ出力",
			level:       "warn",
			shouldDebug: false,
			shouldInfo:  false,
			shouldWarn:  true,
			shouldError: true,
		},
		{
			name:        "errorレベル：errorのみ出力",
			level:       "error",
			shouldDebug: false,
			shouldInfo:  false,
			shouldWarn:  false,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// カスタムコアでロガーを作成
			var buf bytes.Buffer
			encoderConfig := zapcore.EncoderConfig{
				MessageKey:     "msg",
				LevelKey:       "level",
				EncodeLevel:    zapcore.LowercaseLevelEncoder,
				EncodeDuration: zapcore.SecondsDurationEncoder,
				LineEnding:     zapcore.DefaultLineEnding,
			}

			encoder := zapcore.NewJSONEncoder(encoderConfig)
			level, _ := parseLevel(tt.level)
			core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), level)

			logger := newLoggerWithCore(core)

			// 各レベルでログ出力
			logger.Debug("debug message")
			logger.Info("info message")
			logger.Warn("warn message")
			logger.Error("error message")

			// 出力を検証
			output := buf.String()
			lines := strings.Split(strings.TrimSpace(output), "\n")

			// 期待される出力行数を計算
			expectedLines := 0
			if tt.shouldDebug {
				expectedLines++
			}
			if tt.shouldInfo {
				expectedLines++
			}
			if tt.shouldWarn {
				expectedLines++
			}
			if tt.shouldError {
				expectedLines++
			}

			if expectedLines == 0 {
				assert.Empty(t, strings.TrimSpace(output))
			} else {
				assert.Len(t, lines, expectedLines)

				// 各行がJSONとして有効か確認
				for _, line := range lines {
					var logEntry map[string]interface{}
					err := json.Unmarshal([]byte(line), &logEntry)
					require.NoError(t, err)
				}
			}
		})
	}
}

func TestIntegration_StructuredLogging(t *testing.T) {
	// カスタムバッファでロガーを作成
	var buf bytes.Buffer
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	encoder := zapcore.NewJSONEncoder(encoderConfig)
	level, _ := parseLevel("info")
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), level)

	logger := newLoggerWithCore(core)

	logger.Info("user action",
		"user_id", "12345",
		"action", "login",
		"success", true,
		"duration_ms", 150,
	)

	// JSON出力を検証
	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "info", logEntry["level"])
	assert.Equal(t, "user action", logEntry["msg"])
	assert.Equal(t, "12345", logEntry["user_id"])
	assert.Equal(t, "login", logEntry["action"])
	assert.Equal(t, true, logEntry["success"])
	assert.Equal(t, float64(150), logEntry["duration_ms"])
}

func TestIntegration_WithFields(t *testing.T) {
	// カスタムバッファでロガーを作成
	var buf bytes.Buffer
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	encoder := zapcore.NewJSONEncoder(encoderConfig)
	level, _ := parseLevel("info")
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), level)

	logger := newLoggerWithCore(core)

	// フィールドを追加したロガーを作成
	serviceLogger := logger.WithFields(
		"service", "auth",
		"version", "1.2.3",
	)

	serviceLogger.Info("service started")

	// JSON出力を検証
	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "auth", logEntry["service"])
	assert.Equal(t, "1.2.3", logEntry["version"])
	assert.Equal(t, "service started", logEntry["msg"])
}

func TestIntegration_ConcurrentLogging(t *testing.T) {
	logger, err := New(WithLevel("info"))
	require.NoError(t, err)

	// 並行してログ出力してもパニックしないことを確認
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				logger.Info("concurrent log", "goroutine", id, "iteration", j)
			}
			done <- true
		}(i)
	}

	// すべてのgoroutineが完了するまで待つ
	for i := 0; i < 10; i++ {
		<-done
	}
}

// ベンチマークテスト
func BenchmarkLogger_Info(b *testing.B) {
	logger, _ := New(WithLevel("info"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "iteration", i)
	}
}

func BenchmarkLogger_InfoWithFields(b *testing.B) {
	logger, _ := New(WithLevel("info"))
	serviceLogger := logger.WithFields("service", "benchmark", "version", "1.0.0")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		serviceLogger.Info("benchmark message", "iteration", i, "data", "test")
	}
}

func BenchmarkLogger_DebugFiltered(b *testing.B) {
	// infoレベルでdebugメッセージはフィルタされる
	logger, _ := New(WithLevel("info"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Debug("filtered message", "iteration", i)
	}
}

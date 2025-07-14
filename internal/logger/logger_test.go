package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("デフォルト設定でロガーを作成できる", func(t *testing.T) {
		logger, err := New()
		require.NoError(t, err)
		assert.NotNil(t, logger)
	})
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name  string
		level string
		test  func(t *testing.T, logger Logger)
	}{
		{
			name:  "Debugレベルでログ出力できる",
			level: "debug",
			test: func(t *testing.T, logger Logger) {
				assert.NotPanics(t, func() {
					logger.Debug("debug message", "key", "value")
				})
			},
		},
		{
			name:  "Infoレベルでログ出力できる",
			level: "info",
			test: func(t *testing.T, logger Logger) {
				assert.NotPanics(t, func() {
					logger.Info("info message", "key", "value")
				})
			},
		},
		{
			name:  "Warnレベルでログ出力できる",
			level: "warn",
			test: func(t *testing.T, logger Logger) {
				assert.NotPanics(t, func() {
					logger.Warn("warn message", "key", "value")
				})
			},
		},
		{
			name:  "Errorレベルでログ出力できる",
			level: "error",
			test: func(t *testing.T, logger Logger) {
				assert.NotPanics(t, func() {
					logger.Error("error message", "key", "value")
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(WithLevel(tt.level))
			require.NoError(t, err)
			tt.test(t, logger)
		})
	}
}

func TestWithOptions(t *testing.T) {
	t.Run("ログレベルをオプションで設定できる", func(t *testing.T) {
		logger, err := New(WithLevel("debug"))
		require.NoError(t, err)
		assert.NotNil(t, logger)
	})

	t.Run("フォーマットをJSONに設定できる", func(t *testing.T) {
		logger, err := New(WithFormat("json"))
		require.NoError(t, err)
		assert.NotNil(t, logger)
	})

	t.Run("フォーマットをテキストに設定できる", func(t *testing.T) {
		logger, err := New(WithFormat("text"))
		require.NoError(t, err)
		assert.NotNil(t, logger)
	})

	t.Run("複数のオプションを同時に設定できる", func(t *testing.T) {
		logger, err := New(
			WithLevel("debug"),
			WithFormat("json"),
		)
		require.NoError(t, err)
		assert.NotNil(t, logger)
	})
}

func TestWithFields(t *testing.T) {
	t.Run("フィールドを追加したロガーを作成できる", func(t *testing.T) {
		logger, err := New()
		require.NoError(t, err)

		childLogger := logger.WithFields("component", "test", "version", "1.0.0")
		assert.NotNil(t, childLogger)

		assert.NotPanics(t, func() {
			childLogger.Info("message with fields")
		})
	})
}

func TestInvalidOptions(t *testing.T) {
	t.Run("無効なログレベルを指定するとエラーになる", func(t *testing.T) {
		_, err := New(WithLevel("invalid"))
		assert.Error(t, err)
	})

	t.Run("無効なフォーマットを指定するとエラーになる", func(t *testing.T) {
		_, err := New(WithFormat("invalid"))
		assert.Error(t, err)
	})
}

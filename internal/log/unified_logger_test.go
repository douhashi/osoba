package log

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnifiedLogger_Basic(t *testing.T) {
	t.Run("デフォルト設定でロガーを作成できる", func(t *testing.T) {
		logger := New()
		assert.NotNil(t, logger)
	})

	t.Run("コンポーネント名を指定してロガーを作成できる", func(t *testing.T) {
		logger := New(WithComponent("test"))
		assert.NotNil(t, logger)
	})
}

func TestUnifiedLogger_LogLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := New(WithOutput(&buf), WithLevel(DebugLevel))

	t.Run("Debugレベルでログ出力できる", func(t *testing.T) {
		buf.Reset()
		logger.Debug("debug message")
		assert.Contains(t, buf.String(), "debug message")
	})

	t.Run("Infoレベルでログ出力できる", func(t *testing.T) {
		buf.Reset()
		logger.Info("info message")
		assert.Contains(t, buf.String(), "info message")
	})

	t.Run("Warnレベルでログ出力できる", func(t *testing.T) {
		buf.Reset()
		logger.Warn("warn message")
		assert.Contains(t, buf.String(), "warn message")
	})

	t.Run("Errorレベルでログ出力できる", func(t *testing.T) {
		buf.Reset()
		logger.Error("error message")
		assert.Contains(t, buf.String(), "error message")
	})
}

func TestUnifiedLogger_ColorOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := New(WithOutput(&buf), WithColorEnabled(true), WithLevel(DebugLevel))

	t.Run("DEBUGレベルはグレーで出力される", func(t *testing.T) {
		buf.Reset()
		logger.Debug("debug message")
		output := buf.String()
		// ANSIカラーコードが含まれることを確認
		assert.Contains(t, output, "\x1b[37m") // グレー
		assert.Contains(t, output, "DEBUG")
	})

	t.Run("INFOレベルは青で出力される", func(t *testing.T) {
		buf.Reset()
		logger.Info("info message")
		output := buf.String()
		assert.Contains(t, output, "\x1b[34m") // 青
		assert.Contains(t, output, "INFO")
	})

	t.Run("WARNレベルは黄で出力される", func(t *testing.T) {
		buf.Reset()
		logger.Warn("warn message")
		output := buf.String()
		assert.Contains(t, output, "\x1b[33m") // 黄
		assert.Contains(t, output, "WARN")
	})

	t.Run("ERRORレベルは赤で出力される", func(t *testing.T) {
		buf.Reset()
		logger.Error("error message")
		output := buf.String()
		assert.Contains(t, output, "\x1b[31m") // 赤
		assert.Contains(t, output, "ERROR")
	})
}

func TestUnifiedLogger_Format(t *testing.T) {
	var buf bytes.Buffer
	logger := New(WithOutput(&buf), WithComponent("test-component"))

	t.Run("統一されたログフォーマットで出力される", func(t *testing.T) {
		buf.Reset()
		logger.Info("test message")
		output := buf.String()

		// タイムスタンプが含まれることを確認
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`, output)

		// レベルが含まれることを確認
		assert.Contains(t, output, "INFO")

		// コンポーネントが含まれることを確認
		assert.Contains(t, output, "test-component")

		// メッセージが含まれることを確認
		assert.Contains(t, output, "test message")
	})
}

func TestUnifiedLogger_StructuredLog(t *testing.T) {
	var buf bytes.Buffer
	logger := New(WithOutput(&buf), WithFormat(JSONFormat))

	t.Run("JSON形式でログ出力できる", func(t *testing.T) {
		buf.Reset()
		logger.Info("test message")
		output := buf.String()

		// JSON形式の基本構造を確認
		assert.Contains(t, output, `"level":"INFO"`)
		assert.Contains(t, output, `"msg":"test message"`)
		assert.Contains(t, output, `"time":`)
	})

	t.Run("追加フィールドをJSON形式で出力できる", func(t *testing.T) {
		buf.Reset()
		logger.WithField("key", "value").Info("test message")
		output := buf.String()

		assert.Contains(t, output, `"key":"value"`)
		assert.Contains(t, output, `"msg":"test message"`)
	})
}

func TestUnifiedLogger_FileOutput(t *testing.T) {
	// 一時ファイルを作成
	tmpFile, err := os.CreateTemp("", "test-log-*.log")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	t.Run("ファイルにログ出力できる", func(t *testing.T) {
		logger := New(WithFileOutput(tmpFile.Name()))
		logger.Info("test message")

		// ファイルから内容を読み取り
		content, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)

		assert.Contains(t, string(content), "test message")
		assert.Contains(t, string(content), "INFO")
	})
}

func TestUnifiedLogger_LogLevel(t *testing.T) {
	var buf bytes.Buffer

	tests := []struct {
		name       string
		level      Level
		logFunc    func(Logger)
		shouldShow bool
	}{
		{
			name:  "DEBUGレベル設定時はすべてのレベルが出力される",
			level: DebugLevel,
			logFunc: func(l Logger) {
				l.Debug("debug")
				l.Info("info")
				l.Warn("warn")
				l.Error("error")
			},
			shouldShow: true,
		},
		{
			name:  "INFOレベル設定時はDEBUGが出力されない",
			level: InfoLevel,
			logFunc: func(l Logger) {
				l.Debug("debug")
			},
			shouldShow: false,
		},
		{
			name:  "WARNレベル設定時はDEBUGとINFOが出力されない",
			level: WarnLevel,
			logFunc: func(l Logger) {
				l.Info("info")
			},
			shouldShow: false,
		},
		{
			name:  "ERRORレベル設定時はERRORのみ出力される",
			level: ErrorLevel,
			logFunc: func(l Logger) {
				l.Warn("warn")
			},
			shouldShow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			logger := New(WithOutput(&buf), WithLevel(tt.level))
			tt.logFunc(logger)

			if tt.shouldShow {
				assert.NotEmpty(t, buf.String())
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}

func TestUnifiedLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(WithOutput(&buf))

	t.Run("WithFieldでフィールドを追加できる", func(t *testing.T) {
		buf.Reset()
		logger.WithField("key", "value").Info("test message")
		output := buf.String()

		assert.Contains(t, output, "key=value")
		assert.Contains(t, output, "test message")
	})

	t.Run("WithFieldsで複数フィールドを追加できる", func(t *testing.T) {
		buf.Reset()
		logger.WithFields(Fields{
			"key1": "value1",
			"key2": "value2",
		}).Info("test message")
		output := buf.String()

		assert.Contains(t, output, "key1=value1")
		assert.Contains(t, output, "key2=value2")
		assert.Contains(t, output, "test message")
	})

	t.Run("WithComponentでコンポーネントを設定できる", func(t *testing.T) {
		buf.Reset()
		logger.WithComponent("new-component").Info("test message")
		output := buf.String()

		assert.Contains(t, output, "new-component")
		assert.Contains(t, output, "test message")
	})
}

func TestUnifiedLogger_ColorDisabled(t *testing.T) {
	var buf bytes.Buffer
	logger := New(WithOutput(&buf), WithColorEnabled(false))

	t.Run("色分けが無効化されている場合はカラーコードが出力されない", func(t *testing.T) {
		buf.Reset()
		logger.Info("test message")
		output := buf.String()

		// ANSIカラーコードが含まれないことを確認
		assert.NotContains(t, output, "\x1b[")
		assert.Contains(t, output, "INFO")
		assert.Contains(t, output, "test message")
	})
}

func TestUnifiedLogger_LogLevelFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Level
		hasError bool
	}{
		{
			name:     "debug文字列からDebugLevelに変換できる",
			input:    "debug",
			expected: DebugLevel,
			hasError: false,
		},
		{
			name:     "info文字列からInfoLevelに変換できる",
			input:    "info",
			expected: InfoLevel,
			hasError: false,
		},
		{
			name:     "warn文字列からWarnLevelに変換できる",
			input:    "warn",
			expected: WarnLevel,
			hasError: false,
		},
		{
			name:     "error文字列からErrorLevelに変換できる",
			input:    "error",
			expected: ErrorLevel,
			hasError: false,
		},
		{
			name:     "大文字でも変換できる",
			input:    "INFO",
			expected: InfoLevel,
			hasError: false,
		},
		{
			name:     "無効な文字列の場合はエラーになる",
			input:    "invalid",
			expected: InfoLevel,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, err := ParseLevel(tt.input)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, level)
			}
		})
	}
}

func TestUnifiedLogger_Integration(t *testing.T) {
	t.Run("複数のオプションを組み合わせて使用できる", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(
			WithOutput(&buf),
			WithLevel(DebugLevel),
			WithComponent("integration-test"),
			WithColorEnabled(true),
			WithFormat(TextFormat),
		)

		logger.WithField("test", "value").Info("integration test message")
		output := buf.String()

		assert.Contains(t, output, "INFO")
		assert.Contains(t, output, "integration-test")
		assert.Contains(t, output, "test=value")
		assert.Contains(t, output, "integration test message")
		assert.Contains(t, output, "\x1b[34m") // 青色のカラーコード
	})
}

func TestUnifiedLogger_Performance(t *testing.T) {
	t.Run("大量のログ出力でも性能が劣化しない", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(WithOutput(&buf))

		start := time.Now()
		for i := 0; i < 1000; i++ {
			logger.Info(fmt.Sprintf("message %d", i))
		}
		duration := time.Since(start)

		// 1000件のログ出力が1秒以内に完了することを確認
		assert.Less(t, duration, time.Second)

		// 出力された行数を確認
		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		assert.Len(t, lines, 1000)
	})
}

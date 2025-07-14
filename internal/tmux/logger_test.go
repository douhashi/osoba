package tmux

import (
	"testing"

	"github.com/douhashi/osoba/internal/logger"
)

func TestSetLogger(t *testing.T) {
	t.Run("ロガーを設定できる", func(t *testing.T) {
		// Arrange
		mockLogger := &mockLogger{}

		// Act
		SetLogger(mockLogger)

		// Assert
		if pkg.logger != mockLogger {
			t.Error("ロガーが正しく設定されていません")
		}
	})

	t.Run("nilロガーを設定してもパニックしない", func(t *testing.T) {
		// Act & Assert - パニックしないことを確認
		SetLogger(nil)

		if pkg.logger != nil {
			t.Error("nilロガーが設定されていません")
		}
	})
}

func TestGetLogger(t *testing.T) {
	t.Run("設定されたロガーを取得できる", func(t *testing.T) {
		// Arrange
		mockLogger := &mockLogger{}
		SetLogger(mockLogger)

		// Act
		result := GetLogger()

		// Assert
		if result != mockLogger {
			t.Error("正しいロガーが取得できませんでした")
		}
	})

	t.Run("ロガーが設定されていない場合はnilを返す", func(t *testing.T) {
		// Arrange
		SetLogger(nil)

		// Act
		result := GetLogger()

		// Assert
		if result != nil {
			t.Error("nilが返されませんでした")
		}
	})
}

// mockLogger はテスト用のモックロガー
type mockLogger struct {
	debugMessages []string
	infoMessages  []string
	warnMessages  []string
	errorMessages []string
}

func (m *mockLogger) Debug(msg string, keysAndValues ...interface{}) {
	m.debugMessages = append(m.debugMessages, msg)
}

func (m *mockLogger) Info(msg string, keysAndValues ...interface{}) {
	m.infoMessages = append(m.infoMessages, msg)
}

func (m *mockLogger) Warn(msg string, keysAndValues ...interface{}) {
	m.warnMessages = append(m.warnMessages, msg)
}

func (m *mockLogger) Error(msg string, keysAndValues ...interface{}) {
	m.errorMessages = append(m.errorMessages, msg)
}

func (m *mockLogger) WithFields(keysAndValues ...interface{}) logger.Logger {
	return m
}

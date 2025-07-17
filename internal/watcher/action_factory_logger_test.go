package watcher

import (
	"bytes"
	"testing"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// createTestLogger はテスト用のloggerを作成し、出力をキャプチャするバッファも返す
func createTestLogger() (logger.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
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
	core := zapcore.NewCore(encoder, zapcore.AddSync(buf), zapcore.DebugLevel)

	zapLogger := zap.New(core, zap.AddCallerSkip(1))
	sugar := zapLogger.Sugar()

	return &testZapLogger{sugar: sugar}, buf
}

// testZapLogger は内部パッケージからexportされていないため、テスト用に再定義
type testZapLogger struct {
	sugar *zap.SugaredLogger
}

func (l *testZapLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.sugar.Debugw(msg, keysAndValues...)
}

func (l *testZapLogger) Info(msg string, keysAndValues ...interface{}) {
	l.sugar.Infow(msg, keysAndValues...)
}

func (l *testZapLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.sugar.Warnw(msg, keysAndValues...)
}

func (l *testZapLogger) Error(msg string, keysAndValues ...interface{}) {
	l.sugar.Errorw(msg, keysAndValues...)
}

func (l *testZapLogger) WithFields(keysAndValues ...interface{}) logger.Logger {
	return &testZapLogger{
		sugar: l.sugar.With(keysAndValues...),
	}
}

func TestActionFactoryWithLogger(t *testing.T) {
	t.Run("ActionFactoryにloggerを注入できること", func(t *testing.T) {
		// Arrange
		testLogger, buf := createTestLogger()
		sessionName := "test-session"
		ghClient := &github.Client{}
		worktreeManager := &MockWorktreeManager{}
		claudeExecutor := claude.NewClaudeExecutor()
		claudeConfig := &claude.ClaudeConfig{}

		// Act
		factory := NewDefaultActionFactoryWithLogger(
			sessionName,
			ghClient,
			worktreeManager,
			claudeExecutor,
			claudeConfig,
			config.NewConfig(),
			"test-owner",
			"test-repo",
			testLogger,
		)

		// Assert
		assert.NotNil(t, factory)
		assert.NotNil(t, factory.logger)

		// バッファが空であることを確認（まだログ出力がないため）
		assert.Empty(t, buf.String())
	})

	t.Run("PlanActionにloggerが注入されること", func(t *testing.T) {
		// Arrange
		testLogger, buf := createTestLogger()
		factory := &DefaultActionFactory{
			sessionName:     "test-session",
			ghClient:        &github.Client{},
			worktreeManager: &MockWorktreeManager{},
			claudeExecutor:  claude.NewClaudeExecutor(),
			claudeConfig:    &claude.ClaudeConfig{},
			stateManager:    NewIssueStateManager(),
			logger:          testLogger,
		}

		// Act
		action := factory.CreatePlanAction()

		// Assert
		assert.NotNil(t, action)

		// actionにloggerが設定されていることを確認
		// 実際のログ出力テストは個別のActionのテストで行う
		assert.Empty(t, buf.String()) // まだ実行していないためログは空
	})

	t.Run("ImplementationActionにloggerが注入されること", func(t *testing.T) {
		// Arrange
		testLogger, buf := createTestLogger()
		factory := &DefaultActionFactory{
			sessionName:     "test-session",
			ghClient:        &github.Client{},
			worktreeManager: &MockWorktreeManager{},
			claudeExecutor:  claude.NewClaudeExecutor(),
			claudeConfig:    &claude.ClaudeConfig{},
			stateManager:    NewIssueStateManager(),
			config:          config.NewConfig(),
			owner:           "test-owner",
			repo:            "test-repo",
			logger:          testLogger,
		}

		// Act
		action := factory.CreateImplementationAction()

		// Assert
		assert.NotNil(t, action)
		assert.Empty(t, buf.String()) // まだ実行していないためログは空
	})

	t.Run("ReviewActionにloggerが注入されること", func(t *testing.T) {
		// Arrange
		testLogger, buf := createTestLogger()
		factory := &DefaultActionFactory{
			sessionName:     "test-session",
			ghClient:        &github.Client{},
			worktreeManager: &MockWorktreeManager{},
			claudeExecutor:  claude.NewClaudeExecutor(),
			claudeConfig:    &claude.ClaudeConfig{},
			stateManager:    NewIssueStateManager(),
			config:          config.NewConfig(),
			owner:           "test-owner",
			repo:            "test-repo",
			logger:          testLogger,
		}

		// Act
		action := factory.CreateReviewAction()

		// Assert
		assert.NotNil(t, action)
		assert.Empty(t, buf.String()) // まだ実行していないためログは空
	})

	t.Run("logger未設定時のデフォルト動作", func(t *testing.T) {
		// Arrange
		factory := &DefaultActionFactory{
			sessionName:     "test-session",
			ghClient:        &github.Client{},
			worktreeManager: &MockWorktreeManager{},
			claudeExecutor:  claude.NewClaudeExecutor(),
			claudeConfig:    &claude.ClaudeConfig{},
			stateManager:    NewIssueStateManager(),
			// logger: nil (未設定)
		}

		// Act & Assert - アクションは作成できるが、実行時にloggerがnilでエラーになることを期待
		action := factory.CreatePlanAction()
		assert.NotNil(t, action)

		// 実際の実行でエラーが発生することは、各Actionのテストで確認
	})
}

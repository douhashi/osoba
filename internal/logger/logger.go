package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger はアプリケーションのログインターフェース
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	WithFields(keysAndValues ...interface{}) Logger
}

// zapLogger はzapを使用したLogger実装
type zapLogger struct {
	sugar *zap.SugaredLogger
}

// Config はロガーの設定
type Config struct {
	Level  string
	Format string
}

// Option はロガーの設定オプション
type Option func(*Config)

// WithLevel はログレベルを設定するオプション
func WithLevel(level string) Option {
	return func(c *Config) {
		c.Level = level
	}
}

// WithFormat はログフォーマットを設定するオプション
func WithFormat(format string) Option {
	return func(c *Config) {
		c.Format = format
	}
}

// New は新しいロガーを作成する
func New(opts ...Option) (Logger, error) {
	config := &Config{
		Level:  "info",
		Format: "text",
	}

	for _, opt := range opts {
		opt(config)
	}

	// ログレベルの解析
	level, err := parseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	// エンコーダー設定
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

	// エンコーダーの作成
	var encoder zapcore.Encoder
	switch config.Format {
	case "json":
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	case "text":
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	default:
		return nil, fmt.Errorf("invalid format: %s", config.Format)
	}

	// コアの作成
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		level,
	)

	// ロガーの作成
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	sugar := logger.Sugar()

	return &zapLogger{sugar: sugar}, nil
}

// parseLevel は文字列のログレベルをzapcore.Levelに変換する
func parseLevel(level string) (zapcore.Level, error) {
	switch level {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("unknown level: %s", level)
	}
}

// Debug はデバッグレベルのログを出力する
func (l *zapLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.sugar.Debugw(msg, keysAndValues...)
}

// Info は情報レベルのログを出力する
func (l *zapLogger) Info(msg string, keysAndValues ...interface{}) {
	l.sugar.Infow(msg, keysAndValues...)
}

// Warn は警告レベルのログを出力する
func (l *zapLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.sugar.Warnw(msg, keysAndValues...)
}

// Error はエラーレベルのログを出力する
func (l *zapLogger) Error(msg string, keysAndValues ...interface{}) {
	l.sugar.Errorw(msg, keysAndValues...)
}

// WithFields はフィールドを追加した新しいロガーを返す
func (l *zapLogger) WithFields(keysAndValues ...interface{}) Logger {
	return &zapLogger{
		sugar: l.sugar.With(keysAndValues...),
	}
}

// newLoggerWithCore はテスト用にカスタムコアでロガーを作成する
func newLoggerWithCore(core zapcore.Core) Logger {
	logger := zap.New(core, zap.AddCallerSkip(1))
	sugar := logger.Sugar()
	return &zapLogger{sugar: sugar}
}

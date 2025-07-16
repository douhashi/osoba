package log

import (
	"fmt"
	"io"
	"strings"
)

// Level はログレベルを表す型
type Level int

const (
	// DebugLevel はデバッグレベル
	DebugLevel Level = iota
	// InfoLevel は情報レベル
	InfoLevel
	// WarnLevel は警告レベル
	WarnLevel
	// ErrorLevel はエラーレベル
	ErrorLevel
)

// String はログレベルを文字列として返す
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel は文字列からログレベルを解析する
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return DebugLevel, nil
	case "info":
		return InfoLevel, nil
	case "warn", "warning":
		return WarnLevel, nil
	case "error":
		return ErrorLevel, nil
	default:
		return InfoLevel, fmt.Errorf("unknown level: %s", s)
	}
}

// Format はログフォーマットを表す型
type Format int

const (
	// TextFormat はテキスト形式
	TextFormat Format = iota
	// JSONFormat はJSON形式
	JSONFormat
)

// String はフォーマットを文字列として返す
func (f Format) String() string {
	switch f {
	case TextFormat:
		return "text"
	case JSONFormat:
		return "json"
	default:
		return "text"
	}
}

// Fields は構造化ログのフィールドを表す型
type Fields map[string]interface{}

// Logger は統一ログインターフェース
type Logger interface {
	// Debug はデバッグレベルのログを出力する
	Debug(msg string)
	// Info は情報レベルのログを出力する
	Info(msg string)
	// Warn は警告レベルのログを出力する
	Warn(msg string)
	// Error はエラーレベルのログを出力する
	Error(msg string)

	// WithField は指定されたフィールドを持つ新しいロガーを返す
	WithField(key string, value interface{}) Logger
	// WithFields は指定されたフィールドを持つ新しいロガーを返す
	WithFields(fields Fields) Logger
	// WithComponent は指定されたコンポーネント名を持つ新しいロガーを返す
	WithComponent(component string) Logger
}

// Config は統一ログシステムの設定
type Config struct {
	Level        Level
	Format       Format
	Component    string
	ColorEnabled bool
	Output       io.Writer
	FileOutput   string
	Fields       Fields
}

// Option は設定オプション
type Option func(*Config)

// WithLevel はログレベルを設定する
func WithLevel(level Level) Option {
	return func(c *Config) {
		c.Level = level
	}
}

// WithFormat はフォーマットを設定する
func WithFormat(format Format) Option {
	return func(c *Config) {
		c.Format = format
	}
}

// WithComponent はコンポーネント名を設定する
func WithComponent(component string) Option {
	return func(c *Config) {
		c.Component = component
	}
}

// WithColorEnabled は色分け表示を設定する
func WithColorEnabled(enabled bool) Option {
	return func(c *Config) {
		c.ColorEnabled = enabled
	}
}

// WithOutput は出力先を設定する
func WithOutput(output io.Writer) Option {
	return func(c *Config) {
		c.Output = output
	}
}

// WithFileOutput はファイル出力を設定する
func WithFileOutput(filename string) Option {
	return func(c *Config) {
		c.FileOutput = filename
	}
}

// WithFields は初期フィールドを設定する
func WithFields(fields Fields) Option {
	return func(c *Config) {
		if c.Fields == nil {
			c.Fields = make(Fields)
		}
		for k, v := range fields {
			c.Fields[k] = v
		}
	}
}

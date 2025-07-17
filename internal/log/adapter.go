package log

import (
	"github.com/douhashi/osoba/internal/logger"
)

// LoggerAdapter は既存のloggerパッケージのインターフェースを統一ログシステムに変換するアダプター
type LoggerAdapter struct {
	unified Logger
}

// NewLoggerAdapter は既存のloggerパッケージのインターフェースを統一ログシステムに変換するアダプターを作成する
func NewLoggerAdapter(unified Logger) logger.Logger {
	return &LoggerAdapter{unified: unified}
}

// Debug はデバッグレベルのログを出力する
func (a *LoggerAdapter) Debug(msg string, keysAndValues ...interface{}) {
	if len(keysAndValues) > 0 {
		fields := make(Fields)
		for i := 0; i < len(keysAndValues); i += 2 {
			if i+1 < len(keysAndValues) {
				if key, ok := keysAndValues[i].(string); ok {
					fields[key] = keysAndValues[i+1]
				}
			}
		}
		a.unified.WithFields(fields).Debug(msg)
	} else {
		a.unified.Debug(msg)
	}
}

// Info は情報レベルのログを出力する
func (a *LoggerAdapter) Info(msg string, keysAndValues ...interface{}) {
	if len(keysAndValues) > 0 {
		fields := make(Fields)
		for i := 0; i < len(keysAndValues); i += 2 {
			if i+1 < len(keysAndValues) {
				if key, ok := keysAndValues[i].(string); ok {
					fields[key] = keysAndValues[i+1]
				}
			}
		}
		a.unified.WithFields(fields).Info(msg)
	} else {
		a.unified.Info(msg)
	}
}

// Warn は警告レベルのログを出力する
func (a *LoggerAdapter) Warn(msg string, keysAndValues ...interface{}) {
	if len(keysAndValues) > 0 {
		fields := make(Fields)
		for i := 0; i < len(keysAndValues); i += 2 {
			if i+1 < len(keysAndValues) {
				if key, ok := keysAndValues[i].(string); ok {
					fields[key] = keysAndValues[i+1]
				}
			}
		}
		a.unified.WithFields(fields).Warn(msg)
	} else {
		a.unified.Warn(msg)
	}
}

// Error はエラーレベルのログを出力する
func (a *LoggerAdapter) Error(msg string, keysAndValues ...interface{}) {
	if len(keysAndValues) > 0 {
		fields := make(Fields)
		for i := 0; i < len(keysAndValues); i += 2 {
			if i+1 < len(keysAndValues) {
				if key, ok := keysAndValues[i].(string); ok {
					fields[key] = keysAndValues[i+1]
				}
			}
		}
		a.unified.WithFields(fields).Error(msg)
	} else {
		a.unified.Error(msg)
	}
}

// WithFields はフィールドを追加した新しいロガーを返す
func (a *LoggerAdapter) WithFields(keysAndValues ...interface{}) logger.Logger {
	fields := make(Fields)
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			if key, ok := keysAndValues[i].(string); ok {
				fields[key] = keysAndValues[i+1]
			}
		}
	}
	return &LoggerAdapter{unified: a.unified.WithFields(fields)}
}

// ConvertLoggerLevel は既存のloggerパッケージのログレベル文字列を統一ログシステムのレベルに変換する
func ConvertLoggerLevel(level string) Level {
	switch level {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// ConvertLoggerFormat は既存のloggerパッケージのフォーマット文字列を統一ログシステムのフォーマットに変換する
func ConvertLoggerFormat(format string) Format {
	switch format {
	case "json":
		return JSONFormat
	case "text":
		return TextFormat
	default:
		return TextFormat
	}
}

package log

import (
	"io"
	"os"
	"sync"
	"time"
)

// unifiedLogger は統一ログシステムの実装
type unifiedLogger struct {
	config     *Config
	formatter  Formatter
	colorizer  *colorizer
	output     io.Writer
	fileOutput *os.File
	mutex      sync.Mutex
}

// New は新しい統一ログシステムを作成する
func New(opts ...Option) Logger {
	config := &Config{
		Level:        InfoLevel,
		Format:       TextFormat,
		Component:    "",
		ColorEnabled: true,
		Output:       os.Stdout,
		FileOutput:   "",
		Fields:       make(Fields),
	}

	// オプションを適用
	for _, opt := range opts {
		opt(config)
	}

	logger := &unifiedLogger{
		config:    config,
		formatter: newFormatter(config.Format),
		colorizer: newColorizer(config.ColorEnabled),
		output:    config.Output,
	}

	// ファイル出力の設定
	if config.FileOutput != "" {
		if file, err := os.OpenFile(config.FileOutput, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			logger.fileOutput = file
		}
	}

	return logger
}

// Debug はデバッグレベルのログを出力する
func (l *unifiedLogger) Debug(msg string) {
	l.log(DebugLevel, msg)
}

// Info は情報レベルのログを出力する
func (l *unifiedLogger) Info(msg string) {
	l.log(InfoLevel, msg)
}

// Warn は警告レベルのログを出力する
func (l *unifiedLogger) Warn(msg string) {
	l.log(WarnLevel, msg)
}

// Error はエラーレベルのログを出力する
func (l *unifiedLogger) Error(msg string) {
	l.log(ErrorLevel, msg)
}

// WithField は指定されたフィールドを持つ新しいロガーを返す
func (l *unifiedLogger) WithField(key string, value interface{}) Logger {
	newFields := make(Fields)
	for k, v := range l.config.Fields {
		newFields[k] = v
	}
	newFields[key] = value

	return l.withFields(newFields)
}

// WithFields は指定されたフィールドを持つ新しいロガーを返す
func (l *unifiedLogger) WithFields(fields Fields) Logger {
	newFields := make(Fields)
	for k, v := range l.config.Fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return l.withFields(newFields)
}

// WithComponent は指定されたコンポーネント名を持つ新しいロガーを返す
func (l *unifiedLogger) WithComponent(component string) Logger {
	newConfig := *l.config
	newConfig.Component = component

	return &unifiedLogger{
		config:     &newConfig,
		formatter:  l.formatter,
		colorizer:  l.colorizer,
		output:     l.output,
		fileOutput: l.fileOutput,
	}
}

// withFields は指定されたフィールドを持つ新しいロガーを返す
func (l *unifiedLogger) withFields(fields Fields) Logger {
	newConfig := *l.config
	newConfig.Fields = fields

	return &unifiedLogger{
		config:     &newConfig,
		formatter:  l.formatter,
		colorizer:  l.colorizer,
		output:     l.output,
		fileOutput: l.fileOutput,
	}
}

// log は指定されたレベルでログを出力する
func (l *unifiedLogger) log(level Level, msg string) {
	// ログレベルのチェック
	if level < l.config.Level {
		return
	}

	// ログエントリーの作成
	entry := &LogEntry{
		Time:      time.Now(),
		Level:     level,
		Component: l.config.Component,
		Message:   msg,
		Fields:    l.config.Fields,
	}

	// フォーマット
	formatted := l.formatter.Format(entry, l.colorizer)

	// 出力
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// 標準出力またはカスタム出力
	if l.output != nil {
		l.output.Write([]byte(formatted + "\n"))
	}

	// ファイル出力
	if l.fileOutput != nil {
		// ファイル出力では色分けを無効化
		noColorFormatter := newFormatter(l.config.Format)
		noColorColorizer := newColorizer(false)
		fileFormatted := noColorFormatter.Format(entry, noColorColorizer)
		l.fileOutput.Write([]byte(fileFormatted + "\n"))
	}
}

// Close はロガーのリソースを解放する
func (l *unifiedLogger) Close() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.fileOutput != nil {
		return l.fileOutput.Close()
	}
	return nil
}

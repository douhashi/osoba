package logger

import (
	"os"
	"strings"
)

// ConfigFromEnv は環境変数から設定を読み込む
func ConfigFromEnv() *Config {
	config := &Config{
		Level:  "info",
		Format: "text",
	}

	// DEBUG環境変数のチェック
	debug := os.Getenv("DEBUG")
	if isTrue(debug) {
		config.Level = "debug"
	}

	// LOG_LEVEL環境変数のチェック（DEBUGより優先）
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Level = strings.ToLower(level)
	}

	// LOG_FORMAT環境変数のチェック
	if format := os.Getenv("LOG_FORMAT"); format != "" {
		config.Format = strings.ToLower(format)
	}

	return config
}

// NewFromEnv は環境変数から設定を読み込んでロガーを作成する
func NewFromEnv() (Logger, error) {
	config := ConfigFromEnv()
	return New(
		WithLevel(config.Level),
		WithFormat(config.Format),
	)
}

// isTrue は文字列がtrueを表すかチェックする
func isTrue(s string) bool {
	s = strings.ToLower(s)
	return s == "true" || s == "1" || s == "yes" || s == "on"
}

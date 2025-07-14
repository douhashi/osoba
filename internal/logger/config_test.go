package logger

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFromEnv(t *testing.T) {
	// 環境変数のバックアップと復元
	originalDebug := os.Getenv("DEBUG")
	originalLogLevel := os.Getenv("LOG_LEVEL")
	originalLogFormat := os.Getenv("LOG_FORMAT")
	defer func() {
		os.Setenv("DEBUG", originalDebug)
		os.Setenv("LOG_LEVEL", originalLogLevel)
		os.Setenv("LOG_FORMAT", originalLogFormat)
	}()

	tests := []struct {
		name       string
		envVars    map[string]string
		wantLevel  string
		wantFormat string
		wantErr    bool
	}{
		{
			name:       "デフォルト設定（環境変数なし）",
			envVars:    map[string]string{},
			wantLevel:  "info",
			wantFormat: "text",
			wantErr:    false,
		},
		{
			name: "DEBUG=trueでデバッグレベル",
			envVars: map[string]string{
				"DEBUG": "true",
			},
			wantLevel:  "debug",
			wantFormat: "text",
			wantErr:    false,
		},
		{
			name: "LOG_LEVEL指定",
			envVars: map[string]string{
				"LOG_LEVEL": "warn",
			},
			wantLevel:  "warn",
			wantFormat: "text",
			wantErr:    false,
		},
		{
			name: "LOG_FORMAT=json",
			envVars: map[string]string{
				"LOG_FORMAT": "json",
			},
			wantLevel:  "info",
			wantFormat: "json",
			wantErr:    false,
		},
		{
			name: "DEBUG=trueとLOG_LEVELの両方指定（LOG_LEVELが優先）",
			envVars: map[string]string{
				"DEBUG":     "true",
				"LOG_LEVEL": "error",
			},
			wantLevel:  "error",
			wantFormat: "text",
			wantErr:    false,
		},
		{
			name: "すべての環境変数を指定",
			envVars: map[string]string{
				"DEBUG":      "true",
				"LOG_LEVEL":  "info",
				"LOG_FORMAT": "json",
			},
			wantLevel:  "info",
			wantFormat: "json",
			wantErr:    false,
		},
		{
			name: "無効なLOG_LEVEL",
			envVars: map[string]string{
				"LOG_LEVEL": "invalid",
			},
			wantLevel:  "",
			wantFormat: "",
			wantErr:    true,
		},
		{
			name: "無効なLOG_FORMAT",
			envVars: map[string]string{
				"LOG_FORMAT": "xml",
			},
			wantLevel:  "",
			wantFormat: "",
			wantErr:    true,
		},
		{
			name: "DEBUG=false（デフォルトと同じ）",
			envVars: map[string]string{
				"DEBUG": "false",
			},
			wantLevel:  "info",
			wantFormat: "text",
			wantErr:    false,
		},
		{
			name: "DEBUG=1（trueとして扱う）",
			envVars: map[string]string{
				"DEBUG": "1",
			},
			wantLevel:  "debug",
			wantFormat: "text",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数をクリア
			os.Unsetenv("DEBUG")
			os.Unsetenv("LOG_LEVEL")
			os.Unsetenv("LOG_FORMAT")

			// テスト用の環境変数を設定
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// ロガーを作成
			logger, err := NewFromEnv()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, logger)

			// ログ出力が正しく動作することを確認
			assert.NotPanics(t, func() {
				logger.Debug("test debug")
				logger.Info("test info")
				logger.Warn("test warn")
				logger.Error("test error")
			})
		})
	}
}

func TestConfigFromEnv(t *testing.T) {
	// 環境変数のバックアップと復元
	originalDebug := os.Getenv("DEBUG")
	originalLogLevel := os.Getenv("LOG_LEVEL")
	originalLogFormat := os.Getenv("LOG_FORMAT")
	defer func() {
		os.Setenv("DEBUG", originalDebug)
		os.Setenv("LOG_LEVEL", originalLogLevel)
		os.Setenv("LOG_FORMAT", originalLogFormat)
	}()

	tests := []struct {
		name       string
		envVars    map[string]string
		wantLevel  string
		wantFormat string
	}{
		{
			name:       "環境変数なし",
			envVars:    map[string]string{},
			wantLevel:  "info",
			wantFormat: "text",
		},
		{
			name: "DEBUG=true",
			envVars: map[string]string{
				"DEBUG": "true",
			},
			wantLevel:  "debug",
			wantFormat: "text",
		},
		{
			name: "LOG_LEVELとLOG_FORMAT",
			envVars: map[string]string{
				"LOG_LEVEL":  "error",
				"LOG_FORMAT": "json",
			},
			wantLevel:  "error",
			wantFormat: "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数をクリア
			os.Unsetenv("DEBUG")
			os.Unsetenv("LOG_LEVEL")
			os.Unsetenv("LOG_FORMAT")

			// テスト用の環境変数を設定
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// 設定を取得
			config := ConfigFromEnv()

			assert.Equal(t, tt.wantLevel, config.Level)
			assert.Equal(t, tt.wantFormat, config.Format)
		})
	}
}

package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecute(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		wantErr            bool
		wantOutputContains []string
	}{
		{
			name:    "正常系: ヘルプ表示",
			args:    []string{"--help"},
			wantErr: false,
			wantOutputContains: []string{
				"osoba",
				"自律的なソフトウェア開発を支援するCLIツール",
			},
		},
		{
			name:    "正常系: バージョン表示",
			args:    []string{"--version"},
			wantErr: false,
			wantOutputContains: []string{
				"osoba version",
			},
		},
		{
			name:    "異常系: 不正なフラグ",
			args:    []string{"--invalid-flag"},
			wantErr: true,
			wantOutputContains: []string{
				"unknown flag",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// バッファを準備
			buf := new(bytes.Buffer)

			// テスト用のルートコマンドを作成
			rootCmd = newRootCmd()
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs(tt.args)

			// コマンド実行
			err := rootCmd.Execute()

			// エラーチェック
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 出力内容チェック
			output := buf.String()
			for _, want := range tt.wantOutputContains {
				if !strings.Contains(output, want) {
					t.Errorf("Execute() output = %v, want to contain %v", output, want)
				}
			}
		})
	}
}

func TestGlobalFlags(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		checkFunc func(t *testing.T)
	}{
		{
			name: "config フラグ",
			args: []string{"--config", "test.yaml"},
			checkFunc: func(t *testing.T) {
				val, err := rootCmd.Flags().GetString("config")
				if err != nil {
					t.Errorf("Failed to get config flag: %v", err)
				}
				if val != "test.yaml" {
					t.Errorf("config flag = %v, want test.yaml", val)
				}
			},
		},
		{
			name: "verbose フラグ",
			args: []string{"--verbose"},
			checkFunc: func(t *testing.T) {
				val, err := rootCmd.Flags().GetBool("verbose")
				if err != nil {
					t.Errorf("Failed to get verbose flag: %v", err)
				}
				if !val {
					t.Errorf("verbose flag = %v, want true", val)
				}
			},
		},
		{
			name: "log-level フラグ",
			args: []string{"--log-level", "debug"},
			checkFunc: func(t *testing.T) {
				val, err := rootCmd.Flags().GetString("log-level")
				if err != nil {
					t.Errorf("Failed to get log-level flag: %v", err)
				}
				if val != "debug" {
					t.Errorf("log-level flag = %v, want debug", val)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テスト用のルートコマンドを作成
			rootCmd = newRootCmd()
			rootCmd.SetArgs(tt.args)

			// フラグのパース
			err := rootCmd.ParseFlags(tt.args)
			if err != nil {
				t.Errorf("ParseFlags() error = %v", err)
				return
			}

			// チェック関数実行
			tt.checkFunc(t)
		})
	}
}

func TestLogLevelFlag(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantLogLevel string
		wantErr      bool
	}{
		{
			name:         "正常系: debug レベル",
			args:         []string{"--log-level", "debug"},
			wantLogLevel: "debug",
			wantErr:      false,
		},
		{
			name:         "正常系: info レベル",
			args:         []string{"--log-level", "info"},
			wantLogLevel: "info",
			wantErr:      false,
		},
		{
			name:         "正常系: warn レベル",
			args:         []string{"--log-level", "warn"},
			wantLogLevel: "warn",
			wantErr:      false,
		},
		{
			name:         "正常系: error レベル",
			args:         []string{"--log-level", "error"},
			wantLogLevel: "error",
			wantErr:      false,
		},
		{
			name:         "正常系: 短縮形 -l",
			args:         []string{"-l", "debug"},
			wantLogLevel: "debug",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テスト用のルートコマンドを作成
			rootCmd = newRootCmd()
			rootCmd.SetArgs(tt.args)

			// フラグのパース
			err := rootCmd.ParseFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				val, err := rootCmd.Flags().GetString("log-level")
				if err != nil {
					t.Errorf("Failed to get log-level flag: %v", err)
				}
				if val != tt.wantLogLevel {
					t.Errorf("log-level flag = %v, want %v", val, tt.wantLogLevel)
				}
			}
		})
	}
}

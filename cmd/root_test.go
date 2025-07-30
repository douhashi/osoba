package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
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

func TestInitConfig(t *testing.T) {
	tests := []struct {
		name            string
		setupConfigFile func(t *testing.T, tmpDir string) string
		cfgFileOverride string
		wantErr         bool
		checkFunc       func(t *testing.T)
	}{
		{
			name: "正常系: カレントディレクトリの.osoba.ymlを読み込み",
			setupConfigFile: func(t *testing.T, tmpDir string) string {
				configContent := `github:
  poll_interval: 10s
  labels:
    plan: status:needs-plan
    ready: status:ready
    review: status:review-requested
tmux:
  session_prefix: test-osoba-
`
				configPath := filepath.Join(tmpDir, ".osoba.yml")
				if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
					t.Fatalf("Failed to create config file: %v", err)
				}
				return configPath
			},
			wantErr: false,
			checkFunc: func(t *testing.T) {
				if viper.GetString("github.poll_interval") != "10s" {
					t.Errorf("github.poll_interval = %v, want 10s", viper.GetString("github.poll_interval"))
				}
				if viper.GetString("tmux.session_prefix") != "test-osoba-" {
					t.Errorf("tmux.session_prefix = %v, want test-osoba-", viper.GetString("tmux.session_prefix"))
				}
			},
		},
		{
			name: "正常系: カレントディレクトリの.osoba.yamlを読み込み",
			setupConfigFile: func(t *testing.T, tmpDir string) string {
				configContent := `github:
  poll_interval: 15s
tmux:
  session_prefix: yaml-osoba-
`
				configPath := filepath.Join(tmpDir, ".osoba.yaml")
				if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
					t.Fatalf("Failed to create config file: %v", err)
				}
				return configPath
			},
			wantErr: false,
			checkFunc: func(t *testing.T) {
				if viper.GetString("github.poll_interval") != "15s" {
					t.Errorf("github.poll_interval = %v, want 15s", viper.GetString("github.poll_interval"))
				}
				if viper.GetString("tmux.session_prefix") != "yaml-osoba-" {
					t.Errorf("tmux.session_prefix = %v, want yaml-osoba-", viper.GetString("tmux.session_prefix"))
				}
			},
		},
		{
			name: "正常系: 設定ファイルが存在しない場合（デフォルト値使用）",
			setupConfigFile: func(t *testing.T, tmpDir string) string {
				return ""
			},
			wantErr: false,
			checkFunc: func(t *testing.T) {
				// 設定ファイルが存在しない場合はReadInConfigでエラーが発生するが
				// ConfigFileNotFoundErrorの場合は正常終了する
			},
		},
		{
			name: "正常系: 指定されたconfigファイルを読み込み",
			setupConfigFile: func(t *testing.T, tmpDir string) string {
				configContent := `github:
  poll_interval: 30s
tmux:
  session_prefix: custom-osoba-
`
				configPath := filepath.Join(tmpDir, "custom-config.yml")
				if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
					t.Fatalf("Failed to create config file: %v", err)
				}
				return configPath
			},
			cfgFileOverride: "custom-config.yml",
			wantErr:         false,
			checkFunc: func(t *testing.T) {
				if viper.GetString("github.poll_interval") != "30s" {
					t.Errorf("github.poll_interval = %v, want 30s", viper.GetString("github.poll_interval"))
				}
				if viper.GetString("tmux.session_prefix") != "custom-osoba-" {
					t.Errorf("tmux.session_prefix = %v, want custom-osoba-", viper.GetString("tmux.session_prefix"))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テンポラリディレクトリを作成
			tmpDir, err := os.MkdirTemp("", "osoba-test-")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// 元のディレクトリを保存してテンポラリディレクトリに移動
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current dir: %v", err)
			}
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("Failed to change dir: %v", err)
			}
			defer func() {
				os.Chdir(originalDir)
			}()

			// Viperのクリーンアップ
			viper.Reset()

			// テスト用の設定ファイルを作成
			configPath := tt.setupConfigFile(t, tmpDir)

			// cfgFileの設定
			if tt.cfgFileOverride != "" {
				cfgFile = tt.cfgFileOverride
			} else {
				cfgFile = ""
			}

			// initConfig実行
			err = initConfig()

			// エラーチェック
			if (err != nil) != tt.wantErr {
				t.Errorf("initConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 追加チェック
			if tt.checkFunc != nil {
				tt.checkFunc(t)
			}

			// configPathが設定されている場合のファイル読み込み確認
			if configPath != "" && tt.cfgFileOverride == "" && !tt.wantErr {
				// 設定ファイルが正しく使用されたかを確認するため、
				// 実際にviperから値を読み取ってテストする
				configFile := viper.ConfigFileUsed()
				if configFile != "" {
					expectedPath, _ := filepath.Abs(configPath)
					actualPath, _ := filepath.Abs(configFile)
					if actualPath != expectedPath {
						t.Errorf("Used config file = %v, want %v", actualPath, expectedPath)
					}
				}
			}
		})
	}
}

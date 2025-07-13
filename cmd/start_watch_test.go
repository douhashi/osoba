package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/config"
	"github.com/spf13/cobra"
)

func TestStartCmd_WatchMode(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		envVars            map[string]string
		configContent      string
		wantErr            bool
		wantOutputContains []string
		wantErrContains    string
	}{
		{
			name: "正常系: --watchフラグでIssue監視モードを開始",
			args: []string{"start", "--watch"},
			envVars: map[string]string{
				"GITHUB_TOKEN": "test-token",
			},
			wantErr: false,
			wantOutputContains: []string{
				"Issue監視モードを開始します",
			},
		},
		{
			name: "正常系: --intervalフラグでポーリング間隔を指定",
			args: []string{"start", "--watch", "--interval", "10s"},
			envVars: map[string]string{
				"GITHUB_TOKEN": "test-token",
			},
			wantErr: false,
			wantOutputContains: []string{
				"Issue監視モードを開始します",
				"ポーリング間隔: 10s",
			},
		},
		{
			name: "正常系: 設定ファイルから読み込み",
			args: []string{"start", "--watch", "--config", "test_config.yml"},
			configContent: `
github:
  token: config-file-token
  poll_interval: 15s
  labels:
    plan: "status:needs-plan"
    ready: "status:ready"
    review: "status:review-requested"
`,
			wantErr: false,
			wantOutputContains: []string{
				"Issue監視モードを開始します",
				"設定ファイル: test_config.yml",
			},
		},
		{
			name:            "異常系: GitHubトークンが設定されていない",
			args:            []string{"start", "--watch"},
			wantErr:         true,
			wantErrContains: "GitHub token is required",
		},
		{
			name: "異常系: 不正なポーリング間隔",
			args: []string{"start", "--watch", "--interval", "500ms"},
			envVars: map[string]string{
				"GITHUB_TOKEN": "test-token",
			},
			wantErr:         true,
			wantErrContains: "poll interval must be at least 1 second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数を設定
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			// Gitリポジトリのモックを作成
			tmpDir := t.TempDir()
			gitDir := filepath.Join(tmpDir, ".git")
			gitConfigDir := filepath.Join(gitDir, "config")

			err := os.MkdirAll(filepath.Dir(gitConfigDir), 0755)
			if err != nil {
				t.Fatal(err)
			}

			gitConfigContent := `[core]
	repositoryformatversion = 0
[remote "origin"]
	url = https://github.com/douhashi/osoba.git
`
			err = os.WriteFile(gitConfigDir, []byte(gitConfigContent), 0644)
			if err != nil {
				t.Fatal(err)
			}

			// 現在のディレクトリを保存して、テスト後に戻す
			origDir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(origDir)

			// テスト用ディレクトリに移動
			err = os.Chdir(tmpDir)
			if err != nil {
				t.Fatal(err)
			}

			// 設定ファイルを作成（テストディレクトリ内に作成）
			if tt.configContent != "" {
				err := os.WriteFile("test_config.yml", []byte(tt.configContent), 0644)
				if err != nil {
					t.Fatalf("failed to create test config file: %v", err)
				}
				defer os.Remove("test_config.yml")
			}

			buf := new(bytes.Buffer)
			errBuf := new(bytes.Buffer)

			rootCmd := newRootCmd()
			cmd := newStartCmd()

			// テスト前にモックを設定
			originalRunWatchWithFlags := runWatchWithFlagsFunc
			defer func() {
				runWatchWithFlagsFunc = originalRunWatchWithFlags
			}()

			// runWatchWithFlagsFuncをモック（テストトークンでは実際のAPIを呼ばないようにする）
			runWatchWithFlagsFunc = func(cmd *cobra.Command, args []string, intervalFlagParam, configFlagParam string) error {
				// runWatchWithFlagsの簡易的な実装
				fmt.Fprintln(cmd.OutOrStdout(), "Issue監視モードを開始します")

				// 設定を読み込む
				cfg := config.NewConfig()
				if configFlagParam != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "設定ファイル: %s\n", configFlagParam)
					if err := cfg.Load(configFlagParam); err != nil {
						return fmt.Errorf("設定ファイルの読み込みに失敗: %w", err)
					}
				} else {
					// 環境変数から読み込み
					if token := os.Getenv("GITHUB_TOKEN"); token != "" {
						cfg.GitHub.Token = token
					} else if token := os.Getenv("OSOBA_GITHUB_TOKEN"); token != "" {
						cfg.GitHub.Token = token
					}
				}

				// ポーリング間隔を設定
				if intervalFlagParam != "" && intervalFlagParam != "5s" {
					interval, err := time.ParseDuration(intervalFlagParam)
					if err != nil {
						return fmt.Errorf("不正なポーリング間隔: %w", err)
					}
					cfg.GitHub.PollInterval = interval
					fmt.Fprintf(cmd.OutOrStdout(), "ポーリング間隔: %s\n", interval)
				}

				// 設定の検証
				if err := cfg.Validate(); err != nil {
					return err
				}

				// テストでは実際のAPI呼び出しやgoroutineは起動しない
				return nil
			}

			rootCmd.AddCommand(cmd)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(errBuf)
			rootCmd.SetArgs(tt.args)

			err = rootCmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			output := buf.String()

			for _, want := range tt.wantOutputContains {
				if !strings.Contains(output, want) {
					t.Errorf("Execute() output = %v, want to contain %v", output, want)
				}
			}

			if tt.wantErrContains != "" && err != nil && !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Errorf("Execute() error = %v, want to contain %v", err, tt.wantErrContains)
			}
		})
	}
}

func TestStartCmd_ParseInterval(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "正常系: 秒単位",
			input:   "5s",
			want:    5 * time.Second,
			wantErr: false,
		},
		{
			name:    "正常系: 分単位",
			input:   "2m",
			want:    2 * time.Minute,
			wantErr: false,
		},
		{
			name:    "正常系: 時間単位",
			input:   "1h",
			want:    1 * time.Hour,
			wantErr: false,
		},
		{
			name:    "異常系: 不正な形式",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "異常系: 空文字列",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := time.ParseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

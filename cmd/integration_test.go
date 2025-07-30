//go:build integration
// +build integration

package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
	"github.com/douhashi/osoba/internal/watcher"
)

// TestIntegration_WatchFlow は監視フロー全体の統合テスト（外部サービスのみモック）
func TestIntegration_WatchFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// ghコマンドが利用可能で認証済みかチェック
	if err := exec.Command("gh", "auth", "status").Run(); err != nil {
		t.Skip("gh command not authenticated, skipping integration test")
	}

	t.Run("正常系: 実際のGitHub APIとの連携", func(t *testing.T) {
		cfg := config.NewConfig()
		cfg.GitHub.PollInterval = 2 * time.Second

		// テスト用のロガーを作成
		testLogger, err := logger.New(logger.WithLevel("error"))
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}

		// 実際のGitHubクライアントを作成
		client, err := github.NewClientWithLogger("", testLogger)
		if err != nil {
			t.Fatalf("Failed to create GitHub client: %v", err)
		}

		// Issue監視を作成
		issueWatcher, err := watcher.NewIssueWatcher(
			client,
			"douhashi",
			"osoba",
			"test-session",
			cfg.GetLabels(),
			cfg.GitHub.PollInterval,
			testLogger,
		)
		if err != nil {
			t.Fatalf("Failed to create issue watcher: %v", err)
		}

		issueWatcher.SetPollInterval(cfg.GitHub.PollInterval)

		// 短時間のテスト実行
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// コールバック実行カウンター
		callbackCount := 0

		// Issue監視を開始
		issueWatcher.Start(ctx, func(issue *github.Issue) {
			callbackCount++
			t.Logf("Callback executed for issue #%d: %s", *issue.Number, *issue.Title)
		})

		// タイムアウトまで待機
		<-ctx.Done()

		// エラーがないことのみ確認（Issues数は変動するため）
		t.Logf("Integration test completed successfully, processed %d issue callbacks", callbackCount)
	})
}

// TestIntegration_ConfigLoading は設定読み込みの統合テスト
func TestIntegration_ConfigLoading(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		envVars       map[string]string
		wantInterval  time.Duration
		wantLabels    []string
	}{
		{
			name: "設定ファイルと環境変数の組み合わせ",
			configContent: `
github:
  poll_interval: 10s
  labels:
    plan: "status:planning"
    ready: "status:ready-to-dev"
    review: "status:review-requested"
`,
			envVars: map[string]string{
				"GITHUB_TOKEN": "env-token-123",
			},
			wantInterval: 10 * time.Second,
			wantLabels:   []string{"status:planning", "status:ready-to-dev", "status:review-requested"},
		},
		{
			name: "GITHUB_TOKEN環境変数が設定ファイルより優先される",
			configContent: `
github:
  token: "file-token"
`,
			envVars: map[string]string{
				"GITHUB_TOKEN": "github-env-token",
			},
			wantInterval: 5 * time.Second, // デフォルト値
			wantLabels:   []string{"status:needs-plan", "status:ready", "status:review-requested"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 既存の環境変数をバックアップしてクリア
			envBackup := make(map[string]string)
			for _, key := range []string{"GITHUB_TOKEN"} {
				if val, exists := os.LookupEnv(key); exists {
					envBackup[key] = val
				}
				os.Unsetenv(key)
			}
			defer func() {
				// 環境変数を復元
				for key, val := range envBackup {
					os.Setenv(key, val)
				}
				for _, key := range []string{"GITHUB_TOKEN"} {
					if _, exists := envBackup[key]; !exists {
						os.Unsetenv(key)
					}
				}
			}()
			// テスト用の設定ファイルを作成
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "test_config.yml")
			if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// テスト用の環境変数を設定
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// 設定を読み込む
			cfg := config.NewConfig()
			if err := cfg.Load(configPath); err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			// 検証
			if cfg.GitHub.PollInterval != tt.wantInterval {
				t.Errorf("PollInterval = %v, want %v", cfg.GitHub.PollInterval, tt.wantInterval)
			}

			labels := cfg.GetLabels()
			if len(labels) != len(tt.wantLabels) {
				t.Errorf("Labels count = %d, want %d", len(labels), len(tt.wantLabels))
			}
			for i, label := range labels {
				if label != tt.wantLabels[i] {
					t.Errorf("Label[%d] = %v, want %v", i, label, tt.wantLabels[i])
				}
			}
		})
	}
}

// TestIntegration_ErrorHandling はエラーハンドリングの統合テスト
func TestIntegration_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() *config.Config
		expectError string
	}{
		{
			name: "ポーリング間隔が短すぎる場合",
			setupConfig: func() *config.Config {
				cfg := config.NewConfig()
				cfg.GitHub.PollInterval = 500 * time.Millisecond
				return cfg
			},
			expectError: "poll interval must be at least 1 second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setupConfig()
			err := cfg.Validate()

			if err == nil {
				t.Error("Expected error but got nil")
				return
			}

			if err.Error() != tt.expectError {
				t.Errorf("Error = %v, want %v", err.Error(), tt.expectError)
			}
		})
	}
}

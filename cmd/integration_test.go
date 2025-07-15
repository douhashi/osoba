package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/watcher"
	gh "github.com/google/go-github/v67/github"
)

// モックGitHubクライアント
type mockGitHubClient struct {
	issues    []*gh.Issue
	err       error
	callCount int
	rateLimit *gh.RateLimits
}

func (m *mockGitHubClient) GetRepository(ctx context.Context, owner, repo string) (*gh.Repository, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &gh.Repository{
		Name: gh.String(repo),
		Owner: &gh.User{
			Login: gh.String(owner),
		},
	}, nil
}

func (m *mockGitHubClient) ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*gh.Issue, error) {
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	return m.issues, nil
}

func (m *mockGitHubClient) GetRateLimit(ctx context.Context) (*gh.RateLimits, error) {
	if m.rateLimit != nil {
		return m.rateLimit, nil
	}

	resetTime := gh.Timestamp{Time: time.Now().Add(1 * time.Hour)}
	return &gh.RateLimits{
		Core: &gh.Rate{
			Limit:     5000,
			Remaining: 4999,
			Reset:     resetTime,
		},
	}, nil
}

// TestIntegration_WatchFlow は監視フロー全体の統合テスト
func TestIntegration_WatchFlow(t *testing.T) {
	tests := []struct {
		name          string
		setupConfig   func() *config.Config
		mockClient    *mockGitHubClient
		expectedCalls int
		timeout       time.Duration
		wantErr       bool
	}{
		{
			name: "正常系: 新しいIssueを検出してコールバックが実行される",
			setupConfig: func() *config.Config {
				cfg := config.NewConfig()
				cfg.GitHub.Token = "test-token"
				cfg.GitHub.PollInterval = time.Second
				return cfg
			},
			mockClient: &mockGitHubClient{
				issues: []*gh.Issue{
					{
						Number: gh.Int(1),
						Title:  gh.String("Test Issue 1"),
						Labels: []*gh.Label{
							{Name: gh.String("status:needs-plan")},
						},
					},
				},
			},
			expectedCalls: 2,
			timeout:       3 * time.Second,
			wantErr:       false,
		},
		{
			name: "正常系: 複数のIssueを検出",
			setupConfig: func() *config.Config {
				cfg := config.NewConfig()
				cfg.GitHub.Token = "test-token"
				cfg.GitHub.PollInterval = time.Second
				return cfg
			},
			mockClient: &mockGitHubClient{
				issues: []*gh.Issue{
					{
						Number: gh.Int(1),
						Title:  gh.String("Test Issue 1"),
						Labels: []*gh.Label{
							{Name: gh.String("status:needs-plan")},
						},
					},
					{
						Number: gh.Int(2),
						Title:  gh.String("Test Issue 2"),
						Labels: []*gh.Label{
							{Name: gh.String("status:ready")},
						},
					},
				},
			},
			expectedCalls: 2,
			timeout:       3 * time.Second,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setupConfig()

			// Issue監視を作成
			issueWatcher, err := watcher.NewIssueWatcher(
				tt.mockClient,
				"douhashi",
				"osoba",
				"test-session",
				cfg.GetLabels(),
			)
			if err != nil {
				t.Fatalf("Failed to create issue watcher: %v", err)
			}

			issueWatcher.SetPollInterval(cfg.GitHub.PollInterval)

			// コールバック実行カウンター
			callbackCount := 0
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			// Issue監視を開始
			issueWatcher.Start(ctx, func(issue *gh.Issue) {
				callbackCount++
				t.Logf("Callback executed for issue #%d: %s", *issue.Number, *issue.Title)
			})

			// タイムアウトまで待機
			<-ctx.Done()

			// 期待される呼び出し回数を確認
			if tt.mockClient.callCount < tt.expectedCalls {
				t.Errorf("API calls = %d, want at least %d", tt.mockClient.callCount, tt.expectedCalls)
			}

			// ステートレス化により、各ポーリングサイクルで同じIssueが処理される
			// 期待される最小呼び出し回数は Issue数 × API呼び出し回数
			expectedMinCallbacks := len(tt.mockClient.issues) * tt.expectedCalls
			if callbackCount < expectedMinCallbacks {
				t.Errorf("Callback calls = %d, want at least %d", callbackCount, expectedMinCallbacks)
			}
		})
	}
}

// TestIntegration_ConfigLoading は設定読み込みの統合テスト
func TestIntegration_ConfigLoading(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		envVars       map[string]string
		wantToken     string
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
    review: "status:needs-review"
`,
			envVars: map[string]string{
				"GITHUB_TOKEN": "env-token-123",
			},
			wantToken:    "env-token-123",
			wantInterval: 10 * time.Second,
			wantLabels:   []string{"status:planning", "status:ready-to-dev", "status:needs-review"},
		},
		{
			name: "OSOBA_GITHUB_TOKEN環境変数が優先される",
			configContent: `
github:
  token: "file-token"
`,
			envVars: map[string]string{
				"GITHUB_TOKEN":       "github-token",
				"OSOBA_GITHUB_TOKEN": "osoba-token",
			},
			wantToken:    "osoba-token",
			wantInterval: 5 * time.Second, // デフォルト値
			wantLabels:   []string{"status:needs-plan", "status:ready", "status:review-requested"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テスト用の設定ファイルを作成
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "test_config.yml")
			if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// 環境変数を設定
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			// 設定を読み込む
			cfg := config.NewConfig()
			if err := cfg.Load(configPath); err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			// 検証
			if cfg.GitHub.Token != tt.wantToken {
				t.Errorf("Token = %v, want %v", cfg.GitHub.Token, tt.wantToken)
			}
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
			name: "GitHubトークンが空の場合",
			setupConfig: func() *config.Config {
				cfg := config.NewConfig()
				cfg.GitHub.Token = ""
				return cfg
			},
			expectError: "GitHub token is required",
		},
		{
			name: "ポーリング間隔が短すぎる場合",
			setupConfig: func() *config.Config {
				cfg := config.NewConfig()
				cfg.GitHub.Token = "test-token"
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

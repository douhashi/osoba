package watcher

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/github"
	gh "github.com/douhashi/osoba/internal/github"
	"github.com/stretchr/testify/mock"
)

// TestIssueProcessingWithLabelTransition は処理済み判定後もラベル遷移が実行されることを確認する統合テスト
func TestIssueProcessingWithLabelTransition(t *testing.T) {
	tests := []struct {
		name                  string
		issue                 *gh.Issue
		hasBeenProcessed      bool
		expectedRemoveLabel   string
		expectedAddLabel      string
		expectLabelTransition bool
	}{
		{
			name: "already processed issue with needs-plan label should transition",
			issue: &gh.Issue{
				Number: intPtr(123),
				Labels: []*gh.Label{
					{Name: stringPtr("status:needs-plan")},
				},
			},
			hasBeenProcessed:      true,
			expectedRemoveLabel:   "status:needs-plan",
			expectedAddLabel:      "status:planning",
			expectLabelTransition: true,
		},
		{
			name: "already processed issue with ready label should transition",
			issue: &gh.Issue{
				Number: intPtr(456),
				Labels: []*gh.Label{
					{Name: stringPtr("status:ready")},
				},
			},
			hasBeenProcessed:      true,
			expectedRemoveLabel:   "status:ready",
			expectedAddLabel:      "status:implementing",
			expectLabelTransition: true,
		},
		{
			name: "already processed issue with review-requested label should transition",
			issue: &gh.Issue{
				Number: intPtr(789),
				Labels: []*gh.Label{
					{Name: stringPtr("status:review-requested")},
				},
			},
			hasBeenProcessed:      true,
			expectedRemoveLabel:   "status:review-requested",
			expectedAddLabel:      "status:reviewing",
			expectLabelTransition: true,
		},
		{
			name: "not processed issue should also transition",
			issue: &gh.Issue{
				Number: intPtr(999),
				Labels: []*gh.Label{
					{Name: stringPtr("status:needs-plan")},
				},
			},
			hasBeenProcessed:      false,
			expectedRemoveLabel:   "status:needs-plan",
			expectedAddLabel:      "status:planning",
			expectLabelTransition: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックの設定
			mockClient := new(MockGitHubClient)
			mockActionManager := new(MockActionManager)
			mockLogger := NewMockLogger()

			// GitHub APIのモック設定
			mockClient.On("ListIssuesByLabels", mock.Anything, "owner", "repo", []string{"status:needs-plan", "status:ready", "status:review-requested"}).
				Return([]*gh.Issue{tt.issue}, nil)

			// アクション実行のモック設定
			if tt.hasBeenProcessed {
				// 既に処理済みの場合、エラーを返す（処理済みを示す）
				mockActionManager.On("ExecuteAction", mock.Anything, tt.issue).
					Return(nil) // 処理済みでもエラーは返さない
			} else {
				// 未処理の場合、正常に実行
				mockActionManager.On("ExecuteAction", mock.Anything, tt.issue).
					Return(nil)
			}

			// ラベル遷移のモック設定
			if tt.expectLabelTransition {
				mockClient.On("RemoveLabel", mock.Anything, "owner", "repo", *tt.issue.Number, tt.expectedRemoveLabel).
					Return(nil)
				mockClient.On("AddLabel", mock.Anything, "owner", "repo", *tt.issue.Number, tt.expectedAddLabel).
					Return(nil)
			}

			// IssueWatcherの作成
			watcher := &IssueWatcher{
				client:        mockClient,
				owner:         "owner",
				repo:          "repo",
				labels:        []string{"status:needs-plan", "status:ready", "status:review-requested"},
				pollInterval:  100 * time.Millisecond,
				actionManager: mockActionManager,
				logger:        mockLogger,
			}

			// コンテキストの設定
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			// StartWithActionsを実行
			go watcher.StartWithActions(ctx)

			// 処理が完了するまで待機
			time.Sleep(150 * time.Millisecond)

			// モックの検証
			mockClient.AssertExpectations(t)
			mockActionManager.AssertExpectations(t)
		})
	}
}

// MockActionManager はActionManagerのモック実装
type MockActionManager struct {
	mock.Mock
}

func (m *MockActionManager) ExecuteAction(ctx context.Context, issue *gh.Issue) error {
	args := m.Called(ctx, issue)
	return args.Error(0)
}

func (m *MockActionManager) GetActionForIssue(issue *gh.Issue) ActionExecutor {
	args := m.Called(issue)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(ActionExecutor)
}

func (m *MockActionManager) SetActionFactory(factory ActionFactory) {
	m.Called(factory)
}

func (m *MockActionManager) GetStateManager() *IssueStateManager {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*IssueStateManager)
}

// 既存の統合テスト（mainブランチから）
func TestStartWithActionsIntegration(t *testing.T) {
	t.Run("複数のIssueを連続して処理", func(t *testing.T) {
		// テスト用のIssueを作成
		issues := []*github.Issue{
			{
				Number: github.Int(1),
				Title:  github.String("Test Issue 1"),
				Labels: []*github.Label{
					{Name: github.String("status:needs-plan")},
				},
			},
			{
				Number: github.Int(2),
				Title:  github.String("Test Issue 2"),
				Labels: []*github.Label{
					{Name: github.String("status:ready")},
				},
			},
		}

		mockClient := &mockGitHubClient{
			issues: issues,
		}

		// IssueWatcherを作成（ラベル変更追跡有効）
		watcher, err := NewIssueWatcherWithLabelTracking(
			mockClient,
			"douhashi",
			"osoba",
			"test-session",
			[]string{"status:needs-plan", "status:ready"},
			5*time.Second,
			NewMockLogger(),
		)
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		// ポーリング間隔を最小値に設定（テスト用）
		if err := watcher.SetPollInterval(1 * time.Second); err != nil {
			t.Fatalf("failed to set poll interval: %v", err)
		}

		// テスト用のActionFactoryを設定
		factory := &mockActionFactory{
			planAction: &mockAction{
				canExecute: func(issue *github.Issue) bool {
					return hasLabel(issue, "status:needs-plan")
				},
				execute: func(ctx context.Context, issue *github.Issue) error {
					t.Logf("Executing plan action for issue #%d", *issue.Number)
					return nil
				},
			},
			implementationAction: &mockAction{
				canExecute: func(issue *github.Issue) bool {
					return hasLabel(issue, "status:ready")
				},
				execute: func(ctx context.Context, issue *github.Issue) error {
					t.Logf("Executing implementation action for issue #%d", *issue.Number)
					return nil
				},
			},
		}
		watcher.GetActionManager().SetActionFactory(factory)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// アクション実行をカウント
		actionCount := 0
		actionMu := sync.Mutex{}
		factory.planAction.execute = func(ctx context.Context, issue *github.Issue) error {
			actionMu.Lock()
			actionCount++
			actionMu.Unlock()
			t.Logf("Plan action executed for issue #%d", *issue.Number)
			return nil
		}
		factory.implementationAction.execute = func(ctx context.Context, issue *github.Issue) error {
			actionMu.Lock()
			actionCount++
			actionMu.Unlock()
			t.Logf("Implementation action executed for issue #%d", *issue.Number)
			return nil
		}

		// StartWithActionsを実行
		go watcher.StartWithActions(ctx)

		// アクションが実行されるまで待機
		time.Sleep(2 * time.Second)

		// 検証
		actionMu.Lock()
		if actionCount != 2 {
			t.Errorf("Expected 2 actions to be executed, got %d", actionCount)
		}
		actionMu.Unlock()

		t.Log("All actions executed successfully")
	})
}

// 複数のIssue監視の統合テスト
func TestConcurrentWatchers(t *testing.T) {
	t.Run("複数のリポジトリを同時に監視", func(t *testing.T) {
		// 複数のモッククライアントを作成
		mockClient1 := &mockGitHubClient{
			issues: []*github.Issue{
				{
					Number: github.Int(1),
					Title:  github.String("Repo1 Issue"),
					Labels: []*github.Label{
						{Name: github.String("status:ready")},
					},
				},
			},
		}

		mockClient2 := &mockGitHubClient{
			issues: []*github.Issue{
				{
					Number: github.Int(2),
					Title:  github.String("Repo2 Issue"),
					Labels: []*github.Label{
						{Name: github.String("status:ready")},
					},
				},
			},
		}

		// 2つのwatcherを作成
		watcher1, err := NewIssueWatcher(
			mockClient1,
			"douhashi",
			"repo1",
			"test-session-1",
			[]string{"status:ready"},
			5*time.Second,
			NewMockLogger(),
		)
		if err != nil {
			t.Fatalf("failed to create watcher1: %v", err)
		}

		watcher2, err := NewIssueWatcher(
			mockClient2,
			"douhashi",
			"repo2",
			"test-session-2",
			[]string{"status:ready"},
			5*time.Second,
			NewMockLogger(),
		)
		if err != nil {
			t.Fatalf("failed to create watcher2: %v", err)
		}

		// ポーリング間隔を最小値に設定（テスト用）
		if err := watcher1.SetPollInterval(1 * time.Second); err != nil {
			t.Fatalf("failed to set poll interval for watcher1: %v", err)
		}
		if err := watcher2.SetPollInterval(1 * time.Second); err != nil {
			t.Fatalf("failed to set poll interval for watcher2: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 各watcherからのコールバックをカウント
		callbackCount := 0
		callbackMu := sync.Mutex{}

		callback := func(issue *github.Issue) {
			callbackMu.Lock()
			callbackCount++
			callbackMu.Unlock()
			t.Logf("Received callback for issue #%d: %s", *issue.Number, *issue.Title)
		}

		// 両方のwatcherを起動
		go watcher1.Start(ctx, callback)
		go watcher2.Start(ctx, callback)

		// コールバックが実行されるまで待機
		time.Sleep(3 * time.Second)

		// 検証
		callbackMu.Lock()
		if callbackCount < 2 {
			t.Errorf("Expected at least 2 callbacks, got %d", callbackCount)
		}
		callbackMu.Unlock()

		t.Log("Concurrent watchers test completed successfully")
	})
}

// 設定の統合テスト
func TestConfigIntegration(t *testing.T) {
	t.Run("設定ファイルとIssueWatcherの統合", func(t *testing.T) {
		// テスト用設定ファイルを作成
		configContent := `
github:
  token: "test-token"
  owner: "douhashi"
  repo: "osoba"
  poll_interval: 2s
  max_retries: 5
  retry_base_delay: 500ms
  labels:
    - "status:needs-plan"
    - "status:ready"
    - "status:review-requested"
`

		tmpfile, err := os.CreateTemp("", "integration-config-*.yaml")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(configContent)); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatalf("failed to close temp file: %v", err)
		}

		// 設定を読み込み
		config, err := LoadConfig(tmpfile.Name())
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		// 設定値を検証
		if config.GitHub.Token != "test-token" {
			t.Errorf("Token = %v, want test-token", config.GitHub.Token)
		}
		if config.GitHub.PollInterval != 2*time.Second {
			t.Errorf("PollInterval = %v, want 2s", config.GitHub.PollInterval)
		}
		if config.GitHub.MaxRetries != 5 {
			t.Errorf("MaxRetries = %v, want 5", config.GitHub.MaxRetries)
		}
		if config.GitHub.RetryBaseDelay != 500*time.Millisecond {
			t.Errorf("RetryBaseDelay = %v, want 500ms", config.GitHub.RetryBaseDelay)
		}
		if len(config.GitHub.Labels) != 3 {
			t.Errorf("Labels length = %v, want 3", len(config.GitHub.Labels))
		}

		// IssueWatcherに設定を適用
		mockClient := &mockGitHubClient{
			issues: []*github.Issue{
				{
					Number: github.Int(1),
					Title:  github.String("Config Test Issue"),
					Labels: []*github.Label{
						{Name: github.String("status:ready")},
					},
				},
			},
		}

		watcher, err := NewIssueWatcher(
			mockClient,
			config.GitHub.Owner,
			config.GitHub.Repo,
			"test-session",
			config.GitHub.Labels,
			config.GitHub.PollInterval,
			NewMockLogger(),
		)
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		// 設定値をwatcherに適用
		if err := watcher.SetPollInterval(config.GitHub.PollInterval); err != nil {
			t.Fatalf("failed to set poll interval: %v", err)
		}

		// 設定が正しく適用されているか確認
		if watcher.GetPollInterval() != config.GitHub.PollInterval {
			t.Errorf("Watcher poll interval = %v, want %v",
				watcher.GetPollInterval(), config.GitHub.PollInterval)
		}

		t.Log("Config integration test completed successfully")
	})
}
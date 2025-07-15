package watcher

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/go-github/v67/github"
)

// 統合テスト: GitHub Issue監視の全体ワークフローをテスト
func TestIssueWatcherIntegration(t *testing.T) {
	t.Run("Issue検出からイベント通知までの完全なワークフロー", func(t *testing.T) {
		// テスト用のissueデータ
		initialIssues := []*github.Issue{
			{
				Number: github.Int(1),
				Title:  github.String("新機能の実装"),
				Labels: []*github.Label{
					{Name: github.String("status:needs-plan")},
					{Name: github.String("enhancement")},
				},
			},
		}

		updatedIssues := []*github.Issue{
			{
				Number: github.Int(1),
				Title:  github.String("新機能の実装"),
				Labels: []*github.Label{
					{Name: github.String("status:ready")},
					{Name: github.String("enhancement")},
				},
			},
			{
				Number: github.Int(2),
				Title:  github.String("バグ修正"),
				Labels: []*github.Label{
					{Name: github.String("status:needs-plan")},
					{Name: github.String("bug")},
				},
			},
		}

		callCount := 0
		callCountMu := sync.Mutex{}
		mockClient := &mockGitHubClient{
			listIssuesFunc: func(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error) {
				callCountMu.Lock()
				callCount++
				current := callCount
				callCountMu.Unlock()
				if current == 1 {
					return initialIssues, nil
				}
				return updatedIssues, nil
			},
		}

		// EventNotifierを作成
		notifier := NewEventNotifier(20)
		defer notifier.Close()

		// IssueWatcherを作成（ラベル変更追跡有効）
		watcher, err := NewIssueWatcherWithLabelTracking(
			mockClient,
			"douhashi",
			"osoba",
			"test-session",
			[]string{"status:needs-plan", "status:ready"},
		)
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		// イベント通知を有効化
		watcher.SetEventNotifier(notifier)
		if err := watcher.SetPollInterval(time.Second); err != nil {
			t.Fatalf("failed to set poll interval: %v", err)
		}

		// イベント受信用チャネル
		eventCh := notifier.Subscribe()
		receivedEvents := make([]IssueEvent, 0)
		eventMu := sync.Mutex{}

		// イベント収集ゴルーチン
		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case event, ok := <-eventCh:
					if !ok {
						return
					}
					eventMu.Lock()
					receivedEvents = append(receivedEvents, event)
					eventMu.Unlock()
				case <-ctx.Done():
					return
				}
			}
		}()

		// 監視開始
		issueCallbackCount := 0
		issueCallbackMu := sync.Mutex{}
		go watcher.Start(ctx, func(issue *github.Issue) {
			issueCallbackMu.Lock()
			issueCallbackCount++
			issueCallbackMu.Unlock()
			t.Logf("Issue callback called for #%d: %s", *issue.Number, *issue.Title)
		})

		// 十分な時間待機してイベントを収集（複数回のポーリングを実行）
		time.Sleep(2500 * time.Millisecond)

		// 停止
		cancel()
		wg.Wait()

		// 結果を検証
		eventMu.Lock()
		defer eventMu.Unlock()

		// 少なくとも以下のイベントが発生することを期待:
		// 1. Issue #1 detected
		// 2. Issue #1 label changed (status:needs-plan -> status:ready)
		// 3. Issue #2 detected
		if len(receivedEvents) < 3 {
			t.Fatalf("Expected at least 3 events, got %d", len(receivedEvents))
		}

		// イベントタイプごとに分類
		issueDetectedCount := 0
		labelChangedCount := 0
		for _, event := range receivedEvents {
			switch event.Type {
			case IssueDetected:
				issueDetectedCount++
			case LabelChanged:
				labelChangedCount++
				// ラベル変更の詳細を検証
				if event.IssueID == 1 {
					if event.FromLabel != "status:needs-plan" {
						t.Errorf("Expected FromLabel 'status:needs-plan', got '%s'", event.FromLabel)
					}
					if event.ToLabel != "status:ready" {
						t.Errorf("Expected ToLabel 'status:ready', got '%s'", event.ToLabel)
					}
				}
			}
		}

		// 期待するイベント数を検証
		if issueDetectedCount < 2 {
			t.Errorf("Expected at least 2 issue detected events, got %d", issueDetectedCount)
		}
		if labelChangedCount < 1 {
			t.Errorf("Expected at least 1 label changed event, got %d", labelChangedCount)
		}

		// コールバックが呼ばれたことを確認
		issueCallbackMu.Lock()
		finalCallbackCount := issueCallbackCount
		issueCallbackMu.Unlock()
		if finalCallbackCount < 2 {
			t.Errorf("Expected at least 2 callback invocations, got %d", finalCallbackCount)
		}

		t.Logf("Integration test completed successfully:")
		t.Logf("- Total events: %d", len(receivedEvents))
		t.Logf("- Issue detected events: %d", issueDetectedCount)
		t.Logf("- Label changed events: %d", labelChangedCount)
		t.Logf("- Callback invocations: %d", finalCallbackCount)
	})
}

// リトライ機能の統合テスト
func TestRetryIntegration(t *testing.T) {
	t.Run("APIエラー時のリトライとイベント通知", func(t *testing.T) {
		callCount := 0
		callCountMu := sync.Mutex{}
		mockClient := &mockGitHubClient{
			listIssuesFunc: func(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error) {
				callCountMu.Lock()
				callCount++
				current := callCount
				callCountMu.Unlock()
				if current <= 1 {
					// 最初の1回はエラーを返す
					return nil, &github.ErrorResponse{
						Response: &http.Response{
							StatusCode: 503,
							Status:     "503 Service Unavailable",
							Request: &http.Request{
								Method: "GET",
								URL:    &url.URL{Scheme: "https", Host: "api.github.com", Path: "/repos/douhashi/osoba/issues"},
							},
						},
						Message: "Service Unavailable",
					}
				}
				// 2回目は成功
				return []*github.Issue{
					{
						Number: github.Int(1),
						Title:  github.String("Test Issue"),
						Labels: []*github.Label{
							{Name: github.String("status:ready")},
						},
					},
				}, nil
			},
		}

		watcher, err := NewIssueWatcher(
			mockClient,
			"douhashi",
			"osoba",
			"test-session",
			[]string{"status:ready"},
		)
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		// ポーリング間隔を最小値に設定（テスト用）
		if err := watcher.SetPollInterval(1 * time.Second); err != nil {
			t.Fatalf("failed to set poll interval: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		issueCallbackCount := 0
		issueCallbackMu := sync.Mutex{}
		go watcher.Start(ctx, func(issue *github.Issue) {
			issueCallbackMu.Lock()
			issueCallbackCount++
			issueCallbackMu.Unlock()
		})

		// リトライが成功するまで待機（2.5秒で十分）
		time.Sleep(2500 * time.Millisecond)
		cancel()

		// リトライが機能して最終的にissueが検出されることを確認
		issueCallbackMu.Lock()
		finalIssueCallbackCount := issueCallbackCount
		issueCallbackMu.Unlock()
		callCountMu.Lock()
		finalCallCount := callCount
		callCountMu.Unlock()

		if finalIssueCallbackCount == 0 {
			t.Error("Expected issue to be detected after retries")
		}

		// APIが複数回呼ばれたことを確認（リトライが発生した証拠）
		if finalCallCount < 2 {
			t.Errorf("Expected at least 2 API calls due to retries, got %d", finalCallCount)
		}

		t.Logf("Retry integration test completed:")
		t.Logf("- API calls: %d", finalCallCount)
		t.Logf("- Issues detected: %d", finalIssueCallbackCount)
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

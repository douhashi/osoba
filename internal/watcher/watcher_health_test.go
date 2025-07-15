package watcher

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v67/github"
)

func TestIssueWatcher_HealthCheck(t *testing.T) {
	t.Run("最後の正常実行時刻が記録される", func(t *testing.T) {
		mockClient := &mockGitHubClient{
			issues: []*github.Issue{
				{
					Number: github.Int(1),
					Title:  github.String("Test Issue"),
					Labels: []*github.Label{
						{Name: github.String("status:needs-plan")},
					},
				},
			},
		}

		watcher, err := NewIssueWatcher(mockClient, "douhashi", "osoba", "test-session", []string{"status:needs-plan"})
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		watcher.pollInterval = 100 * time.Millisecond

		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		// 初期状態では最後の実行時刻は未設定
		lastExecution := watcher.GetLastExecutionTime()
		if !lastExecution.IsZero() {
			t.Errorf("Expected zero time for initial state, got %v", lastExecution)
		}

		go watcher.Start(ctx, func(issue *github.Issue) {})

		// 最初の実行が完了するまで待つ
		time.Sleep(150 * time.Millisecond)

		// 最後の実行時刻が更新されているか確認
		lastExecution = watcher.GetLastExecutionTime()
		if lastExecution.IsZero() {
			t.Error("Last execution time was not updated")
		}

		// 実行時刻が現在時刻に近いか確認（1秒以内）
		if time.Since(lastExecution) > time.Second {
			t.Errorf("Last execution time is too old: %v", lastExecution)
		}
	})

	t.Run("統計情報が正しく記録される", func(t *testing.T) {
		callCount := 0
		mockClient := &mockGitHubClient{
			listIssuesFunc: func(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error) {
				callCount++
				if callCount%2 == 0 {
					// 偶数回はエラーを返す
					return nil, &github.RateLimitError{Message: "rate limit"}
				}
				return []*github.Issue{
					{
						Number: github.Int(callCount),
						Title:  github.String("Test Issue"),
						Labels: []*github.Label{
							{Name: github.String("status:needs-plan")},
						},
					},
				}, nil
			},
		}

		watcher, err := NewIssueWatcher(mockClient, "douhashi", "osoba", "test-session", []string{"status:needs-plan"})
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		watcher.pollInterval = 100 * time.Millisecond

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		go watcher.Start(ctx, func(issue *github.Issue) {})

		time.Sleep(550 * time.Millisecond)

		// 統計情報を取得
		stats := watcher.GetHealthStats()

		// 実行回数の確認（少なくとも3回は実行されているはず）
		if stats.TotalExecutions < 3 {
			t.Errorf("Expected at least 3 executions, got %d", stats.TotalExecutions)
		}

		// 成功と失敗の両方が記録されているか
		if stats.SuccessfulExecutions == 0 {
			t.Error("No successful executions recorded")
		}
		if stats.FailedExecutions == 0 {
			t.Error("No failed executions recorded")
		}

		// 成功率の確認（約50%のはず）
		successRate := float64(stats.SuccessfulExecutions) / float64(stats.TotalExecutions) * 100
		if successRate < 30 || successRate > 70 {
			t.Errorf("Unexpected success rate: %.2f%%", successRate)
		}
	})

	t.Run("長時間実行されていない場合のアラート", func(t *testing.T) {
		mockClient := &mockGitHubClient{
			issues: []*github.Issue{},
		}

		watcher, err := NewIssueWatcher(mockClient, "douhashi", "osoba", "test-session", []string{"status:needs-plan"})
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		// ヘルスチェックのアラート閾値を設定（デフォルト: 10分）
		healthStatus := watcher.CheckHealth(5 * time.Minute)

		// 初期状態では未実行なのでアラートが出る
		if healthStatus.IsHealthy {
			t.Error("Expected unhealthy status for never-executed watcher")
		}

		if !strings.Contains(healthStatus.Message, "never been executed") {
			t.Errorf("Expected 'never been executed' message, got: %s", healthStatus.Message)
		}

		// 実行を開始
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		watcher.pollInterval = 50 * time.Millisecond
		go watcher.Start(ctx, func(issue *github.Issue) {})

		time.Sleep(150 * time.Millisecond)

		// 実行後は健全な状態
		healthStatus = watcher.CheckHealth(5 * time.Minute)
		if !healthStatus.IsHealthy {
			t.Errorf("Expected healthy status after execution, got: %s", healthStatus.Message)
		}

		// 時間経過をシミュレート（実際の実装では最後の実行時刻を手動で設定できるようにする）
		// この部分は実装時に対応
	})
}

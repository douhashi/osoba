package watcher

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-github/v67/github"
)

func TestIssueWatcher_MemoryManagement(t *testing.T) {
	t.Run("seenIssuesマップのサイズが制限される", func(t *testing.T) {
		// 大量のIssueを生成
		var testIssues []*github.Issue
		for i := 1; i <= 1000; i++ {
			testIssues = append(testIssues, &github.Issue{
				Number: github.Int(i),
				Title:  github.String("Test Issue"),
				Labels: []*github.Label{
					{Name: github.String("status:needs-plan")},
				},
			})
		}

		callCount := 0
		mockClient := &mockGitHubClient{
			listIssuesFunc: func(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error) {
				callCount++
				// 毎回異なるIssueを返す
				start := (callCount - 1) * 100
				end := start + 100
				if end > len(testIssues) {
					end = len(testIssues)
				}
				return testIssues[start:end], nil
			},
		}

		watcher, err := NewIssueWatcher(mockClient, "douhashi", "osoba", "test-session", []string{"status:needs-plan"})
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		// 最大サイズを設定（デフォルトは500に設定する予定）
		const maxSeenIssues = 500

		watcher.pollInterval = 10 * time.Millisecond

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		go watcher.Start(ctx, func(issue *github.Issue) {})

		// 十分な回数のサイクルが実行されるまで待つ
		time.Sleep(250 * time.Millisecond)

		// seenIssuesのサイズを確認
		seenCount := watcher.GetSeenIssuesCount()
		if seenCount > maxSeenIssues {
			t.Errorf("seenIssues map size exceeded limit: got %d, want <= %d",
				seenCount, maxSeenIssues)
		}
	})

	t.Run("古いエントリが削除される", func(t *testing.T) {
		// 最初の100個のIssue
		initialIssues := make([]*github.Issue, 100)
		for i := 0; i < 100; i++ {
			initialIssues[i] = &github.Issue{
				Number: github.Int(i + 1),
				Title:  github.String("Initial Issue"),
				Labels: []*github.Label{
					{Name: github.String("status:needs-plan")},
				},
			}
		}

		// 新しい600個のIssue（古いものを押し出すため）
		newIssues := make([]*github.Issue, 600)
		for i := 0; i < 600; i++ {
			newIssues[i] = &github.Issue{
				Number: github.Int(i + 101),
				Title:  github.String("New Issue"),
				Labels: []*github.Label{
					{Name: github.String("status:needs-plan")},
				},
			}
		}

		callCount := 0
		mockClient := &mockGitHubClient{
			listIssuesFunc: func(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error) {
				callCount++
				if callCount == 1 {
					return initialIssues, nil
				}
				// 2回目以降は新しいIssueを返す
				return newIssues, nil
			},
		}

		watcher, err := NewIssueWatcher(mockClient, "douhashi", "osoba", "test-session", []string{"status:needs-plan"})
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		watcher.pollInterval = 50 * time.Millisecond

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		go watcher.Start(ctx, func(issue *github.Issue) {})

		time.Sleep(250 * time.Millisecond)

		// 古いIssue（ID 1-100）が削除されているか確認
		oldIssueFound := false
		for i := 1; i <= 100; i++ {
			if watcher.HasSeenIssue(int64(i)) {
				oldIssueFound = true
				break
			}
		}

		if !oldIssueFound {
			// 期待通り：古いIssueが削除されている
			return
		}

		// 新しいIssueが存在することを確認
		newIssueFound := false
		for i := 101; i <= 700; i++ {
			if watcher.HasSeenIssue(int64(i)) {
				newIssueFound = true
				break
			}
		}

		if !newIssueFound {
			t.Error("New issues were not tracked")
		}
	})
}

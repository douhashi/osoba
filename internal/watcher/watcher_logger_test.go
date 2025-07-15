package watcher

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v67/github"
)

func TestIssueWatcher_Logging(t *testing.T) {

	// モックIssueデータ
	testIssues := []*github.Issue{
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

	t.Run("各監視サイクルの開始と終了がログに記録される", func(t *testing.T) {
		// ログ出力をキャプチャするためのバッファ
		var logBuf bytes.Buffer
		originalOutput := log.Writer()
		log.SetOutput(&logBuf)
		defer log.SetOutput(originalOutput)

		mockClient := &mockGitHubClient{
			issues: testIssues,
		}

		watcher, err := NewIssueWatcher(mockClient, "douhashi", "osoba", "test-session", []string{"status:needs-plan", "status:ready"})
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		// ポーリング間隔を短くしてテストを高速化
		watcher.pollInterval = 100 * time.Millisecond

		ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
		defer cancel()

		// 監視を開始
		done := make(chan struct{})
		go func() {
			watcher.Start(ctx, func(issue *github.Issue) {})
			close(done)
		}()

		// ログが記録されるまで待つ
		time.Sleep(400 * time.Millisecond)
		cancel()

		// goroutineの終了を待つ
		<-done

		logOutput := logBuf.String()

		// 期待されるログメッセージが含まれているか確認
		expectedLogs := []string{
			"Starting issue check cycle",
			"Completed issue check cycle",
			"checked issues:",
			"processed issues:",
			"time taken:",
		}

		for _, expected := range expectedLogs {
			if !strings.Contains(logOutput, expected) {
				t.Errorf("Expected log message not found: %s", expected)
			}
		}

		// 少なくとも2回のサイクルが実行されていることを確認
		cycleCount := strings.Count(logOutput, "Starting issue check cycle")
		if cycleCount < 2 {
			t.Errorf("Expected at least 2 cycles, got %d", cycleCount)
		}
	})

	t.Run("エラー発生時に詳細なコンテキスト情報がログに記録される", func(t *testing.T) {
		// ログ出力をキャプチャするためのバッファ
		var logBuf bytes.Buffer
		originalOutput := log.Writer()
		log.SetOutput(&logBuf)
		defer log.SetOutput(originalOutput)

		mockClient := &mockGitHubClient{
			listIssuesFunc: func(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error) {
				return nil, &github.RateLimitError{
					Message: "API rate limit exceeded",
				}
			},
		}

		watcher, err := NewIssueWatcher(mockClient, "douhashi", "osoba", "test-session", []string{"status:needs-plan"})
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		watcher.pollInterval = 100 * time.Millisecond

		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		defer cancel()

		done := make(chan struct{})
		go func() {
			watcher.Start(ctx, func(issue *github.Issue) {})
			close(done)
		}()

		time.Sleep(200 * time.Millisecond)
		cancel()

		// goroutineの終了を待つ
		<-done

		logOutput := logBuf.String()

		// エラーコンテキスト情報が含まれているか確認
		expectedContextInfo := []string{
			"Failed to list issues",
			"owner=douhashi",
			"repo=osoba",
			"labels=[status:needs-plan]",
		}

		for _, expected := range expectedContextInfo {
			if !strings.Contains(logOutput, expected) {
				t.Errorf("Expected error context not found: %s", expected)
			}
		}
	})
}

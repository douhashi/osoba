package watcher

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/go-github/v50/github"
)

func TestNewIssueWatcher(t *testing.T) {
	tests := []struct {
		name    string
		owner   string
		repo    string
		labels  []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "正常系: Issue監視を作成できる",
			owner:   "douhashi",
			repo:    "osoba",
			labels:  []string{"status:needs-plan", "status:ready", "status:review-requested"},
			wantErr: false,
		},
		{
			name:    "異常系: ownerが空でエラー",
			owner:   "",
			repo:    "osoba",
			labels:  []string{"status:needs-plan"},
			wantErr: true,
			errMsg:  "owner is required",
		},
		{
			name:    "異常系: repoが空でエラー",
			owner:   "douhashi",
			repo:    "",
			labels:  []string{"status:needs-plan"},
			wantErr: true,
			errMsg:  "repo is required",
		},
		{
			name:    "異常系: labelsが空でエラー",
			owner:   "douhashi",
			repo:    "osoba",
			labels:  []string{},
			wantErr: true,
			errMsg:  "at least one label is required",
		},
		{
			name:    "異常系: labelsがnilでエラー",
			owner:   "douhashi",
			repo:    "osoba",
			labels:  nil,
			wantErr: true,
			errMsg:  "at least one label is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックGitHubクライアント
			mockClient := &mockGitHubClient{}

			watcher, err := NewIssueWatcher(mockClient, tt.owner, tt.repo, "test-session", tt.labels)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewIssueWatcher() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("NewIssueWatcher() error = %v, want %v", err.Error(), tt.errMsg)
			}
			if !tt.wantErr && watcher == nil {
				t.Error("NewIssueWatcher() returned nil watcher")
			}
		})
	}
}

func TestIssueWatcher_Start(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	t.Run("正常系: Issue検出時にコールバックが呼ばれる", func(t *testing.T) {
		mockClient := &mockGitHubClient{
			issues: testIssues,
		}

		watcher, err := NewIssueWatcher(mockClient, "douhashi", "osoba", "test-session", []string{"status:needs-plan", "status:ready"})
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		// 検出されたIssueを記録
		detectedIssues := make(map[int]bool)
		var mu sync.Mutex
		callback := func(issue *github.Issue) {
			mu.Lock()
			detectedIssues[*issue.Number] = true
			mu.Unlock()
		}

		// ポーリング間隔を短くしてテストを高速化
		if err := watcher.SetPollInterval(100 * time.Millisecond); err != nil {
			// テスト環境では1秒未満を許可
			watcher.pollInterval = 100 * time.Millisecond
		}

		// 監視を開始
		go watcher.Start(ctx, callback)

		// 少し待ってIssueが検出されることを確認
		time.Sleep(300 * time.Millisecond)

		mu.Lock()
		detected1 := detectedIssues[1]
		detected2 := detectedIssues[2]
		mu.Unlock()

		if !detected1 {
			t.Error("Issue #1 was not detected")
		}
		if !detected2 {
			t.Error("Issue #2 was not detected")
		}
	})

	t.Run("正常系: contextキャンセルで停止する", func(t *testing.T) {
		mockClient := &mockGitHubClient{
			issues: testIssues,
		}

		watcher, err := NewIssueWatcher(mockClient, "douhashi", "osoba", "test-session", []string{"status:needs-plan"})
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())

		// 監視が終了したことを確認するためのチャネル
		done := make(chan bool)
		go func() {
			watcher.Start(ctx, func(issue *github.Issue) {})
			done <- true
		}()

		// 少し待ってからキャンセル
		time.Sleep(100 * time.Millisecond)
		cancel()

		// 監視が終了することを確認
		select {
		case <-done:
			// 正常に終了
		case <-time.After(1 * time.Second):
			t.Error("watcher did not stop after context cancel")
		}
	})

	t.Run("異常系: APIエラー時は継続する", func(t *testing.T) {
		callCount := 0
		var callMu sync.Mutex
		mockClient := &mockGitHubClient{
			listIssuesFunc: func(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error) {
				callMu.Lock()
				callCount++
				count := callCount
				callMu.Unlock()

				if count == 1 {
					// 初回はエラーを返す
					return nil, fmt.Errorf("not found")
				}
				// 2回目以降は正常な結果を返す
				return testIssues, nil
			},
		}

		watcher, err := NewIssueWatcher(mockClient, "douhashi", "osoba", "test-session", []string{"status:needs-plan"})
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		detectedIssues := make(map[int]bool)
		var mu sync.Mutex
		callback := func(issue *github.Issue) {
			mu.Lock()
			detectedIssues[*issue.Number] = true
			mu.Unlock()
		}

		if err := watcher.SetPollInterval(100 * time.Millisecond); err != nil {
			// テスト環境では1秒未満を許可
			watcher.pollInterval = 100 * time.Millisecond
		}

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		go watcher.Start(ctx, callback)

		// エラー後も継続してIssueが検出されることを確認
		time.Sleep(350 * time.Millisecond)

		callMu.Lock()
		finalCallCount := callCount
		callMu.Unlock()

		if finalCallCount < 2 {
			t.Error("API was not retried after error")
		}

		mu.Lock()
		detected1 := detectedIssues[1]
		mu.Unlock()

		if !detected1 {
			t.Error("Issue #1 was not detected after retry")
		}
	})
}

func TestIssueWatcher_GetRateLimit(t *testing.T) {
	t.Run("正常系: レート制限情報を取得できる", func(t *testing.T) {
		mockClient := &mockGitHubClient{
			rateLimit: &github.RateLimits{
				Core: &github.Rate{
					Limit:     5000,
					Remaining: 4999,
				},
			},
		}

		watcher, err := NewIssueWatcher(mockClient, "douhashi", "osoba", "test-session", []string{"status:needs-plan"})
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		rateLimit, err := watcher.GetRateLimit(context.Background())
		if err != nil {
			t.Errorf("GetRateLimit() error = %v", err)
			return
		}
		if rateLimit == nil {
			t.Error("GetRateLimit() returned nil")
			return
		}
		if rateLimit.Core.Remaining != 4999 {
			t.Errorf("GetRateLimit() remaining = %d, want 4999", rateLimit.Core.Remaining)
		}
	})
}

// モッククライアント
type mockGitHubClient struct {
	issues         []*github.Issue
	rateLimit      *github.RateLimits
	listIssuesFunc func(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error)
}

func (m *mockGitHubClient) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	return &github.Repository{
		Name:  github.String(repo),
		Owner: &github.User{Login: github.String(owner)},
	}, nil
}

func (m *mockGitHubClient) ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error) {
	if m.listIssuesFunc != nil {
		return m.listIssuesFunc(ctx, owner, repo, labels)
	}
	return m.issues, nil
}

func (m *mockGitHubClient) GetRateLimit(ctx context.Context) (*github.RateLimits, error) {
	if m.rateLimit != nil {
		return m.rateLimit, nil
	}
	return &github.RateLimits{
		Core: &github.Rate{
			Limit:     5000,
			Remaining: 5000,
		},
	}, nil
}

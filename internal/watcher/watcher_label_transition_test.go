package watcher

import (
	"context"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/github"
	gogithub "github.com/douhashi/osoba/internal/github"
)

type mockGitHubClientWithTransition struct {
	github.GitHubClient
	transitionCalls []transitionCall
}

type transitionCall struct {
	owner       string
	repo        string
	issueNumber int
}

func (m *mockGitHubClientWithTransition) TransitionIssueLabelWithInfo(ctx context.Context, owner, repo string, issueNumber int) (bool, *github.TransitionInfo, error) {
	m.transitionCalls = append(m.transitionCalls, transitionCall{
		owner:       owner,
		repo:        repo,
		issueNumber: issueNumber,
	})
	return true, &github.TransitionInfo{
		FromLabel: "status:needs-plan",
		ToLabel:   "status:planning",
	}, nil
}

func (m *mockGitHubClientWithTransition) TransitionIssueLabel(ctx context.Context, owner, repo string, issueNumber int) (bool, error) {
	m.transitionCalls = append(m.transitionCalls, transitionCall{
		owner:       owner,
		repo:        repo,
		issueNumber: issueNumber,
	})
	return true, nil
}

func (m *mockGitHubClientWithTransition) ListIssues(ctx context.Context, owner, repo string, opts *gogithub.IssueListByRepoOptions) ([]*gogithub.Issue, *gogithub.Response, error) {
	return []*gogithub.Issue{}, &gogithub.Response{}, nil
}

func (m *mockGitHubClientWithTransition) ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*gogithub.Issue, error) {
	// テスト用にstatus:needs-planラベル付きのIssueを返す
	if len(labels) > 0 && labels[0] == "status:needs-plan" {
		return []*gogithub.Issue{
			{
				Number: gogithub.Int(123),
				Title:  gogithub.String("test issue"),
				Labels: []*gogithub.Label{
					{Name: gogithub.String("status:needs-plan")},
				},
			},
		}, nil
	}
	return []*gogithub.Issue{}, nil
}

func (m *mockGitHubClientWithTransition) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, comment string) error {
	return nil
}

func (m *mockGitHubClientWithTransition) RemoveLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	return nil
}

func (m *mockGitHubClientWithTransition) AddLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	return nil
}

func (m *mockGitHubClientWithTransition) GetRateLimit(ctx context.Context) (*gogithub.RateLimits, error) {
	return &gogithub.RateLimits{}, nil
}

func (m *mockGitHubClientWithTransition) GetRepository(ctx context.Context, owner, repo string) (*gogithub.Repository, error) {
	return &gogithub.Repository{}, nil
}

func (m *mockGitHubClientWithTransition) EnsureLabelsExist(ctx context.Context, owner, repo string) error {
	return nil
}

// 現在の問題を示すテスト（Red フェーズ）
func TestIssueWatcher_CurrentProblemWithLabelTransition(t *testing.T) {
	t.Run("現在: processIssueメソッドでラベル遷移が実行される（問題）", func(t *testing.T) {
		mockClient := &mockGitHubClientWithTransition{
			transitionCalls: []transitionCall{},
		}

		// IssueWatcherを作成
		watcher, err := NewIssueWatcher(mockClient, "test-owner", "test-repo", "test-session", []string{"status:needs-plan"}, 5*time.Second, NewMockLogger())
		if err != nil {
			t.Fatalf("NewIssueWatcher failed: %v", err)
		}

		ctx := context.Background()

		// checkIssuesメソッドを直接呼び出し（これによりラベル遷移が実行される）
		watcher.checkIssues(ctx, func(issue *gogithub.Issue) {})

		// ラベル遷移が実行されたことを確認（これが問題）
		if len(mockClient.transitionCalls) == 0 {
			t.Log("現在はラベル遷移が実行されていません - これは期待する動作です")
		} else {
			t.Logf("processIssuesメソッドでラベル遷移が%d回実行されました。これは修正が必要です", len(mockClient.transitionCalls))
			// このテストは問題を示すためのもので、今は失敗することを期待しない
		}
	})
}

// 修正後のテスト（Green フェーズ用）
func TestIssueWatcher_FixedLabelTransition(t *testing.T) {
	t.Run("修正後: Issue検知時にはラベル遷移を実行しない", func(t *testing.T) {
		mockClient := &mockGitHubClientWithTransition{
			transitionCalls: []transitionCall{},
		}

		// IssueWatcherを作成
		watcher, err := NewIssueWatcher(mockClient, "test-owner", "test-repo", "test-session", []string{"status:needs-plan"}, 5*time.Second, NewMockLogger())
		if err != nil {
			t.Fatalf("NewIssueWatcher failed: %v", err)
		}

		ctx := context.Background()

		// checkIssuesメソッドを呼び出し
		watcher.checkIssues(ctx, func(issue *gogithub.Issue) {})

		// ラベル遷移が実行されていないことを確認
		if len(mockClient.transitionCalls) > 0 {
			t.Errorf("修正後もIssue検知時にラベル遷移が実行されています。実行回数: %d",
				len(mockClient.transitionCalls))
		}
	})
}

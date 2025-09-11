package watcher

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGitHubClientForAutoPlan はauto_plan機能テスト用のモッククライアント
type MockGitHubClientForAutoPlan struct {
	mock.Mock
}

func (m *MockGitHubClientForAutoPlan) ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error) {
	args := m.Called(ctx, owner, repo, labels)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*github.Issue), args.Error(1)
}

func (m *MockGitHubClientForAutoPlan) ListAllOpenIssues(ctx context.Context, owner, repo string) ([]*github.Issue, error) {
	args := m.Called(ctx, owner, repo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*github.Issue), args.Error(1)
}

func (m *MockGitHubClientForAutoPlan) AddLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

func (m *MockGitHubClientForAutoPlan) TransitionLabels(ctx context.Context, owner, repo string, issueNumber int, removeLabel, addLabel string) error {
	args := m.Called(ctx, owner, repo, issueNumber, removeLabel, addLabel)
	return args.Error(0)
}

func (m *MockGitHubClientForAutoPlan) GetRateLimit(ctx context.Context) (*github.RateLimits, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.RateLimits), args.Error(1)
}

func (m *MockGitHubClientForAutoPlan) TransitionIssueLabel(ctx context.Context, owner, repo string, issueNumber int) (bool, error) {
	args := m.Called(ctx, owner, repo, issueNumber)
	return args.Bool(0), args.Error(1)
}

func (m *MockGitHubClientForAutoPlan) TransitionIssueLabelWithInfo(ctx context.Context, owner, repo string, issueNumber int) (bool, *github.TransitionInfo, error) {
	args := m.Called(ctx, owner, repo, issueNumber)
	if args.Get(1) == nil {
		return args.Bool(0), nil, args.Error(2)
	}
	return args.Bool(0), args.Get(1).(*github.TransitionInfo), args.Error(2)
}

func (m *MockGitHubClientForAutoPlan) EnsureLabelsExist(ctx context.Context, owner, repo string) error {
	args := m.Called(ctx, owner, repo)
	return args.Error(0)
}

func (m *MockGitHubClientForAutoPlan) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, comment string) error {
	args := m.Called(ctx, owner, repo, issueNumber, comment)
	return args.Error(0)
}

func (m *MockGitHubClientForAutoPlan) RemoveLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

func (m *MockGitHubClientForAutoPlan) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	args := m.Called(ctx, owner, repo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.Repository), args.Error(1)
}

func (m *MockGitHubClientForAutoPlan) GetPullRequestForIssue(ctx context.Context, issueNumber int) (*github.PullRequest, error) {
	args := m.Called(ctx, issueNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.PullRequest), args.Error(1)
}

func (m *MockGitHubClientForAutoPlan) MergePullRequest(ctx context.Context, prNumber int) error {
	args := m.Called(ctx, prNumber)
	return args.Error(0)
}

func (m *MockGitHubClientForAutoPlan) GetPullRequestStatus(ctx context.Context, prNumber int) (*github.PullRequest, error) {
	args := m.Called(ctx, prNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.PullRequest), args.Error(1)
}

func (m *MockGitHubClientForAutoPlan) ListPullRequestsByLabels(ctx context.Context, owner, repo string, labels []string) ([]*github.PullRequest, error) {
	args := m.Called(ctx, owner, repo, labels)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*github.PullRequest), args.Error(1)
}

func (m *MockGitHubClientForAutoPlan) GetClosingIssueNumber(ctx context.Context, prNumber int) (int, error) {
	args := m.Called(ctx, prNumber)
	return args.Int(0), args.Error(1)
}

func (m *MockGitHubClientForAutoPlan) ListClosedIssues(ctx context.Context, owner, repo string) ([]*github.Issue, error) {
	args := m.Called(ctx, owner, repo)
	if issues := args.Get(0); issues != nil {
		return issues.([]*github.Issue), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestExecuteAutoPlanIfNoActiveIssues(t *testing.T) {
	testLogger, _ := logger.New(logger.WithLevel("debug"))

	t.Run("正常系: status:*ラベルがない場合、最も若い番号のIssueにラベル付与", func(t *testing.T) {
		mockClient := new(MockGitHubClientForAutoPlan)

		// status:*ラベル付きIssueなし
		mockClient.On("ListIssuesByLabels", mock.Anything, "test-owner", "test-repo",
			[]string{"status:needs-plan", "status:planning", "status:ready", "status:implementing", "status:review-requested", "status:reviewing", "status:lgtm", "status:requires-changes", "status:revising"}).
			Return([]*github.Issue{}, nil)

		// ラベルなしIssueが存在
		allIssues := []*github.Issue{
			{
				Number: github.Int(5),
				Title:  github.String("Issue 5"),
				Labels: []*github.Label{}, // ラベルなし
			},
			{
				Number: github.Int(3),
				Title:  github.String("Issue 3"),
				Labels: []*github.Label{}, // ラベルなし
			},
			{
				Number: github.Int(7),
				Title:  github.String("Issue 7"),
				Labels: []*github.Label{
					{Name: github.String("bug")}, // status:*以外のラベル
				},
			},
		}
		mockClient.On("ListAllOpenIssues", mock.Anything, "test-owner", "test-repo").
			Return(allIssues, nil)

		// Issue #3（最も若い番号）にラベル付与
		mockClient.On("AddLabel", mock.Anything, "test-owner", "test-repo", 3, "status:needs-plan").
			Return(nil)

		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				AutoPlanIssue: true,
			},
		}

		err := executeAutoPlanIfNoActiveIssues(context.Background(), cfg, mockClient, "test-owner", "test-repo", testLogger)

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("正常系: auto_plan_issue設定が無効の場合はスキップ", func(t *testing.T) {
		mockClient := new(MockGitHubClientForAutoPlan)

		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				AutoPlanIssue: false,
			},
		}

		err := executeAutoPlanIfNoActiveIssues(context.Background(), cfg, mockClient, "test-owner", "test-repo", testLogger)

		assert.NoError(t, err)
		mockClient.AssertNotCalled(t, "ListIssuesByLabels")
		mockClient.AssertNotCalled(t, "ListAllOpenIssues")
	})

	t.Run("正常系: status:*ラベル付きIssueが存在する場合はスキップ", func(t *testing.T) {
		mockClient := new(MockGitHubClientForAutoPlan)

		// status:*ラベル付きIssueが存在
		activeIssues := []*github.Issue{
			{
				Number: github.Int(10),
				Title:  github.String("Active Issue"),
				Labels: []*github.Label{
					{Name: github.String("status:implementing")},
				},
			},
		}
		mockClient.On("ListIssuesByLabels", mock.Anything, "test-owner", "test-repo",
			[]string{"status:needs-plan", "status:planning", "status:ready", "status:implementing", "status:review-requested", "status:reviewing", "status:lgtm", "status:requires-changes", "status:revising"}).
			Return(activeIssues, nil)

		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				AutoPlanIssue: true,
			},
		}

		err := executeAutoPlanIfNoActiveIssues(context.Background(), cfg, mockClient, "test-owner", "test-repo", testLogger)

		assert.NoError(t, err)
		mockClient.AssertNotCalled(t, "ListAllOpenIssues")
		mockClient.AssertNotCalled(t, "AddLabel")
	})

	t.Run("正常系: status:revisingラベル付きIssueが存在する場合はスキップ", func(t *testing.T) {
		mockClient := new(MockGitHubClientForAutoPlan)

		// status:revisingラベル付きIssueが存在
		activeIssues := []*github.Issue{
			{
				Number: github.Int(20),
				Title:  github.String("Revising Issue"),
				Labels: []*github.Label{
					{Name: github.String("status:revising")},
				},
			},
		}
		mockClient.On("ListIssuesByLabels", mock.Anything, "test-owner", "test-repo",
			[]string{"status:needs-plan", "status:planning", "status:ready", "status:implementing", "status:review-requested", "status:reviewing", "status:lgtm", "status:requires-changes", "status:revising"}).
			Return(activeIssues, nil)

		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				AutoPlanIssue: true,
			},
		}

		err := executeAutoPlanIfNoActiveIssues(context.Background(), cfg, mockClient, "test-owner", "test-repo", testLogger)

		assert.NoError(t, err)
		mockClient.AssertNotCalled(t, "ListAllOpenIssues")
		mockClient.AssertNotCalled(t, "AddLabel")
	})

	t.Run("正常系: ラベルなしIssueが存在しない場合はスキップ", func(t *testing.T) {
		mockClient := new(MockGitHubClientForAutoPlan)

		// status:*ラベル付きIssueなし
		mockClient.On("ListIssuesByLabels", mock.Anything, "test-owner", "test-repo",
			[]string{"status:needs-plan", "status:planning", "status:ready", "status:implementing", "status:review-requested", "status:reviewing", "status:lgtm", "status:requires-changes", "status:revising"}).
			Return([]*github.Issue{}, nil)

		// すべてのIssueがstatus:*ラベル付き
		allIssues := []*github.Issue{
			{
				Number: github.Int(5),
				Title:  github.String("Issue 5"),
				Labels: []*github.Label{
					{Name: github.String("status:planning")},
				},
			},
			{
				Number: github.Int(3),
				Title:  github.String("Issue 3"),
				Labels: []*github.Label{
					{Name: github.String("status:ready")},
					{Name: github.String("enhancement")},
				},
			},
		}
		mockClient.On("ListAllOpenIssues", mock.Anything, "test-owner", "test-repo").
			Return(allIssues, nil)

		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				AutoPlanIssue: true,
			},
		}

		err := executeAutoPlanIfNoActiveIssues(context.Background(), cfg, mockClient, "test-owner", "test-repo", testLogger)

		assert.NoError(t, err)
		mockClient.AssertNotCalled(t, "AddLabel")
	})

	t.Run("異常系: GitHub API呼び出し失敗", func(t *testing.T) {
		mockClient := new(MockGitHubClientForAutoPlan)

		// API呼び出しが失敗
		mockClient.On("ListIssuesByLabels", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return(nil, errors.New("API error"))

		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				AutoPlanIssue: true,
			},
		}

		err := executeAutoPlanIfNoActiveIssues(context.Background(), cfg, mockClient, "test-owner", "test-repo", testLogger)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list active issues")
	})

	t.Run("異常系: ラベル付与失敗", func(t *testing.T) {
		mockClient := new(MockGitHubClientForAutoPlan)

		// status:*ラベル付きIssueなし
		mockClient.On("ListIssuesByLabels", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return([]*github.Issue{}, nil)

		// ラベルなしIssueが存在
		allIssues := []*github.Issue{
			{
				Number: github.Int(1),
				Title:  github.String("Issue 1"),
				Labels: []*github.Label{},
			},
		}
		mockClient.On("ListAllOpenIssues", mock.Anything, "test-owner", "test-repo").
			Return(allIssues, nil)

		// ラベル付与が失敗
		mockClient.On("AddLabel", mock.Anything, "test-owner", "test-repo", 1, "status:needs-plan").
			Return(errors.New("label add error"))

		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				AutoPlanIssue: true,
			},
		}

		err := executeAutoPlanIfNoActiveIssues(context.Background(), cfg, mockClient, "test-owner", "test-repo", testLogger)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to add status:needs-plan label")
	})
}

func TestFindLowestNumberIssueWithoutStatusLabel(t *testing.T) {
	t.Run("正常系: 最も若い番号のラベルなしIssueを特定", func(t *testing.T) {
		issues := []*github.Issue{
			{
				Number: github.Int(10),
				Labels: []*github.Label{
					{Name: github.String("bug")},
				},
			},
			{
				Number: github.Int(5),
				Labels: []*github.Label{
					{Name: github.String("status:ready")}, // status:*ラベル付き
					{Name: github.String("enhancement")},
				},
			},
			{
				Number: github.Int(3),
				Labels: []*github.Label{
					{Name: github.String("documentation")},
				},
			},
			{
				Number: github.Int(15),
				Labels: []*github.Label{}, // ラベルなし
			},
		}

		result := findLowestNumberIssueWithoutStatusLabel(issues)

		assert.NotNil(t, result)
		assert.Equal(t, 3, *result.Number)
	})

	t.Run("正常系: ラベルなしIssueが存在しない", func(t *testing.T) {
		issues := []*github.Issue{
			{
				Number: github.Int(10),
				Labels: []*github.Label{
					{Name: github.String("status:planning")},
				},
			},
			{
				Number: github.Int(5),
				Labels: []*github.Label{
					{Name: github.String("status:ready")},
				},
			},
		}

		result := findLowestNumberIssueWithoutStatusLabel(issues)

		assert.Nil(t, result)
	})

	t.Run("正常系: 空のIssueリスト", func(t *testing.T) {
		issues := []*github.Issue{}

		result := findLowestNumberIssueWithoutStatusLabel(issues)

		assert.Nil(t, result)
	})
}

func TestHasStatusLabel(t *testing.T) {
	t.Run("正常系: status:*ラベルが存在", func(t *testing.T) {
		issue := &github.Issue{
			Labels: []*github.Label{
				{Name: github.String("bug")},
				{Name: github.String("status:implementing")},
				{Name: github.String("high-priority")},
			},
		}

		result := hasStatusLabel(issue)

		assert.True(t, result)
	})

	t.Run("正常系: status:*ラベルが存在しない", func(t *testing.T) {
		issue := &github.Issue{
			Labels: []*github.Label{
				{Name: github.String("bug")},
				{Name: github.String("enhancement")},
				{Name: github.String("high-priority")},
			},
		}

		result := hasStatusLabel(issue)

		assert.False(t, result)
	})

	t.Run("正常系: ラベルなし", func(t *testing.T) {
		issue := &github.Issue{
			Labels: []*github.Label{},
		}

		result := hasStatusLabel(issue)

		assert.False(t, result)
	})

	t.Run("境界値: nilラベル", func(t *testing.T) {
		issue := &github.Issue{
			Labels: nil,
		}

		result := hasStatusLabel(issue)

		assert.False(t, result)
	})
}

// TestConcurrentAutoPlanExecution は並行実行時の排他制御をテストする
func TestConcurrentAutoPlanExecution(t *testing.T) {
	testLogger, _ := logger.New(logger.WithLevel("debug"))

	t.Run("並行実行時にmutexが排他制御する", func(t *testing.T) {
		mockClient := new(MockGitHubClientForAutoPlan)

		// AddLabel呼び出しの記録用
		var addLabelCalls []int
		var callOrder []string

		// status:*ラベル付きIssueなし
		mockClient.On("ListIssuesByLabels", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "ListIssuesByLabels")
			}).Return([]*github.Issue{}, nil)

		// 全オープンIssueに未ラベルIssueあり
		allIssues := []*github.Issue{
			{
				Number: github.Int(100),
				Title:  github.String("Unlabeled Issue 100"),
				Labels: []*github.Label{},
			},
		}
		mockClient.On("ListAllOpenIssues", mock.Anything, "test-owner", "test-repo").
			Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "ListAllOpenIssues")
			}).Return(allIssues, nil)

		// AddLabel呼び出しの記録用
		mockClient.On("AddLabel", mock.Anything, "test-owner", "test-repo", mock.AnythingOfType("int"), "status:needs-plan").
			Run(func(args mock.Arguments) {
				issueNumber := args.Get(3).(int)
				addLabelCalls = append(addLabelCalls, issueNumber)
				callOrder = append(callOrder, "AddLabel")
			}).Return(nil)

		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				AutoPlanIssue: true,
			},
		}

		// IssueWatcherを作成（autoPlanMu付き）
		watcher, err := NewIssueWatcherWithConfig(
			mockClient,
			"test-owner",
			"test-repo",
			"test-session",
			[]string{"status:needs-plan"},
			time.Second,
			testLogger,
			cfg,
			nil,
		)
		assert.NoError(t, err)

		// 並行実行を開始
		const numGoroutines = 3
		errChan := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				err := watcher.executeAutoPlanWithMutex(context.Background())
				errChan <- err
			}()
		}

		// すべてのgoroutineの完了を待機
		for i := 0; i < numGoroutines; i++ {
			err := <-errChan
			assert.NoError(t, err)
		}

		// 複数の呼び出しがあったことを確認（mutexにより順次実行）
		assert.GreaterOrEqual(t, len(callOrder), 3*3, "各goroutineが順次実行されるべき") // 3つのgoroutine * 最低3回の呼び出し

		// AddLabelが複数回呼ばれた場合（mutex無しの場合）、すべて同じIssueであることを確認
		for _, issueNumber := range addLabelCalls {
			assert.Equal(t, 100, issueNumber, "すべてのラベル付与は同じIssue (#100) に対するもの")
		}
	})

	t.Run("status:needs-plan存在時は並行実行でも処理されない", func(t *testing.T) {
		mockClient := new(MockGitHubClientForAutoPlan)

		// status:needs-planラベル付きIssueが既に存在
		activeIssues := []*github.Issue{
			{
				Number: github.Int(99),
				Title:  github.String("Already needs plan"),
				Labels: []*github.Label{
					{Name: github.String("status:needs-plan")},
				},
			},
		}
		mockClient.On("ListIssuesByLabels", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return(activeIssues, nil)

		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				AutoPlanIssue: true,
			},
		}

		// IssueWatcherを作成（autoPlanMu付き）
		watcher, err := NewIssueWatcherWithConfig(
			mockClient,
			"test-owner",
			"test-repo",
			"test-session",
			[]string{"status:needs-plan"},
			time.Second,
			testLogger,
			cfg,
			nil,
		)
		assert.NoError(t, err)

		// 並行実行を開始
		const numGoroutines = 3
		errChan := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				err := watcher.executeAutoPlanWithMutex(context.Background())
				errChan <- err
			}()
		}

		// すべてのgoroutineの完了を待機
		for i := 0; i < numGoroutines; i++ {
			err := <-errChan
			assert.NoError(t, err)
		}

		// ListAllOpenIssuesやAddLabelは呼ばれないことを確認
		mockClient.AssertNotCalled(t, "ListAllOpenIssues")
		mockClient.AssertNotCalled(t, "AddLabel")
	})
}

// TestExecuteAutoPlanWithOptimisticLock は楽観的ロック機能のテスト
func TestExecuteAutoPlanWithOptimisticLock(t *testing.T) {
	testLogger, _ := logger.New(logger.WithLevel("debug"))

	t.Run("正常系: 楽観的ロックによる競合検出と成功", func(t *testing.T) {
		mockClient := new(MockGitHubClientForAutoPlan)

		// 最初のチェック: status:*ラベル付きIssueなし
		mockClient.On("ListIssuesByLabels", mock.Anything, "test-owner", "test-repo",
			[]string{"status:needs-plan", "status:planning", "status:ready", "status:implementing", "status:review-requested", "status:reviewing", "status:lgtm", "status:requires-changes", "status:revising"}).
			Return([]*github.Issue{}, nil).Once()

		// オープンIssueにラベルなしIssueが存在
		allIssues := []*github.Issue{
			{
				Number: github.Int(3),
				Title:  github.String("Test Issue 3"),
				Labels: []*github.Label{}, // ラベルなし
			},
		}
		mockClient.On("ListAllOpenIssues", mock.Anything, "test-owner", "test-repo").
			Return(allIssues, nil)

		// 楽観的ロック: ラベル付与前の再確認（まだアクティブIssueなし）
		mockClient.On("ListIssuesByLabels", mock.Anything, "test-owner", "test-repo",
			[]string{"status:needs-plan", "status:planning", "status:ready", "status:implementing", "status:review-requested", "status:reviewing", "status:lgtm", "status:requires-changes", "status:revising"}).
			Return([]*github.Issue{}, nil).Once()

		// ラベル付与
		mockClient.On("AddLabel", mock.Anything, "test-owner", "test-repo", 3, "status:needs-plan").
			Return(nil)

		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				AutoPlanIssue: true,
			},
		}

		err := executeAutoPlanWithOptimisticLock(context.Background(), cfg, mockClient, "test-owner", "test-repo", testLogger)

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("競合検出: ラベル付与前に他のプロセスが先にラベル付与済み", func(t *testing.T) {
		mockClient := new(MockGitHubClientForAutoPlan)

		// 最初のチェック: status:*ラベル付きIssueなし
		mockClient.On("ListIssuesByLabels", mock.Anything, "test-owner", "test-repo",
			[]string{"status:needs-plan", "status:planning", "status:ready", "status:implementing", "status:review-requested", "status:reviewing", "status:lgtm", "status:requires-changes", "status:revising"}).
			Return([]*github.Issue{}, nil).Once()

		// オープンIssueにラベルなしIssueが存在
		allIssues := []*github.Issue{
			{
				Number: github.Int(5),
				Title:  github.String("Test Issue 5"),
				Labels: []*github.Label{}, // ラベルなし
			},
		}
		mockClient.On("ListAllOpenIssues", mock.Anything, "test-owner", "test-repo").
			Return(allIssues, nil)

		// 楽観的ロック: 再確認時に他のプロセスが先にラベル付与済み
		competingIssue := []*github.Issue{
			{
				Number: github.Int(5),
				Title:  github.String("Test Issue 5"),
				Labels: []*github.Label{
					{Name: github.String("status:needs-plan")}, // 他のプロセスが付与済み
				},
			},
		}
		mockClient.On("ListIssuesByLabels", mock.Anything, "test-owner", "test-repo",
			[]string{"status:needs-plan", "status:planning", "status:ready", "status:implementing", "status:review-requested", "status:reviewing", "status:lgtm", "status:requires-changes", "status:revising"}).
			Return(competingIssue, nil).Once()

		// AddLabelは呼ばれない（競合検出でスキップ）

		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				AutoPlanIssue: true,
			},
		}

		err := executeAutoPlanWithOptimisticLock(context.Background(), cfg, mockClient, "test-owner", "test-repo", testLogger)

		// 競合検出は正常な動作なのでエラーではない
		assert.NoError(t, err)
		mockClient.AssertNotCalled(t, "AddLabel")
	})

	t.Run("異常系: GitHub API呼び出し失敗時のリトライ", func(t *testing.T) {
		mockClient := new(MockGitHubClientForAutoPlan)

		// 最初の呼び出しは失敗
		mockClient.On("ListIssuesByLabels", mock.Anything, "test-owner", "test-repo",
			[]string{"status:needs-plan", "status:planning", "status:ready", "status:implementing", "status:review-requested", "status:reviewing", "status:lgtm", "status:requires-changes", "status:revising"}).
			Return(nil, errors.New("API error")).Once()

		// リトライ後は成功
		mockClient.On("ListIssuesByLabels", mock.Anything, "test-owner", "test-repo",
			[]string{"status:needs-plan", "status:planning", "status:ready", "status:implementing", "status:review-requested", "status:reviewing", "status:lgtm", "status:requires-changes", "status:revising"}).
			Return([]*github.Issue{}, nil).Once()

		allIssues := []*github.Issue{
			{
				Number: github.Int(1),
				Title:  github.String("Test Issue 1"),
				Labels: []*github.Label{},
			},
		}
		mockClient.On("ListAllOpenIssues", mock.Anything, "test-owner", "test-repo").
			Return(allIssues, nil)

		// 楽観的ロック再確認
		mockClient.On("ListIssuesByLabels", mock.Anything, "test-owner", "test-repo",
			[]string{"status:needs-plan", "status:planning", "status:ready", "status:implementing", "status:review-requested", "status:reviewing", "status:lgtm", "status:requires-changes", "status:revising"}).
			Return([]*github.Issue{}, nil).Once()

		mockClient.On("AddLabel", mock.Anything, "test-owner", "test-repo", 1, "status:needs-plan").
			Return(nil)

		cfg := &config.Config{
			GitHub: config.GitHubConfig{
				AutoPlanIssue: true,
			},
		}

		err := executeAutoPlanWithOptimisticLockWithRetry(context.Background(), cfg, mockClient, "test-owner", "test-repo", testLogger)

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})
}

// TestRaceConditionError は新しい競合状態エラーのテスト
func TestRaceConditionError(t *testing.T) {
	t.Run("RaceConditionError構造体のテスト", func(t *testing.T) {
		err := &RaceConditionError{
			Type:        "optimistic_lock_failure",
			Message:     "race condition detected during label assignment",
			IssueNumber: github.Int(123),
			Timestamp:   time.Now(),
		}

		assert.Contains(t, err.Error(), "race condition detected")
		assert.Contains(t, err.Error(), "#123")
		assert.NotNil(t, err.Timestamp)
	})
}

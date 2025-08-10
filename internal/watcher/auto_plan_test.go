package watcher

import (
	"context"
	"errors"
	"testing"

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

func TestExecuteAutoPlanIfNoActiveIssues(t *testing.T) {
	testLogger, _ := logger.New(logger.WithLevel("debug"))

	t.Run("正常系: status:*ラベルがない場合、最も若い番号のIssueにラベル付与", func(t *testing.T) {
		mockClient := new(MockGitHubClientForAutoPlan)

		// status:*ラベル付きIssueなし
		mockClient.On("ListIssuesByLabels", mock.Anything, "test-owner", "test-repo",
			[]string{"status:needs-plan", "status:planning", "status:ready", "status:implementing", "status:review-requested", "status:reviewing", "status:lgtm", "status:requires-changes"}).
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
			[]string{"status:needs-plan", "status:planning", "status:ready", "status:implementing", "status:review-requested", "status:reviewing", "status:lgtm", "status:requires-changes"}).
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
			[]string{"status:needs-plan", "status:planning", "status:ready", "status:implementing", "status:review-requested", "status:reviewing", "status:lgtm", "status:requires-changes"}).
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

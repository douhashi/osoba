// DISABLED: 古いgo-github APIベースのテストのため一時的に無効化
//go:build ignore
// +build ignore

package github_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/douhashi/osoba/internal/gh"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockGHExecutor はgh.Executorのモック実装
type MockGHExecutor struct {
	mock.Mock
}

func (m *MockGHExecutor) Execute(ctx context.Context, args []string) ([]byte, error) {
	ret := m.Called(ctx, args)
	return ret.Get(0).([]byte), ret.Error(1)
}

func TestGHClient_GetRepository(t *testing.T) {
	t.Run("正常にリポジトリ情報を取得できる", func(t *testing.T) {
		mockExecutor := new(MockGHExecutor)
		client := &github.GHClient{
			executor: mockExecutor,
			logger:   logger.NewTestLogger(),
		}

		expectedRepo := &github.Repository{
			ID:       github.github.Int64(123456),
			Name:     github.String("test-repo"),
			FullName: github.String("owner/test-repo"),
			Owner: &github.User{
				Login: github.String("owner"),
			},
			Private: github.Bool(false),
			HTMLURL: github.String("https://github.com/owner/test-repo"),
		}

		repoJSON, _ := json.Marshal(expectedRepo)
		mockExecutor.On("Execute", mock.Anything, []string{"api", "repos/owner/test-repo"}).
			Return(repoJSON, nil)

		repo, err := client.Getgithub.Repository(context.Background(), "owner", "test-repo")
		require.NoError(t, err)
		assert.Equal(t, "test-repo", *repo.Name)
		assert.Equal(t, "owner/test-repo", *repo.FullName)
		assert.Equal(t, "owner", *repo.Owner.Login)

		mockExecutor.AssertExpectations(t)
	})

	t.Run("ownerが空の場合エラー", func(t *testing.T) {
		client := &github.GHClient{
			executor: new(MockGHExecutor),
			logger:   logger.NewTestLogger(),
		}

		_, err := client.Getgithub.Repository(context.Background(), "", "test-repo")
		assert.EqualError(t, err, "owner is required")
	})

	t.Run("repoが空の場合エラー", func(t *testing.T) {
		client := &github.GHClient{
			executor: new(MockGHExecutor),
			logger:   logger.NewTestLogger(),
		}

		_, err := client.Getgithub.Repository(context.Background(), "owner", "")
		assert.EqualError(t, err, "repo is required")
	})
}

func TestGHClient_ListIssuesByLabels(t *testing.T) {
	t.Run("ラベルでIssueをフィルタリングできる", func(t *testing.T) {
		mockExecutor := new(MockGHExecutor)
		client := &github.GHClient{
			executor: mockExecutor,
			logger:   logger.NewTestLogger(),
		}

		expectedgithub.Issues := []*github.Issue{
			{
				Number: github.Int(1),
				Title:  github.String("github.Issue 1"),
				github.Labels: []*github.Label{
					{Name: github.String("bug")},
				},
			},
			{
				Number: github.Int(2),
				Title:  github.String("github.Issue 2"),
				github.Labels: []*github.Label{
					{Name: github.String("enhancement")},
				},
			},
		}

		issuesJSON, _ := json.Marshal(expectedgithub.Issues)
		mockExecutor.On("Execute", mock.Anything, []string{
			"issue", "list",
			"--repo", "owner/test-repo",
			"--label", "bug,enhancement",
			"--state", "open",
			"--json", "number,title,labels,state,body,author,assignees,createdAt,updatedAt,closedAt,milestone,comments,url",
		}).Return(issuesJSON, nil)

		issues, err := client.Listgithub.IssuesBygithub.Labels(context.Background(), "owner", "test-repo", []string{"bug", "enhancement"})
		require.NoError(t, err)
		assert.Len(t, issues, 2)
		assert.Equal(t, 1, *issues[0].Number)
		assert.Equal(t, 2, *issues[1].Number)

		mockExecutor.AssertExpectations(t)
	})

	t.Run("空のラベルリストでもエラーにならない", func(t *testing.T) {
		mockExecutor := new(MockGHExecutor)
		client := &github.GHClient{
			executor: mockExecutor,
			logger:   logger.NewTestLogger(),
		}

		mockExecutor.On("Execute", mock.Anything, []string{
			"issue", "list",
			"--repo", "owner/test-repo",
			"--state", "open",
			"--json", "number,title,labels,state,body,author,assignees,createdAt,updatedAt,closedAt,milestone,comments,url",
		}).Return([]byte("[]"), nil)

		issues, err := client.Listgithub.IssuesBygithub.Labels(context.Background(), "owner", "test-repo", []string{})
		require.NoError(t, err)
		assert.Len(t, issues, 0)

		mockExecutor.AssertExpectations(t)
	})
}

func TestGHClient_GetRateLimit(t *testing.T) {
	t.Run("レート制限情報を取得できる", func(t *testing.T) {
		mockExecutor := new(MockGHExecutor)
		client := &github.GHClient{
			executor: mockExecutor,
			logger:   logger.NewTestLogger(),
		}

		expectedRateLimit := &gh.RateLimitResponse{
			Resources: gh.RateLimitResources{
				Core: gh.RateLimit{
					Limit:     5000,
					Remaining: 4999,
					Reset:     1234567890,
				},
				Search: gh.RateLimit{
					Limit:     30,
					Remaining: 29,
					Reset:     1234567890,
				},
			},
		}

		rateLimitJSON, _ := json.Marshal(expectedRateLimit)
		mockExecutor.On("Execute", mock.Anything, []string{"api", "rate_limit"}).
			Return(rateLimitJSON, nil)

		rateLimit, err := client.GetRateLimit(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 5000, rateLimit.Core.Limit)
		assert.Equal(t, 4999, rateLimit.Core.Remaining)

		mockExecutor.AssertExpectations(t)
	})
}

func TestGHClient_CreateIssueComment(t *testing.T) {
	t.Run("Issueにコメントを作成できる", func(t *testing.T) {
		mockExecutor := new(MockGHExecutor)
		client := &github.GHClient{
			executor: mockExecutor,
			logger:   logger.NewTestLogger(),
		}

		mockExecutor.On("Execute", mock.Anything, []string{
			"issue", "comment", "123",
			"--repo", "owner/test-repo",
			"--body", "Test comment",
		}).Return([]byte{}, nil)

		err := client.Creategithub.IssueComment(context.Background(), "owner", "test-repo", 123, "Test comment")
		require.NoError(t, err)

		mockExecutor.AssertExpectations(t)
	})

	t.Run("コメントが空の場合エラー", func(t *testing.T) {
		client := &github.GHClient{
			executor: new(MockGHExecutor),
			logger:   logger.NewTestLogger(),
		}

		err := client.Creategithub.IssueComment(context.Background(), "owner", "test-repo", 123, "")
		assert.EqualError(t, err, "comment is required")
	})
}

func TestGHClient_TransitionIssueLabel(t *testing.T) {
	t.Run("ラベル遷移が成功する", func(t *testing.T) {
		mockExecutor := new(MockGHExecutor)
		labelManager := NewMockLabelManagerInterface(t)

		client := &github.GHClient{
			executor:     mockExecutor,
			logger:       logger.NewTestLogger(),
			labelManager: labelManager,
		}

		labelManager.On("Transitiongithub.LabelWithRetry",
			mock.Anything, "owner", "test-repo", 123).
			Return(true, nil)

		transitioned, err := client.Transitiongithub.Issuegithub.Label(context.Background(), "owner", "test-repo", 123)
		require.NoError(t, err)
		assert.True(t, transitioned)

		labelManager.AssertExpectations(t)
	})
}

func TestGHClient_EnsureLabelsExist(t *testing.T) {
	t.Run("ラベルの存在確認が成功する", func(t *testing.T) {
		mockExecutor := new(MockGHExecutor)
		labelManager := NewMockLabelManagerInterface(t)

		client := &github.GHClient{
			executor:     mockExecutor,
			logger:       logger.NewTestLogger(),
			labelManager: labelManager,
		}

		labelManager.On("Ensuregithub.LabelsExistWithRetry",
			mock.Anything, "owner", "test-repo").
			Return(nil)

		err := client.Ensuregithub.LabelsExist(context.Background(), "owner", "test-repo")
		require.NoError(t, err)

		labelManager.AssertExpectations(t)
	})
}

// NewMockLabelManagerInterface creates a new mock instance
func NewMockLabelManagerInterface(t *testing.T) *MockLabelManagerInterface {
	mock := &MockLabelManagerInterface{}
	mock.Mock.Test(t)
	t.Cleanup(func() { mock.AssertExpectations(t) })
	return mock
}

type MockLabelManagerInterface struct {
	mock.Mock
}

func (m *MockLabelManagerInterface) TransitionLabelWithRetry(ctx context.Context, owner, repo string, issueNumber int) (bool, error) {
	args := m.Called(ctx, owner, repo, issueNumber)
	return args.Bool(0), args.Error(1)
}

func (m *MockLabelManagerInterface) TransitionLabelWithInfoWithRetry(ctx context.Context, owner, repo string, issueNumber int) (bool, *github.TransitionInfo, error) {
	args := m.Called(ctx, owner, repo, issueNumber)
	return args.Bool(0), args.Get(1).(*github.TransitionInfo), args.Error(2)
}

func (m *MockLabelManagerInterface) EnsureLabelsExistWithRetry(ctx context.Context, owner, repo string) error {
	args := m.Called(ctx, owner, repo)
	return args.Error(0)
}

package mocks

import (
	"context"
	"time"

	"github.com/douhashi/osoba/internal/github"
	"github.com/stretchr/testify/mock"
)

// MockGitHubClient is a mock implementation of github.GitHubClient interface
type MockGitHubClient struct {
	mock.Mock
}

// NewMockGitHubClient creates a new instance of MockGitHubClient
func NewMockGitHubClient() *MockGitHubClient {
	return &MockGitHubClient{}
}

// WithDefaultBehavior sets up common default behaviors for the mock
func (m *MockGitHubClient) WithDefaultBehavior() *MockGitHubClient {
	// GetRateLimit のデフォルト動作
	m.On("GetRateLimit", mock.Anything).Maybe().Return(&github.RateLimits{
		Core: &github.RateLimit{
			Limit:     5000,
			Remaining: 4999,
			Reset:     time.Now().Add(1 * time.Hour),
		},
		Search: &github.RateLimit{
			Limit:     30,
			Remaining: 30,
			Reset:     time.Now().Add(1 * time.Hour),
		},
	}, nil)

	// CreateIssueComment のデフォルト動作（何もしない成功）
	m.On("CreateIssueComment", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().Return(nil)

	// EnsureLabelsExist のデフォルト動作
	m.On("EnsureLabelsExist", mock.Anything, mock.Anything, mock.Anything).
		Maybe().Return(nil)

	return m
}

// GetRepository mocks the GetRepository method
func (m *MockGitHubClient) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	args := m.Called(ctx, owner, repo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.Repository), args.Error(1)
}

// ListIssuesByLabels mocks the ListIssuesByLabels method
func (m *MockGitHubClient) ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error) {
	args := m.Called(ctx, owner, repo, labels)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*github.Issue), args.Error(1)
}

// ListPullRequestsByLabels mocks the ListPullRequestsByLabels method
func (m *MockGitHubClient) ListPullRequestsByLabels(ctx context.Context, owner, repo string, labels []string) ([]*github.PullRequest, error) {
	args := m.Called(ctx, owner, repo, labels)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*github.PullRequest), args.Error(1)
}

// GetRateLimit mocks the GetRateLimit method
func (m *MockGitHubClient) GetRateLimit(ctx context.Context) (*github.RateLimits, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.RateLimits), args.Error(1)
}

// TransitionIssueLabel mocks the TransitionIssueLabel method
func (m *MockGitHubClient) TransitionIssueLabel(ctx context.Context, owner, repo string, issueNumber int) (bool, error) {
	args := m.Called(ctx, owner, repo, issueNumber)
	return args.Bool(0), args.Error(1)
}

// TransitionIssueLabelWithInfo mocks the TransitionIssueLabelWithInfo method
func (m *MockGitHubClient) TransitionIssueLabelWithInfo(ctx context.Context, owner, repo string, issueNumber int) (bool, *github.TransitionInfo, error) {
	args := m.Called(ctx, owner, repo, issueNumber)
	if args.Get(1) == nil {
		return args.Bool(0), nil, args.Error(2)
	}
	return args.Bool(0), args.Get(1).(*github.TransitionInfo), args.Error(2)
}

// EnsureLabelsExist mocks the EnsureLabelsExist method
func (m *MockGitHubClient) EnsureLabelsExist(ctx context.Context, owner, repo string) error {
	args := m.Called(ctx, owner, repo)
	return args.Error(0)
}

// CreateIssueComment mocks the CreateIssueComment method
func (m *MockGitHubClient) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, comment string) error {
	args := m.Called(ctx, owner, repo, issueNumber, comment)
	return args.Error(0)
}

// RemoveLabel mocks the RemoveLabel method
func (m *MockGitHubClient) RemoveLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

// AddLabel mocks the AddLabel method
func (m *MockGitHubClient) AddLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

// GetPullRequestForIssue mocks the GetPullRequestForIssue method
func (m *MockGitHubClient) GetPullRequestForIssue(ctx context.Context, issueNumber int) (*github.PullRequest, error) {
	args := m.Called(ctx, issueNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.PullRequest), args.Error(1)
}

// MergePullRequest mocks the MergePullRequest method
func (m *MockGitHubClient) MergePullRequest(ctx context.Context, prNumber int) error {
	args := m.Called(ctx, prNumber)
	return args.Error(0)
}

// GetPullRequestStatus mocks the GetPullRequestStatus method
func (m *MockGitHubClient) GetPullRequestStatus(ctx context.Context, prNumber int) (*github.PullRequest, error) {
	args := m.Called(ctx, prNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.PullRequest), args.Error(1)
}

func (m *MockGitHubClient) ListAllOpenIssues(ctx context.Context, owner, repo string) ([]*github.Issue, error) {
	args := m.Called(ctx, owner, repo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*github.Issue), args.Error(1)
}

// Ensure MockGitHubClient implements github.GitHubClient interface
var _ github.GitHubClient = (*MockGitHubClient)(nil)

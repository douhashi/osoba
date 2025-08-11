package watcher

import (
	"context"
	"errors"
	"testing"

	"github.com/douhashi/osoba/internal/github"
	gh "github.com/douhashi/osoba/internal/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGitHubClient はGitHubクライアントのモック実装
type MockGitHubClient struct {
	mock.Mock
}

func (m *MockGitHubClient) GetRepository(ctx context.Context, owner, repo string) (*gh.Repository, error) {
	args := m.Called(ctx, owner, repo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gh.Repository), args.Error(1)
}

func (m *MockGitHubClient) ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*gh.Issue, error) {
	args := m.Called(ctx, owner, repo, labels)
	return args.Get(0).([]*gh.Issue), args.Error(1)
}

func (m *MockGitHubClient) GetRateLimit(ctx context.Context) (*gh.RateLimits, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gh.RateLimits), args.Error(1)
}

func (m *MockGitHubClient) TransitionIssueLabel(ctx context.Context, owner, repo string, issueNumber int) (bool, error) {
	args := m.Called(ctx, owner, repo, issueNumber)
	return args.Bool(0), args.Error(1)
}

func (m *MockGitHubClient) TransitionIssueLabelWithInfo(ctx context.Context, owner, repo string, issueNumber int) (bool, *gh.TransitionInfo, error) {
	args := m.Called(ctx, owner, repo, issueNumber)
	if args.Get(1) == nil {
		return args.Bool(0), nil, args.Error(2)
	}
	return args.Bool(0), args.Get(1).(*gh.TransitionInfo), args.Error(2)
}

func (m *MockGitHubClient) EnsureLabelsExist(ctx context.Context, owner, repo string) error {
	args := m.Called(ctx, owner, repo)
	return args.Error(0)
}

func (m *MockGitHubClient) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, comment string) error {
	args := m.Called(ctx, owner, repo, issueNumber, comment)
	return args.Error(0)
}

func (m *MockGitHubClient) RemoveLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

func (m *MockGitHubClient) AddLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

func (m *MockGitHubClient) GetPullRequestForIssue(ctx context.Context, issueNumber int) (*github.PullRequest, error) {
	args := m.Called(ctx, issueNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.PullRequest), args.Error(1)
}

func (m *MockGitHubClient) MergePullRequest(ctx context.Context, prNumber int) error {
	args := m.Called(ctx, prNumber)
	return args.Error(0)
}

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

func (m *MockGitHubClient) ListPullRequestsByLabels(ctx context.Context, owner, repo string, labels []string) ([]*github.PullRequest, error) {
	args := m.Called(ctx, owner, repo, labels)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*github.PullRequest), args.Error(1)
}

func (m *MockGitHubClient) GetClosingIssueNumber(ctx context.Context, prNumber int) (int, error) {
	args := m.Called(ctx, prNumber)
	return args.Int(0), args.Error(1)
}

func TestExecuteLabelTransition(t *testing.T) {
	tests := []struct {
		name          string
		issue         *gh.Issue
		setupMock     func(*MockGitHubClient)
		expectedError string
	}{
		{
			name: "needs-plan to planning transition",
			issue: &gh.Issue{
				Number: intPtr(123),
				Labels: []*gh.Label{
					{Name: stringPtr("status:needs-plan")},
				},
			},
			setupMock: func(m *MockGitHubClient) {
				m.On("RemoveLabel", mock.Anything, "owner", "repo", 123, "status:needs-plan").Return(nil)
				m.On("AddLabel", mock.Anything, "owner", "repo", 123, "status:planning").Return(nil)
			},
		},
		{
			name: "ready to implementing transition",
			issue: &gh.Issue{
				Number: intPtr(456),
				Labels: []*gh.Label{
					{Name: stringPtr("status:ready")},
				},
			},
			setupMock: func(m *MockGitHubClient) {
				m.On("RemoveLabel", mock.Anything, "owner", "repo", 456, "status:ready").Return(nil)
				m.On("AddLabel", mock.Anything, "owner", "repo", 456, "status:implementing").Return(nil)
			},
		},
		{
			name: "review-requested to reviewing transition",
			issue: &gh.Issue{
				Number: intPtr(789),
				Labels: []*gh.Label{
					{Name: stringPtr("status:review-requested")},
				},
			},
			setupMock: func(m *MockGitHubClient) {
				m.On("RemoveLabel", mock.Anything, "owner", "repo", 789, "status:review-requested").Return(nil)
				m.On("AddLabel", mock.Anything, "owner", "repo", 789, "status:reviewing").Return(nil)
			},
		},
		{
			name: "no transition needed - no matching labels",
			issue: &gh.Issue{
				Number: intPtr(999),
				Labels: []*gh.Label{
					{Name: stringPtr("status:completed")},
				},
			},
			setupMock: func(m *MockGitHubClient) {
				// No mock expectations - no API calls should be made
			},
		},
		{
			name:  "nil issue",
			issue: nil,
			setupMock: func(m *MockGitHubClient) {
				// No mock expectations
			},
			expectedError: "invalid issue",
		},
		{
			name: "issue with nil number",
			issue: &gh.Issue{
				Number: nil,
			},
			setupMock: func(m *MockGitHubClient) {
				// No mock expectations
			},
			expectedError: "invalid issue",
		},
		{
			name: "remove label fails",
			issue: &gh.Issue{
				Number: intPtr(123),
				Labels: []*gh.Label{
					{Name: stringPtr("status:needs-plan")},
				},
			},
			setupMock: func(m *MockGitHubClient) {
				// リトライメカニズムのため、3回呼ばれることを期待
				m.On("RemoveLabel", mock.Anything, "owner", "repo", 123, "status:needs-plan").Return(errors.New("API error")).Times(3)
			},
			expectedError: "failed to remove label status:needs-plan (attempt 3/3): API error",
		},
		{
			name: "add label fails",
			issue: &gh.Issue{
				Number: intPtr(123),
				Labels: []*gh.Label{
					{Name: stringPtr("status:needs-plan")},
				},
			},
			setupMock: func(m *MockGitHubClient) {
				m.On("RemoveLabel", mock.Anything, "owner", "repo", 123, "status:needs-plan").Return(nil)
				// リトライメカニズムのため、3回呼ばれることを期待
				m.On("AddLabel", mock.Anything, "owner", "repo", 123, "status:planning").Return(errors.New("API error")).Times(3)
			},
			expectedError: "failed to add label status:planning (attempt 3/3): API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockGitHubClient)
			tt.setupMock(mockClient)

			watcher := &IssueWatcher{
				client: mockClient,
				owner:  "owner",
				repo:   "repo",
				logger: NewMockLogger(),
			}

			err := watcher.executeLabelTransition(context.Background(), tt.issue)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

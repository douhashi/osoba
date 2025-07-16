package watcher

import (
	"bytes"
	"context"
	"log"
	"testing"

	"github.com/douhashi/osoba/internal/github"
	gh "github.com/google/go-github/v67/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGitHubClientWithInfo is a mock for testing the new logging
type MockGitHubClientWithInfo struct {
	mock.Mock
}

func (m *MockGitHubClientWithInfo) GetRepository(ctx context.Context, owner, repo string) (*gh.Repository, error) {
	args := m.Called(ctx, owner, repo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gh.Repository), args.Error(1)
}

func (m *MockGitHubClientWithInfo) ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*gh.Issue, error) {
	args := m.Called(ctx, owner, repo, labels)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*gh.Issue), args.Error(1)
}

func (m *MockGitHubClientWithInfo) GetRateLimit(ctx context.Context) (*gh.RateLimits, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gh.RateLimits), args.Error(1)
}

func (m *MockGitHubClientWithInfo) TransitionIssueLabel(ctx context.Context, owner, repo string, issueNumber int) (bool, error) {
	args := m.Called(ctx, owner, repo, issueNumber)
	return args.Bool(0), args.Error(1)
}

func (m *MockGitHubClientWithInfo) TransitionIssueLabelWithInfo(ctx context.Context, owner, repo string, issueNumber int) (bool, *github.TransitionInfo, error) {
	args := m.Called(ctx, owner, repo, issueNumber)
	if args.Get(1) == nil {
		return args.Bool(0), nil, args.Error(2)
	}
	return args.Bool(0), args.Get(1).(*github.TransitionInfo), args.Error(2)
}

func (m *MockGitHubClientWithInfo) EnsureLabelsExist(ctx context.Context, owner, repo string) error {
	args := m.Called(ctx, owner, repo)
	return args.Error(0)
}

func (m *MockGitHubClientWithInfo) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, comment string) error {
	args := m.Called(ctx, owner, repo, issueNumber, comment)
	return args.Error(0)
}

func (m *MockGitHubClientWithInfo) RemoveLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

func (m *MockGitHubClientWithInfo) AddLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

func TestIssueWatcher_LogsDetailedTransitionInfo(t *testing.T) {
	ctx := context.Background()

	// Capture log output
	var logBuf bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(originalOutput)

	// Create mock client
	mockClient := new(MockGitHubClientWithInfo)

	// Setup expectations
	issueNumber := 123
	issueTitle := "Test Issue"
	issues := []*gh.Issue{
		{
			Number: &issueNumber,
			Title:  &issueTitle,
			Labels: []*gh.Label{
				{Name: gh.String("status:needs-plan")},
			},
		},
	}

	mockClient.On("ListIssuesByLabels", ctx, "test-owner", "test-repo", []string{"status:needs-plan"}).
		Return(issues, nil)

	// Issue #75の修正により、Issue検知時にはラベル遷移を実行しないため、
	// TransitionIssueLabelWithInfoの呼び出しは期待しない

	// Create watcher
	watcher := &IssueWatcher{
		client:      mockClient,
		owner:       "test-owner",
		repo:        "test-repo",
		labels:      []string{"status:needs-plan"},
		issueLabels: make(map[int64][]string),
	}

	// Execute
	var callbackCalled bool
	callback := func(issue *gh.Issue) {
		callbackCalled = true
	}

	watcher.checkIssues(ctx, callback)

	// Verify
	assert.True(t, callbackCalled)

	// Check log output - Issue検知時にはラベル遷移ログは出力されない
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "Issue #123 - Test Issue (labels: [status:needs-plan]) - Process: true",
		"Log should contain issue detection info")

	// Issue #75の修正により、Issue検知時にはラベル遷移は実行されない
	assert.NotContains(t, logOutput, "Issue #123: status:needs-plan → status:planning",
		"Log should NOT contain label transition info during issue detection")

	mockClient.AssertExpectations(t)
}

func TestIssueWatcher_LogsFailedTransition(t *testing.T) {
	ctx := context.Background()

	// Capture log output
	var logBuf bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(originalOutput)

	// Create mock client
	mockClient := new(MockGitHubClientWithInfo)

	// Setup expectations
	issueNumber := 456
	issueTitle := "Another Test Issue"
	issues := []*gh.Issue{
		{
			Number: &issueNumber,
			Title:  &issueTitle,
			Labels: []*gh.Label{
				{Name: gh.String("status:ready")},
			},
		},
	}

	mockClient.On("ListIssuesByLabels", ctx, "test-owner", "test-repo", []string{"status:ready"}).
		Return(issues, nil)

	// Issue #75の修正により、Issue検知時にはラベル遷移を実行しないため、
	// TransitionIssueLabelWithInfoの呼び出しは期待しない

	// Create watcher
	watcher := &IssueWatcher{
		client:      mockClient,
		owner:       "test-owner",
		repo:        "test-repo",
		labels:      []string{"status:ready"},
		issueLabels: make(map[int64][]string),
	}

	// Execute
	callback := func(issue *gh.Issue) {}
	watcher.checkIssues(ctx, callback)

	// Check log output - Issue検知時にはラベル遷移失敗ログは出力されない
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "Issue #456 - Another Test Issue (labels: [status:ready]) - Process: true",
		"Log should contain issue detection info")

	// Issue #75の修正により、Issue検知時にはラベル遷移は実行されないため失敗ログも出力されない
	assert.NotContains(t, logOutput, "Failed to transition label for issue #456",
		"Log should NOT contain label transition failure message during issue detection")

	mockClient.AssertExpectations(t)
}

func TestIssueWatcher_NoTransitionNeeded(t *testing.T) {
	ctx := context.Background()

	// Capture log output
	var logBuf bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(originalOutput)

	// Create mock client
	mockClient := new(MockGitHubClientWithInfo)

	// Setup expectations
	issueNumber := 789
	issueTitle := "Already in progress"
	issues := []*gh.Issue{
		{
			Number: &issueNumber,
			Title:  &issueTitle,
			Labels: []*gh.Label{
				{Name: gh.String("status:planning")},
			},
		},
	}

	mockClient.On("ListIssuesByLabels", ctx, "test-owner", "test-repo", []string{"status:needs-plan"}).
		Return(issues, nil)

	// Create watcher - watching for trigger labels
	watcher := &IssueWatcher{
		client:      mockClient,
		owner:       "test-owner",
		repo:        "test-repo",
		labels:      []string{"status:needs-plan"},
		issueLabels: make(map[int64][]string),
	}

	// Execute
	callback := func(issue *gh.Issue) {}
	watcher.checkIssues(ctx, callback)

	// Check log output - should not contain transition log
	logOutput := logBuf.String()
	assert.NotContains(t, logOutput, "→",
		"Log should not contain transition arrow when no transition happens")

	mockClient.AssertExpectations(t)
}

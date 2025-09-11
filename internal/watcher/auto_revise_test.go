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

// MockGitHubClientForAutoRevise はテスト用のGitHubClientモック
type MockGitHubClientForAutoRevise struct {
	mock.Mock
}

func (m *MockGitHubClientForAutoRevise) ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error) {
	args := m.Called(ctx, owner, repo, labels)
	return args.Get(0).([]*github.Issue), args.Error(1)
}

func (m *MockGitHubClientForAutoRevise) ListPullRequestsByLabels(ctx context.Context, owner, repo string, labels []string) ([]*github.PullRequest, error) {
	args := m.Called(ctx, owner, repo, labels)
	return args.Get(0).([]*github.PullRequest), args.Error(1)
}

func (m *MockGitHubClientForAutoRevise) GetClosingIssueNumber(ctx context.Context, prNumber int) (int, error) {
	args := m.Called(ctx, prNumber)
	return args.Int(0), args.Error(1)
}

func (m *MockGitHubClientForAutoRevise) ListClosedIssues(ctx context.Context, owner, repo string) ([]*github.Issue, error) {
	args := m.Called(ctx, owner, repo)
	if issues := args.Get(0); issues != nil {
		return issues.([]*github.Issue), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockGitHubClientForAutoRevise) GetPullRequestForIssue(ctx context.Context, issueNumber int) (*github.PullRequest, error) {
	args := m.Called(ctx, issueNumber)
	if pr := args.Get(0); pr != nil {
		return pr.(*github.PullRequest), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockGitHubClientForAutoRevise) MergePullRequest(ctx context.Context, prNumber int) error {
	args := m.Called(ctx, prNumber)
	return args.Error(0)
}

func (m *MockGitHubClientForAutoRevise) GetPullRequestStatus(ctx context.Context, prNumber int) (*github.PullRequest, error) {
	args := m.Called(ctx, prNumber)
	if pr := args.Get(0); pr != nil {
		return pr.(*github.PullRequest), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockGitHubClientForAutoRevise) AddLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

func (m *MockGitHubClientForAutoRevise) RemoveLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

func (m *MockGitHubClientForAutoRevise) TransitionLabels(ctx context.Context, owner, repo string, issueNumber int, fromLabel, toLabel string) error {
	args := m.Called(ctx, owner, repo, issueNumber, fromLabel, toLabel)
	return args.Error(0)
}

func (m *MockGitHubClientForAutoRevise) GetRateLimit(ctx context.Context) (*github.RateLimits, error) {
	args := m.Called(ctx)
	if rl := args.Get(0); rl != nil {
		return rl.(*github.RateLimits), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockGitHubClientForAutoRevise) EnsureLabelsExist(ctx context.Context, owner, repo string) error {
	args := m.Called(ctx, owner, repo)
	return args.Error(0)
}

func (m *MockGitHubClientForAutoRevise) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, comment string) error {
	args := m.Called(ctx, owner, repo, issueNumber, comment)
	return args.Error(0)
}

func (m *MockGitHubClientForAutoRevise) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	args := m.Called(ctx, owner, repo)
	if r := args.Get(0); r != nil {
		return r.(*github.Repository), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockGitHubClientForAutoRevise) ListAllOpenIssues(ctx context.Context, owner, repo string) ([]*github.Issue, error) {
	args := m.Called(ctx, owner, repo)
	return args.Get(0).([]*github.Issue), args.Error(1)
}

func (m *MockGitHubClientForAutoRevise) TransitionIssueLabel(ctx context.Context, owner, repo string, issueNumber int) (bool, error) {
	args := m.Called(ctx, owner, repo, issueNumber)
	return args.Bool(0), args.Error(1)
}

func (m *MockGitHubClientForAutoRevise) TransitionIssueLabelWithInfo(ctx context.Context, owner, repo string, issueNumber int) (bool, *github.TransitionInfo, error) {
	args := m.Called(ctx, owner, repo, issueNumber)
	if info := args.Get(1); info != nil {
		return args.Bool(0), info.(*github.TransitionInfo), args.Error(2)
	}
	return args.Bool(0), nil, args.Error(2)
}

// MockActionManagerForAutoRevise はテスト用のActionManagerモック
type MockActionManagerForAutoRevise struct {
	mock.Mock
}

func (m *MockActionManagerForAutoRevise) ExecuteAction(ctx context.Context, issue *github.Issue) error {
	args := m.Called(ctx, issue)
	return args.Error(0)
}

func (m *MockActionManagerForAutoRevise) GetActionForIssue(issue *github.Issue) ActionExecutor {
	args := m.Called(issue)
	if action := args.Get(0); action != nil {
		return action.(ActionExecutor)
	}
	return nil
}

func (m *MockActionManagerForAutoRevise) SetActionFactory(factory ActionFactory) {
	m.Called(factory)
}

// TestHasRequiresChangesLabel tests the hasRequiresChangesLabel function
func TestHasRequiresChangesLabel(t *testing.T) {
	tests := []struct {
		name     string
		pr       *github.PullRequest
		expected bool
	}{
		{
			name: "PR exists",
			pr: &github.PullRequest{
				Number: 123,
			},
			expected: true,
		},
		{
			name:     "nil PR",
			pr:       nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasRequiresChangesLabel(tt.pr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExecuteAutoReviseIfRequiresChanges tests the executeAutoReviseIfRequiresChanges function
func TestExecuteAutoReviseIfRequiresChanges(t *testing.T) {
	tests := []struct {
		name               string
		pr                 *github.PullRequest
		autoReviseEnabled  bool
		closingIssueNumber int
		closingIssueError  error
		executeActionError error
		expectError        bool
		expectActionCalled bool
	}{
		{
			name: "successful auto-revise execution",
			pr: &github.PullRequest{
				Number: 123,
			},
			autoReviseEnabled:  true,
			closingIssueNumber: 100,
			closingIssueError:  nil,
			executeActionError: nil,
			expectError:        false,
			expectActionCalled: true,
		},
		{
			name: "auto-revise disabled in config",
			pr: &github.PullRequest{
				Number: 124,
			},
			autoReviseEnabled:  false,
			expectError:        false,
			expectActionCalled: false,
		},
		{
			name: "failed to get closing issue number",
			pr: &github.PullRequest{
				Number: 126,
			},
			autoReviseEnabled:  true,
			closingIssueError:  errors.New("API error"),
			expectError:        true,
			expectActionCalled: false,
		},
		{
			name: "no closing issue found",
			pr: &github.PullRequest{
				Number: 127,
			},
			autoReviseEnabled:  true,
			closingIssueNumber: 0,
			closingIssueError:  nil,
			expectError:        false,
			expectActionCalled: false,
		},
		{
			name: "action execution failed",
			pr: &github.PullRequest{
				Number: 128,
			},
			autoReviseEnabled:  true,
			closingIssueNumber: 101,
			closingIssueError:  nil,
			executeActionError: errors.New("action failed"),
			expectError:        true,
			expectActionCalled: true,
		},
		{
			name:               "nil PR",
			pr:                 nil,
			autoReviseEnabled:  true,
			expectError:        false,
			expectActionCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockGH := new(MockGitHubClientForAutoRevise)
			mockAM := new(MockActionManagerForAutoRevise)

			cfg := &config.Config{
				GitHub: config.GitHubConfig{
					AutoRevisePR: tt.autoReviseEnabled,
					Labels: config.LabelConfig{
						RequiresChanges: "status:requires-changes",
					},
				},
			}

			ctx := context.Background()

			// Setup expectations
			if tt.autoReviseEnabled && hasRequiresChangesLabel(tt.pr) {
				mockGH.On("GetClosingIssueNumber", mock.Anything, tt.pr.Number).
					Return(tt.closingIssueNumber, tt.closingIssueError)

				if tt.closingIssueError == nil && tt.closingIssueNumber > 0 {
					mockAM.On("ExecuteAction", mock.Anything, mock.Anything).
						Return(tt.executeActionError)
				}
			}

			// Execute
			err := executeAutoReviseIfRequiresChanges(ctx, tt.pr, cfg, mockGH, mockAM, "test-session")

			// Assert
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectActionCalled {
				mockAM.AssertCalled(t, "ExecuteAction", mock.Anything, mock.Anything)
			} else {
				mockAM.AssertNotCalled(t, "ExecuteAction", mock.Anything, mock.Anything)
			}

			mockGH.AssertExpectations(t)
			mockAM.AssertExpectations(t)
		})
	}
}

// TestExecuteAutoReviseIfRequiresChangesWithLogger tests the executeAutoReviseIfRequiresChangesWithLogger function
func TestExecuteAutoReviseIfRequiresChangesWithLogger(t *testing.T) {
	tests := []struct {
		name               string
		pr                 *github.PullRequest
		autoReviseEnabled  bool
		closingIssueNumber int
		closingIssueError  error
		executeActionError error
		expectError        bool
	}{
		{
			name: "successful execution with logging",
			pr: &github.PullRequest{
				Number: 200,
			},
			autoReviseEnabled:  true,
			closingIssueNumber: 150,
			closingIssueError:  nil,
			executeActionError: nil,
			expectError:        false,
		},
		{
			name: "disabled feature with logging",
			pr: &github.PullRequest{
				Number: 201,
			},
			autoReviseEnabled: false,
			expectError:       false,
		},
		{
			name:              "nil PR with logging",
			pr:                nil,
			autoReviseEnabled: true,
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockGH := new(MockGitHubClientForAutoRevise)
			mockAM := new(MockActionManagerForAutoRevise)
			log, _ := logger.New(logger.WithLevel("debug"))

			cfg := &config.Config{
				GitHub: config.GitHubConfig{
					AutoRevisePR: tt.autoReviseEnabled,
				},
			}

			ctx := context.Background()

			// Setup expectations
			if tt.autoReviseEnabled && hasRequiresChangesLabel(tt.pr) {
				mockGH.On("GetClosingIssueNumber", mock.Anything, tt.pr.Number).
					Return(tt.closingIssueNumber, tt.closingIssueError)

				if tt.closingIssueError == nil && tt.closingIssueNumber > 0 {
					// ActionManagerはGetActionForIssueでnilを返し、その後ExecuteActionが呼ばれる
					mockAM.On("GetActionForIssue", mock.Anything).
						Return(nil)

					// 直接ReviseActionが作成される
					mockAM.On("ExecuteAction", mock.Anything, mock.Anything).
						Return(tt.executeActionError)
				}
			}

			// Execute
			err := executeAutoReviseIfRequiresChangesWithLogger(ctx, tt.pr, cfg, mockGH, mockAM, "test-session", log)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockGH.AssertExpectations(t)
			mockAM.AssertExpectations(t)
		})
	}
}

package watcher

import (
	"context"
	"errors"
	"testing"

	gh "github.com/douhashi/osoba/internal/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTmuxManager はtmux操作のモック実装
type MockTmuxManager struct {
	mock.Mock
}

func (m *MockTmuxManager) KillWindowsForIssue(sessionName string, issueNumber int) error {
	args := m.Called(sessionName, issueNumber)
	return args.Error(0)
}

func TestExecuteLabelTransition_RequiresChanges(t *testing.T) {
	tests := []struct {
		name          string
		issue         *gh.Issue
		sessionName   string
		setupMock     func(*MockGitHubClient)
		expectedError string
	}{
		{
			name: "requires-changes to ready transition with tmux cleanup",
			issue: &gh.Issue{
				Number: intPtr(206),
				Labels: []*gh.Label{
					{Name: stringPtr("status:requires-changes")},
				},
			},
			sessionName: "osoba",
			setupMock: func(githubMock *MockGitHubClient) {
				// ラベルの遷移
				githubMock.On("TransitionLabels", mock.Anything, "owner", "repo", 206, "status:requires-changes", "status:ready").Return(nil)
			},
		},
		{
			name: "requires-changes transition with empty sessionName",
			issue: &gh.Issue{
				Number: intPtr(208),
				Labels: []*gh.Label{
					{Name: stringPtr("status:requires-changes")},
				},
			},
			sessionName: "",
			setupMock: func(githubMock *MockGitHubClient) {
				// sessionNameが空の場合、tmux削除はスキップされる
				// ラベルの遷移は実行される
				githubMock.On("TransitionLabels", mock.Anything, "owner", "repo", 208, "status:requires-changes", "status:ready").Return(nil)
			},
		},
		{
			name: "requires-changes transition with label removal failure",
			issue: &gh.Issue{
				Number: intPtr(209),
				Labels: []*gh.Label{
					{Name: stringPtr("status:requires-changes")},
				},
			},
			sessionName: "osoba",
			setupMock: func(githubMock *MockGitHubClient) {
				// ラベル削除が失敗（リトライ3回）
				githubMock.On("RemoveLabel", mock.Anything, "owner", "repo", 209, "status:requires-changes").Return(errors.New("API error")).Times(3)
			},
			expectedError: "failed to remove label status:requires-changes (attempt 3/3): API error",
		},
		{
			name: "requires-changes transition with label addition failure",
			issue: &gh.Issue{
				Number: intPtr(210),
				Labels: []*gh.Label{
					{Name: stringPtr("status:requires-changes")},
				},
			},
			sessionName: "osoba",
			setupMock: func(githubMock *MockGitHubClient) {
				// ラベル削除は成功
				githubMock.On("RemoveLabel", mock.Anything, "owner", "repo", 210, "status:requires-changes").Return(nil)
				// ラベル追加が失敗（リトライ3回）
				githubMock.On("AddLabel", mock.Anything, "owner", "repo", 210, "status:ready").Return(errors.New("API error")).Times(3)
			},
			expectedError: "failed to add label status:ready (attempt 3/3): API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGitHub := new(MockGitHubClient)
			tt.setupMock(mockGitHub)

			watcher := &IssueWatcher{
				client:      mockGitHub,
				owner:       "owner",
				repo:        "repo",
				sessionName: tt.sessionName,
				logger:      NewMockLogger(),
			}

			err := watcher.executeLabelTransition(context.Background(), tt.issue)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockGitHub.AssertExpectations(t)
		})
	}
}

func TestExecuteLabelTransition_MixedTransitions(t *testing.T) {
	// 複数のラベルが同時に存在する場合のテスト
	tests := []struct {
		name          string
		issue         *gh.Issue
		sessionName   string
		setupMock     func(*MockGitHubClient)
		expectedError string
	}{
		{
			name: "requires-changes takes priority over other transitions",
			issue: &gh.Issue{
				Number: intPtr(211),
				Labels: []*gh.Label{
					{Name: stringPtr("status:requires-changes")},
					{Name: stringPtr("status:needs-plan")}, // これは無視される
				},
			},
			sessionName: "osoba",
			setupMock: func(githubMock *MockGitHubClient) {
				// requires-changesの遷移のみが実行される
				githubMock.On("RemoveLabel", mock.Anything, "owner", "repo", 211, "status:requires-changes").Return(nil)
				githubMock.On("AddLabel", mock.Anything, "owner", "repo", 211, "status:ready").Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGitHub := new(MockGitHubClient)
			tt.setupMock(mockGitHub)

			watcher := &IssueWatcher{
				client:      mockGitHub,
				owner:       "owner",
				repo:        "repo",
				sessionName: tt.sessionName,
				logger:      NewMockLogger(),
			}

			err := watcher.executeLabelTransition(context.Background(), tt.issue)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockGitHub.AssertExpectations(t)
		})
	}
}

package watcher

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockGitHubClientForTransition はテスト用のモッククライアント
type mockGitHubClientForTransition struct {
	mock.Mock
	github.GitHubClient
}

func (m *mockGitHubClientForTransition) RemoveLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

func (m *mockGitHubClientForTransition) AddLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

func (m *mockGitHubClientForTransition) TransitionLabels(ctx context.Context, owner, repo string, issueNumber int, removeLabel, addLabel string) error {
	args := m.Called(ctx, owner, repo, issueNumber, removeLabel, addLabel)
	return args.Error(0)
}

func TestExecuteLabelTransition_Enhanced(t *testing.T) {
	ctx := context.Background()
	log, _ := logger.New(logger.WithLevel("debug"))

	tests := []struct {
		name          string
		issue         *github.Issue
		setupMock     func(*mockGitHubClientForTransition)
		expectedError string
	}{
		{
			name:  "nilのIssueの場合はエラー",
			issue: nil,
			setupMock: func(m *mockGitHubClientForTransition) {
				// 何も呼ばれないはず
			},
			expectedError: "invalid issue: nil issue or issue number",
		},
		{
			name: "Issue番号がnilの場合はエラー",
			issue: &github.Issue{
				Number: nil,
			},
			setupMock: func(m *mockGitHubClientForTransition) {
				// 何も呼ばれないはず
			},
			expectedError: "invalid issue: nil issue or issue number",
		},
		{
			name:  "needs-planからplanningへの正常な遷移",
			issue: createTestIssueWithLabels([]string{"status:needs-plan", "bug"}),
			setupMock: func(m *mockGitHubClientForTransition) {
				m.On("TransitionLabels", ctx, "owner", "repo", 1, "status:needs-plan", "status:planning").Return(nil)
			},
			expectedError: "",
		},
		{
			name:  "readyからimplementingへの正常な遷移",
			issue: createTestIssueWithLabels([]string{"status:ready", "enhancement"}),
			setupMock: func(m *mockGitHubClientForTransition) {
				m.On("TransitionLabels", ctx, "owner", "repo", 1, "status:ready", "status:implementing").Return(nil)
			},
			expectedError: "",
		},
		{
			name:  "review-requestedからreviewingへの正常な遷移",
			issue: createTestIssueWithLabels([]string{"status:review-requested"}),
			setupMock: func(m *mockGitHubClientForTransition) {
				m.On("TransitionLabels", ctx, "owner", "repo", 1, "status:review-requested", "status:reviewing").Return(nil)
			},
			expectedError: "",
		},
		{
			name:  "トリガーラベルがない場合は何もしない",
			issue: createTestIssueWithLabels([]string{"bug", "enhancement"}),
			setupMock: func(m *mockGitHubClientForTransition) {
				// 何も呼ばれないはず
			},
			expectedError: "",
		},
		{
			name:  "ラベル削除で1回失敗後に成功",
			issue: createTestIssueWithLabels([]string{"status:needs-plan"}),
			setupMock: func(m *mockGitHubClientForTransition) {
				m.On("TransitionLabels", ctx, "owner", "repo", 1, "status:needs-plan", "status:planning").
					Return(errors.New("temporary error")).Once()
				m.On("TransitionLabels", ctx, "owner", "repo", 1, "status:needs-plan", "status:planning").
					Return(nil).Once()
			},
			expectedError: "",
		},
		{
			name:  "ラベル削除で3回失敗",
			issue: createTestIssueWithLabels([]string{"status:needs-plan"}),
			setupMock: func(m *mockGitHubClientForTransition) {
				m.On("TransitionLabels", ctx, "owner", "repo", 1, "status:needs-plan", "status:planning").
					Return(errors.New("persistent error"))
			},
			expectedError: "failed to transition labels from status:needs-plan to status:planning (attempt 3/3): persistent error",
		},
		// TransitionLabelsは原子的操作なので、個別のラベル追加失敗ケースは不要
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockClient := new(mockGitHubClientForTransition)
			tt.setupMock(mockClient)

			watcher := &IssueWatcher{
				client: mockClient,
				owner:  "owner",
				repo:   "repo",
				logger: log.WithFields("test", tt.name),
			}

			// Act
			err := watcher.executeLabelTransition(ctx, tt.issue)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			// モックの呼び出し検証
			mockClient.AssertExpectations(t)
		})
	}
}

func TestExecuteLabelTransition_RetryTiming(t *testing.T) {
	// テストモードではスリープをスキップするためこのテストは実行しない
	if os.Getenv("OSOBA_TEST_MODE") == "true" {
		t.Skip("Skipping retry timing test in test mode")
	}
	
	ctx := context.Background()
	log, _ := logger.New(logger.WithLevel("debug"))

	// Arrange
	mockClient := new(mockGitHubClientForTransition)
	issue := createTestIssueWithLabels([]string{"status:needs-plan"})

	// 2回失敗して3回目で成功
	mockClient.On("TransitionLabels", ctx, "owner", "repo", 1, "status:needs-plan", "status:planning").
		Return(errors.New("error 1")).Once()
	mockClient.On("TransitionLabels", ctx, "owner", "repo", 1, "status:needs-plan", "status:planning").
		Return(errors.New("error 2")).Once()
	mockClient.On("TransitionLabels", ctx, "owner", "repo", 1, "status:needs-plan", "status:planning").
		Return(nil).Once()

	watcher := &IssueWatcher{
		client: mockClient,
		owner:  "owner",
		repo:   "repo",
		logger: log,
	}

	// Act
	start := time.Now()
	err := watcher.executeLabelTransition(ctx, issue)
	elapsed := time.Since(start)

	// Assert
	assert.NoError(t, err)
	// リトライのバックオフ時間を確認（1秒 + 2秒 + 実行時間）
	assert.Greater(t, elapsed, 3*time.Second)
	assert.Less(t, elapsed, 5*time.Second)

	mockClient.AssertExpectations(t)
}

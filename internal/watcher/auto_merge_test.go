package watcher

import (
	"context"
	"errors"
	"testing"

	"github.com/douhashi/osoba/internal/cleanup"
	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockGitHubClientForAutoMerge はGitHubClientのモック（自動マージテスト用）
type MockGitHubClientForAutoMerge struct {
	mock.Mock
}

func (m *MockGitHubClientForAutoMerge) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	args := m.Called(ctx, owner, repo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.Repository), args.Error(1)
}

func (m *MockGitHubClientForAutoMerge) ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error) {
	args := m.Called(ctx, owner, repo, labels)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*github.Issue), args.Error(1)
}

func (m *MockGitHubClientForAutoMerge) GetRateLimit(ctx context.Context) (*github.RateLimits, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.RateLimits), args.Error(1)
}

func (m *MockGitHubClientForAutoMerge) TransitionIssueLabel(ctx context.Context, owner, repo string, issueNumber int) (bool, error) {
	args := m.Called(ctx, owner, repo, issueNumber)
	return args.Bool(0), args.Error(1)
}

func (m *MockGitHubClientForAutoMerge) TransitionIssueLabelWithInfo(ctx context.Context, owner, repo string, issueNumber int) (bool, *github.TransitionInfo, error) {
	args := m.Called(ctx, owner, repo, issueNumber)
	if args.Get(1) == nil {
		return args.Bool(0), nil, args.Error(2)
	}
	return args.Bool(0), args.Get(1).(*github.TransitionInfo), args.Error(2)
}

func (m *MockGitHubClientForAutoMerge) EnsureLabelsExist(ctx context.Context, owner, repo string) error {
	args := m.Called(ctx, owner, repo)
	return args.Error(0)
}

func (m *MockGitHubClientForAutoMerge) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, comment string) error {
	args := m.Called(ctx, owner, repo, issueNumber, comment)
	return args.Error(0)
}

func (m *MockGitHubClientForAutoMerge) RemoveLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

func (m *MockGitHubClientForAutoMerge) AddLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

func (m *MockGitHubClientForAutoMerge) GetPullRequestForIssue(ctx context.Context, issueNumber int) (*github.PullRequest, error) {
	args := m.Called(ctx, issueNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.PullRequest), args.Error(1)
}

func (m *MockGitHubClientForAutoMerge) MergePullRequest(ctx context.Context, prNumber int) error {
	args := m.Called(ctx, prNumber)
	return args.Error(0)
}

func (m *MockGitHubClientForAutoMerge) GetPullRequestStatus(ctx context.Context, prNumber int) (*github.PullRequest, error) {
	args := m.Called(ctx, prNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.PullRequest), args.Error(1)
}

// MockCleanupManager はCleanupManagerのモック
type MockCleanupManager struct {
	mock.Mock
}

func (m *MockCleanupManager) CleanupIssueResources(ctx context.Context, issueNumber int) error {
	args := m.Called(ctx, issueNumber)
	return args.Error(0)
}

// cleanup.Managerインターフェースを実装していることを確認
var _ cleanup.Manager = (*MockCleanupManager)(nil)

func TestExecuteAutoMergeIfLGTM(t *testing.T) {
	tests := []struct {
		name          string
		issue         *github.Issue
		config        *config.Config
		prResponse    *github.PullRequest
		prError       error
		mergeError    error
		cleanupError  error
		expectMerge   bool
		expectCleanup bool
		expectError   bool
		errorContains string
	}{
		{
			name: "正常系: status:lgtmラベルでPRを自動マージ",
			issue: &github.Issue{
				Number: github.Int(123),
				Labels: []*github.Label{
					{Name: github.String("status:lgtm")},
				},
			},
			config: &config.Config{
				GitHub: config.GitHubConfig{
					AutoMergeLGTM: true,
				},
			},
			prResponse: &github.PullRequest{
				Number:    456,
				State:     "OPEN",
				Mergeable: "MERGEABLE",
			},
			expectMerge:   true,
			expectCleanup: true,
			expectError:   false,
		},
		{
			name: "正常系: auto_merge_lgtmが無効の場合はスキップ",
			issue: &github.Issue{
				Number: github.Int(123),
				Labels: []*github.Label{
					{Name: github.String("status:lgtm")},
				},
			},
			config: &config.Config{
				GitHub: config.GitHubConfig{
					AutoMergeLGTM: false,
				},
			},
			expectMerge:   false,
			expectCleanup: false,
			expectError:   false,
		},
		{
			name: "正常系: status:lgtmラベルがない場合はスキップ",
			issue: &github.Issue{
				Number: github.Int(123),
				Labels: []*github.Label{
					{Name: github.String("status:ready")},
				},
			},
			config: &config.Config{
				GitHub: config.GitHubConfig{
					AutoMergeLGTM: true,
				},
			},
			expectMerge:   false,
			expectCleanup: false,
			expectError:   false,
		},
		{
			name: "正常系: PRが存在しない場合はスキップ",
			issue: &github.Issue{
				Number: github.Int(123),
				Labels: []*github.Label{
					{Name: github.String("status:lgtm")},
				},
			},
			config: &config.Config{
				GitHub: config.GitHubConfig{
					AutoMergeLGTM: true,
				},
			},
			prResponse:    nil,
			expectMerge:   false,
			expectCleanup: false,
			expectError:   false,
		},
		{
			name: "正常系: PRがマージ不可の場合はスキップ",
			issue: &github.Issue{
				Number: github.Int(123),
				Labels: []*github.Label{
					{Name: github.String("status:lgtm")},
				},
			},
			config: &config.Config{
				GitHub: config.GitHubConfig{
					AutoMergeLGTM: true,
				},
			},
			prResponse: &github.PullRequest{
				Number:    456,
				State:     "OPEN",
				Mergeable: "CONFLICTING",
			},
			expectMerge:   false,
			expectCleanup: false,
			expectError:   false,
		},
		{
			name: "正常系: PRがドラフトの場合はスキップ",
			issue: &github.Issue{
				Number: github.Int(123),
				Labels: []*github.Label{
					{Name: github.String("status:lgtm")},
				},
			},
			config: &config.Config{
				GitHub: config.GitHubConfig{
					AutoMergeLGTM: true,
				},
			},
			prResponse: &github.PullRequest{
				Number:    456,
				State:     "OPEN",
				Mergeable: "MERGEABLE",
				IsDraft:   true,
			},
			expectMerge:   false,
			expectCleanup: false,
			expectError:   false,
		},
		{
			name: "異常系: PR取得エラー",
			issue: &github.Issue{
				Number: github.Int(123),
				Labels: []*github.Label{
					{Name: github.String("status:lgtm")},
				},
			},
			config: &config.Config{
				GitHub: config.GitHubConfig{
					AutoMergeLGTM: true,
				},
			},
			prError:       errors.New("github api error"),
			expectMerge:   false,
			expectCleanup: false,
			expectError:   true,
			errorContains: "failed to get pull request",
		},
		{
			name: "異常系: マージエラー",
			issue: &github.Issue{
				Number: github.Int(123),
				Labels: []*github.Label{
					{Name: github.String("status:lgtm")},
				},
			},
			config: &config.Config{
				GitHub: config.GitHubConfig{
					AutoMergeLGTM: true,
				},
			},
			prResponse: &github.PullRequest{
				Number:    456,
				State:     "OPEN",
				Mergeable: "MERGEABLE",
			},
			mergeError:    errors.New("merge conflict"),
			expectMerge:   true,
			expectCleanup: false,
			expectError:   true,
			errorContains: "failed to merge pull request",
		},
		{
			name: "正常系: クリーンアップエラーは無視される",
			issue: &github.Issue{
				Number: github.Int(123),
				Labels: []*github.Label{
					{Name: github.String("status:lgtm")},
				},
			},
			config: &config.Config{
				GitHub: config.GitHubConfig{
					AutoMergeLGTM: true,
				},
			},
			prResponse: &github.PullRequest{
				Number:    456,
				State:     "OPEN",
				Mergeable: "MERGEABLE",
			},
			cleanupError:  errors.New("cleanup failed"),
			expectMerge:   true,
			expectCleanup: true,
			expectError:   false, // クリーンアップエラーは無視される
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGH := new(MockGitHubClientForAutoMerge)
			mockCleanup := new(MockCleanupManager)

			// モックの設定
			// auto_merge_lgtmが有効でstatus:lgtmラベルがある場合はGetPullRequestForIssueが呼ばれる
			if tt.config != nil && tt.config.GitHub.AutoMergeLGTM && hasLGTMLabel(tt.issue) {
				mockGH.On("GetPullRequestForIssue", mock.Anything, *tt.issue.Number).
					Return(tt.prResponse, tt.prError)
			}

			if tt.expectMerge && tt.prResponse != nil {
				mockGH.On("MergePullRequest", mock.Anything, tt.prResponse.Number).
					Return(tt.mergeError)
			}

			if tt.expectCleanup {
				mockCleanup.On("CleanupIssueResources", mock.Anything, *tt.issue.Number).
					Return(tt.cleanupError)
			}

			// 実行
			err := executeAutoMergeIfLGTM(
				context.Background(),
				tt.issue,
				tt.config,
				mockGH,
				mockCleanup,
			)

			// 検証
			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
			}

			// モックの呼び出し回数を検証
			if tt.config != nil && tt.config.GitHub.AutoMergeLGTM && hasLGTMLabel(tt.issue) {
				mockGH.AssertCalled(t, "GetPullRequestForIssue", mock.Anything, *tt.issue.Number)
			} else {
				mockGH.AssertNotCalled(t, "GetPullRequestForIssue", mock.Anything, mock.Anything)
			}

			if tt.expectMerge && tt.prResponse != nil && tt.prError == nil {
				mockGH.AssertCalled(t, "MergePullRequest", mock.Anything, tt.prResponse.Number)
			} else {
				mockGH.AssertNotCalled(t, "MergePullRequest", mock.Anything, mock.Anything)
			}

			if tt.expectCleanup {
				mockCleanup.AssertCalled(t, "CleanupIssueResources", mock.Anything, *tt.issue.Number)
			} else {
				mockCleanup.AssertNotCalled(t, "CleanupIssueResources", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestHasLGTMLabel(t *testing.T) {
	tests := []struct {
		name     string
		issue    *github.Issue
		expected bool
	}{
		{
			name: "status:lgtmラベルが存在する",
			issue: &github.Issue{
				Labels: []*github.Label{
					{Name: github.String("status:ready")},
					{Name: github.String("status:lgtm")},
					{Name: github.String("priority:high")},
				},
			},
			expected: true,
		},
		{
			name: "status:lgtmラベルが存在しない",
			issue: &github.Issue{
				Labels: []*github.Label{
					{Name: github.String("status:ready")},
					{Name: github.String("priority:high")},
				},
			},
			expected: false,
		},
		{
			name: "ラベルが空の場合",
			issue: &github.Issue{
				Labels: []*github.Label{},
			},
			expected: false,
		},
		{
			name:     "ラベルがnilの場合",
			issue:    &github.Issue{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasLGTMLabel(tt.issue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsMergeable(t *testing.T) {
	tests := []struct {
		name     string
		pr       *github.PullRequest
		expected bool
	}{
		{
			name: "マージ可能: OPENかつMERGEABLE",
			pr: &github.PullRequest{
				State:     "OPEN",
				Mergeable: "MERGEABLE",
				IsDraft:   false,
			},
			expected: true,
		},
		{
			name: "マージ不可: CLOSEDステート",
			pr: &github.PullRequest{
				State:     "CLOSED",
				Mergeable: "MERGEABLE",
				IsDraft:   false,
			},
			expected: false,
		},
		{
			name: "マージ不可: MERGEDステート",
			pr: &github.PullRequest{
				State:     "MERGED",
				Mergeable: "MERGEABLE",
				IsDraft:   false,
			},
			expected: false,
		},
		{
			name: "マージ不可: CONFLICTING",
			pr: &github.PullRequest{
				State:     "OPEN",
				Mergeable: "CONFLICTING",
				IsDraft:   false,
			},
			expected: false,
		},
		{
			name: "マージ不可: ドラフトPR",
			pr: &github.PullRequest{
				State:     "OPEN",
				Mergeable: "MERGEABLE",
				IsDraft:   true,
			},
			expected: false,
		},
		{
			name: "マージ可能性不明: UNKNOWN",
			pr: &github.PullRequest{
				State:     "OPEN",
				Mergeable: "UNKNOWN",
				IsDraft:   false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isMergeable(tt.pr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

package mocks_test

import (
	"context"
	"errors"
	"testing"

	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMockGitHubClient_GetRepository(t *testing.T) {
	tests := []struct {
		name       string
		setupMock  func(*mocks.MockGitHubClient)
		owner      string
		repo       string
		wantRepo   *github.Repository
		wantErr    bool
		errMessage string
	}{
		{
			name: "success",
			setupMock: func(m *mocks.MockGitHubClient) {
				m.On("GetRepository", mock.Anything, "owner", "repo").
					Return(&github.Repository{Name: github.String("repo")}, nil)
			},
			owner:    "owner",
			repo:     "repo",
			wantRepo: &github.Repository{Name: github.String("repo")},
			wantErr:  false,
		},
		{
			name: "error",
			setupMock: func(m *mocks.MockGitHubClient) {
				m.On("GetRepository", mock.Anything, "owner", "repo").
					Return((*github.Repository)(nil), errors.New("api error"))
			},
			owner:      "owner",
			repo:       "repo",
			wantRepo:   nil,
			wantErr:    true,
			errMessage: "api error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGH := mocks.NewMockGitHubClient()
			tt.setupMock(mockGH)

			repo, err := mockGH.GetRepository(context.Background(), tt.owner, tt.repo)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantRepo, repo)
			}

			mockGH.AssertExpectations(t)
		})
	}
}

func TestMockGitHubClient_ListIssuesByLabels(t *testing.T) {
	mockGH := mocks.NewMockGitHubClient()

	expectedIssues := []*github.Issue{
		{Number: github.Int(1), Title: github.String("Issue 1")},
		{Number: github.Int(2), Title: github.String("Issue 2")},
	}

	mockGH.On("ListIssuesByLabels", mock.Anything, "owner", "repo", []string{"bug", "enhancement"}).
		Return(expectedIssues, nil)

	issues, err := mockGH.ListIssuesByLabels(context.Background(), "owner", "repo", []string{"bug", "enhancement"})

	assert.NoError(t, err)
	assert.Equal(t, expectedIssues, issues)
	mockGH.AssertExpectations(t)
}

func TestMockGitHubClient_WithDefaultBehavior(t *testing.T) {
	mockGH := mocks.NewMockGitHubClient().WithDefaultBehavior()

	// デフォルト動作のテスト
	rateLimit, err := mockGH.GetRateLimit(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, rateLimit)
	assert.Greater(t, rateLimit.Core.Remaining, 0)

	// CreateIssueCommentのデフォルト動作
	err = mockGH.CreateIssueComment(context.Background(), "owner", "repo", 1, "comment")
	assert.NoError(t, err)
}

func TestMockGitHubClient_TransitionIssueLabel(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mocks.MockGitHubClient)
		wantOk    bool
		wantErr   bool
	}{
		{
			name: "successful transition",
			setupMock: func(m *mocks.MockGitHubClient) {
				m.On("TransitionIssueLabel", mock.Anything, "owner", "repo", 123).
					Return(true, nil)
			},
			wantOk:  true,
			wantErr: false,
		},
		{
			name: "no transition needed",
			setupMock: func(m *mocks.MockGitHubClient) {
				m.On("TransitionIssueLabel", mock.Anything, "owner", "repo", 123).
					Return(false, nil)
			},
			wantOk:  false,
			wantErr: false,
		},
		{
			name: "error during transition",
			setupMock: func(m *mocks.MockGitHubClient) {
				m.On("TransitionIssueLabel", mock.Anything, "owner", "repo", 123).
					Return(false, errors.New("transition error"))
			},
			wantOk:  false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGH := mocks.NewMockGitHubClient()
			tt.setupMock(mockGH)

			ok, err := mockGH.TransitionIssueLabel(context.Background(), "owner", "repo", 123)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantOk, ok)
			mockGH.AssertExpectations(t)
		})
	}
}

func TestMockGitHubClient_LabelOperations(t *testing.T) {
	mockGH := mocks.NewMockGitHubClient()

	// AddLabel
	mockGH.On("AddLabel", mock.Anything, "owner", "repo", 1, "bug").Return(nil)
	err := mockGH.AddLabel(context.Background(), "owner", "repo", 1, "bug")
	assert.NoError(t, err)

	// RemoveLabel
	mockGH.On("RemoveLabel", mock.Anything, "owner", "repo", 1, "bug").Return(nil)
	err = mockGH.RemoveLabel(context.Background(), "owner", "repo", 1, "bug")
	assert.NoError(t, err)

	// EnsureLabelsExist
	mockGH.On("EnsureLabelsExist", mock.Anything, "owner", "repo").Return(nil)
	err = mockGH.EnsureLabelsExist(context.Background(), "owner", "repo")
	assert.NoError(t, err)

	mockGH.AssertExpectations(t)
}

func TestMockGitHubClient_ComplexArgumentMatching(t *testing.T) {
	mockGH := mocks.NewMockGitHubClient()

	// 複雑な引数マッチングの例
	mockGH.On("CreateIssueComment", mock.Anything, "owner", "repo", mock.MatchedBy(func(n int) bool {
		return n > 0 && n < 100
	}), mock.MatchedBy(func(s string) bool {
		return len(s) > 0
	})).Return(nil)

	// 正常なケース
	err := mockGH.CreateIssueComment(context.Background(), "owner", "repo", 50, "Valid comment")
	assert.NoError(t, err)

	// マッチしないケース（モックの期待値外）
	// これはエラーになることが期待される
	mockGH.AssertExpectations(t)
}

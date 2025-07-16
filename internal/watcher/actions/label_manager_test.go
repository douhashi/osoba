package actions

import (
	"context"
	"errors"
	"testing"

	"github.com/douhashi/osoba/internal/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockGitHubLabelClient はGitHubラベル操作のモック
type mockGitHubLabelClient struct {
	mock.Mock
}

func (m *mockGitHubLabelClient) AddLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

func (m *mockGitHubLabelClient) RemoveLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := m.Called(ctx, owner, repo, issueNumber, label)
	return args.Error(0)
}

// GitHub APIクライアントのその他のメソッド（テストでは使用しないが、インターフェースを満たすため）
func (m *mockGitHubLabelClient) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	return nil, nil
}

func (m *mockGitHubLabelClient) ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error) {
	return nil, nil
}

func (m *mockGitHubLabelClient) GetRateLimit(ctx context.Context) (*github.RateLimits, error) {
	return nil, nil
}

func (m *mockGitHubLabelClient) TransitionIssueLabel(ctx context.Context, owner, repo string, issueNumber int) (bool, error) {
	return false, nil
}

func (m *mockGitHubLabelClient) TransitionIssueLabelWithInfo(ctx context.Context, owner, repo string, issueNumber int) (bool, *github.TransitionInfo, error) {
	return false, nil, nil
}

func (m *mockGitHubLabelClient) EnsureLabelsExist(ctx context.Context, owner, repo string) error {
	return nil
}

func (m *mockGitHubLabelClient) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, comment string) error {
	return nil
}

func TestDefaultLabelManager_AddLabel(t *testing.T) {
	tests := []struct {
		name        string
		issueNumber int
		label       string
		setupMock   func(*mockGitHubLabelClient)
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "正常なラベル追加",
			issueNumber: 123,
			label:       "status:implementing",
			setupMock: func(client *mockGitHubLabelClient) {
				client.On("AddLabel", mock.Anything, "owner", "repo", 123, "status:implementing").Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "GitHub APIエラー",
			issueNumber: 456,
			label:       "status:planning",
			setupMock: func(client *mockGitHubLabelClient) {
				client.On("AddLabel", mock.Anything, "owner", "repo", 456, "status:planning").Return(errors.New("API error"))
			},
			wantErr: true,
			errMsg:  "API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックのセットアップ
			mockClient := new(mockGitHubLabelClient)
			tt.setupMock(mockClient)

			// DefaultLabelManagerの作成
			manager := &DefaultLabelManager{
				Owner:        "owner",
				Repo:         "repo",
				GitHubClient: mockClient,
			}

			// テスト実行
			err := manager.AddLabel(context.Background(), tt.issueNumber, tt.label)

			// アサーション
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			// モックの期待値を検証
			mockClient.AssertExpectations(t)
		})
	}
}

func TestDefaultLabelManager_RemoveLabel(t *testing.T) {
	tests := []struct {
		name        string
		issueNumber int
		label       string
		setupMock   func(*mockGitHubLabelClient)
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "正常なラベル削除",
			issueNumber: 123,
			label:       "status:ready",
			setupMock: func(client *mockGitHubLabelClient) {
				client.On("RemoveLabel", mock.Anything, "owner", "repo", 123, "status:ready").Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "存在しないラベル",
			issueNumber: 456,
			label:       "status:unknown",
			setupMock: func(client *mockGitHubLabelClient) {
				client.On("RemoveLabel", mock.Anything, "owner", "repo", 456, "status:unknown").Return(errors.New("label not found"))
			},
			wantErr: true,
			errMsg:  "label not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックのセットアップ
			mockClient := new(mockGitHubLabelClient)
			tt.setupMock(mockClient)

			// DefaultLabelManagerの作成
			manager := &DefaultLabelManager{
				Owner:        "owner",
				Repo:         "repo",
				GitHubClient: mockClient,
			}

			// テスト実行
			err := manager.RemoveLabel(context.Background(), tt.issueNumber, tt.label)

			// アサーション
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			// モックの期待値を検証
			mockClient.AssertExpectations(t)
		})
	}
}

func TestDefaultLabelManager_TransitionLabel(t *testing.T) {
	tests := []struct {
		name        string
		issueNumber int
		from        string
		to          string
		setupMock   func(*mockGitHubLabelClient)
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "正常なラベル遷移",
			issueNumber: 123,
			from:        "status:ready",
			to:          "status:implementing",
			setupMock: func(client *mockGitHubLabelClient) {
				client.On("RemoveLabel", mock.Anything, "owner", "repo", 123, "status:ready").Return(nil)
				client.On("AddLabel", mock.Anything, "owner", "repo", 123, "status:implementing").Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "削除失敗",
			issueNumber: 456,
			from:        "status:unknown",
			to:          "status:implementing",
			setupMock: func(client *mockGitHubLabelClient) {
				client.On("RemoveLabel", mock.Anything, "owner", "repo", 456, "status:unknown").Return(errors.New("label not found"))
			},
			wantErr: true,
			errMsg:  "failed to remove label",
		},
		{
			name:        "追加失敗",
			issueNumber: 789,
			from:        "status:ready",
			to:          "status:implementing",
			setupMock: func(client *mockGitHubLabelClient) {
				client.On("RemoveLabel", mock.Anything, "owner", "repo", 789, "status:ready").Return(nil)
				client.On("AddLabel", mock.Anything, "owner", "repo", 789, "status:implementing").Return(errors.New("permission denied"))
			},
			wantErr: true,
			errMsg:  "failed to add label",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックのセットアップ
			mockClient := new(mockGitHubLabelClient)
			tt.setupMock(mockClient)

			// DefaultLabelManagerの作成
			manager := &DefaultLabelManager{
				Owner:        "owner",
				Repo:         "repo",
				GitHubClient: mockClient,
			}

			// テスト実行
			err := manager.TransitionLabel(context.Background(), tt.issueNumber, tt.from, tt.to)

			// アサーション
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			// モックの期待値を検証
			mockClient.AssertExpectations(t)
		})
	}
}

func TestDefaultLabelManager_InitializationError(t *testing.T) {
	tests := []struct {
		name    string
		manager *DefaultLabelManager
		action  func(*DefaultLabelManager) error
		errMsg  string
	}{
		{
			name: "GitHubClient未初期化_AddLabel",
			manager: &DefaultLabelManager{
				Owner:        "owner",
				Repo:         "repo",
				GitHubClient: nil,
			},
			action: func(m *DefaultLabelManager) error {
				return m.AddLabel(context.Background(), 123, "status:implementing")
			},
			errMsg: "GitHub client is not initialized",
		},
		{
			name: "GitHubClient未初期化_RemoveLabel",
			manager: &DefaultLabelManager{
				Owner:        "owner",
				Repo:         "repo",
				GitHubClient: nil,
			},
			action: func(m *DefaultLabelManager) error {
				return m.RemoveLabel(context.Background(), 123, "status:ready")
			},
			errMsg: "GitHub client is not initialized",
		},
		{
			name: "GitHubClient未初期化_TransitionLabel",
			manager: &DefaultLabelManager{
				Owner:        "owner",
				Repo:         "repo",
				GitHubClient: nil,
			},
			action: func(m *DefaultLabelManager) error {
				return m.TransitionLabel(context.Background(), 123, "status:ready", "status:implementing")
			},
			errMsg: "GitHub client is not initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.action(tt.manager)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

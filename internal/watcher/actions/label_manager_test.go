package actions

import (
	"context"
	"errors"
	"testing"

	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDefaultLabelManager_AddLabel(t *testing.T) {
	tests := []struct {
		name        string
		issueNumber int
		label       string
		setupMock   func(*mocks.MockGitHubClient)
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "正常なラベル追加",
			issueNumber: 123,
			label:       "status:implementing",
			setupMock: func(client *mocks.MockGitHubClient) {
				client.On("AddLabel", mock.Anything, "owner", "repo", 123, "status:implementing").Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "GitHub APIエラー",
			issueNumber: 456,
			label:       "status:planning",
			setupMock: func(client *mocks.MockGitHubClient) {
				client.On("AddLabel", mock.Anything, "owner", "repo", 456, "status:planning").Return(errors.New("API error"))
			},
			wantErr: true,
			errMsg:  "API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックのセットアップ
			mockClient := new(mocks.MockGitHubClient)
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
		setupMock   func(*mocks.MockGitHubClient)
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "正常なラベル削除",
			issueNumber: 123,
			label:       "status:ready",
			setupMock: func(client *mocks.MockGitHubClient) {
				client.On("RemoveLabel", mock.Anything, "owner", "repo", 123, "status:ready").Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "存在しないラベル",
			issueNumber: 456,
			label:       "status:unknown",
			setupMock: func(client *mocks.MockGitHubClient) {
				client.On("RemoveLabel", mock.Anything, "owner", "repo", 456, "status:unknown").Return(errors.New("label not found"))
			},
			wantErr: true,
			errMsg:  "label not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックのセットアップ
			mockClient := new(mocks.MockGitHubClient)
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
		setupMock   func(*mocks.MockGitHubClient)
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "正常なラベル遷移",
			issueNumber: 123,
			from:        "status:ready",
			to:          "status:implementing",
			setupMock: func(client *mocks.MockGitHubClient) {
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
			setupMock: func(client *mocks.MockGitHubClient) {
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
			setupMock: func(client *mocks.MockGitHubClient) {
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
			mockClient := new(mocks.MockGitHubClient)
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

// TestDefaultLabelManager_EmptyOwnerRepo はowner/repoが空の場合のエラー処理を確認
func TestDefaultLabelManager_EmptyOwnerRepo(t *testing.T) {
	tests := []struct {
		name      string
		owner     string
		repo      string
		setupMock func(*mocks.MockGitHubClient)
		wantErr   bool
	}{
		{
			name:  "owner が空の場合",
			owner: "",
			repo:  "test-repo",
			setupMock: func(client *mocks.MockGitHubClient) {
				// GitHub APIクライアントがowner空文字列でエラーを返すことを期待
				client.On("RemoveLabel", mock.Anything, "", "test-repo", 123, "status:ready").
					Return(errors.New("owner is required"))
			},
			wantErr: true,
		},
		{
			name:  "repo が空の場合",
			owner: "test-owner",
			repo:  "",
			setupMock: func(client *mocks.MockGitHubClient) {
				// GitHub APIクライアントがrepo空文字列でエラーを返すことを期待
				client.On("RemoveLabel", mock.Anything, "test-owner", "", 123, "status:ready").
					Return(errors.New("repo is required"))
			},
			wantErr: true,
		},
		{
			name:  "owner と repo が両方空の場合",
			owner: "",
			repo:  "",
			setupMock: func(client *mocks.MockGitHubClient) {
				// GitHub APIクライアントがowner/repo空文字列でエラーを返すことを期待
				client.On("RemoveLabel", mock.Anything, "", "", 123, "status:ready").
					Return(errors.New("owner is required"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックのセットアップ
			mockClient := new(mocks.MockGitHubClient)
			tt.setupMock(mockClient)

			// DefaultLabelManagerの作成（owner/repoを意図的に空にする）
			manager := &DefaultLabelManager{
				Owner:        tt.owner,
				Repo:         tt.repo,
				GitHubClient: mockClient,
			}

			// テスト実行 - RemoveLabelを呼び出す
			err := manager.RemoveLabel(context.Background(), 123, "status:ready")

			// アサーション - エラーが発生することを確認
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// モックの期待値を検証
			mockClient.AssertExpectations(t)
		})
	}
}

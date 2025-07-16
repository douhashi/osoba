package actions

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockPhaseTransitioner はPhaseTransitionerのモック実装
type mockPhaseTransitioner struct {
	mock.Mock
}

func (m *mockPhaseTransitioner) TransitionPhase(ctx context.Context, issueNumber int, phase string, from, to string) error {
	args := m.Called(ctx, issueNumber, phase, from, to)
	return args.Error(0)
}

func TestPhaseTransitioner_TransitionPhase(t *testing.T) {
	tests := []struct {
		name        string
		issueNumber int
		phase       string
		from        string
		to          string
		setupMock   func(*mockGitHubClient, *mockConfigProvider)
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "計画フェーズの正常な遷移",
			issueNumber: 123,
			phase:       "plan",
			from:        "status:needs-plan",
			to:          "status:planning",
			setupMock: func(ghClient *mockGitHubClient, config *mockConfigProvider) {
				// コメント投稿
				config.On("GetPhaseMessage", "plan").Return("osoba: 計画を作成します", true)
				ghClient.On("CreateIssueComment", mock.Anything, "owner", "repo", 123, "osoba: 計画を作成します").Return(nil)

				// ラベル遷移
				ghClient.On("TransitionLabel", mock.Anything, 123, "status:needs-plan", "status:planning").Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "実装フェーズの正常な遷移",
			issueNumber: 456,
			phase:       "implement",
			from:        "status:ready",
			to:          "status:implementing",
			setupMock: func(ghClient *mockGitHubClient, config *mockConfigProvider) {
				// コメント投稿
				config.On("GetPhaseMessage", "implement").Return("osoba: 実装を開始します", true)
				ghClient.On("CreateIssueComment", mock.Anything, "owner", "repo", 456, "osoba: 実装を開始します").Return(nil)

				// ラベル遷移
				ghClient.On("TransitionLabel", mock.Anything, 456, "status:ready", "status:implementing").Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "レビューフェーズの正常な遷移",
			issueNumber: 789,
			phase:       "review",
			from:        "status:review-requested",
			to:          "status:reviewing",
			setupMock: func(ghClient *mockGitHubClient, config *mockConfigProvider) {
				// コメント投稿
				config.On("GetPhaseMessage", "review").Return("osoba: レビューを開始します", true)
				ghClient.On("CreateIssueComment", mock.Anything, "owner", "repo", 789, "osoba: レビューを開始します").Return(nil)

				// ラベル遷移
				ghClient.On("TransitionLabel", mock.Anything, 789, "status:review-requested", "status:reviewing").Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "コメント投稿失敗でもラベル遷移は実行される",
			issueNumber: 999,
			phase:       "plan",
			from:        "status:needs-plan",
			to:          "status:planning",
			setupMock: func(ghClient *mockGitHubClient, config *mockConfigProvider) {
				// コメント投稿（失敗）
				config.On("GetPhaseMessage", "plan").Return("osoba: 計画を作成します", true)
				ghClient.On("CreateIssueComment", mock.Anything, "owner", "repo", 999, "osoba: 計画を作成します").Return(errors.New("API error"))

				// ラベル遷移（成功）
				ghClient.On("TransitionLabel", mock.Anything, 999, "status:needs-plan", "status:planning").Return(nil)
			},
			wantErr: false, // コメント投稿が失敗してもエラーを返さない
		},
		{
			name:        "ラベル遷移失敗",
			issueNumber: 111,
			phase:       "plan",
			from:        "status:needs-plan",
			to:          "status:planning",
			setupMock: func(ghClient *mockGitHubClient, config *mockConfigProvider) {
				// コメント投稿（成功）
				config.On("GetPhaseMessage", "plan").Return("osoba: 計画を作成します", true)
				ghClient.On("CreateIssueComment", mock.Anything, "owner", "repo", 111, "osoba: 計画を作成します").Return(nil)

				// ラベル遷移（失敗）
				ghClient.On("TransitionLabel", mock.Anything, 111, "status:needs-plan", "status:planning").Return(errors.New("label not found"))
			},
			wantErr: true,
			errMsg:  "failed to transition label",
		},
		{
			name:        "存在しないフェーズ",
			issueNumber: 222,
			phase:       "unknown",
			from:        "status:unknown",
			to:          "status:unknown",
			setupMock: func(ghClient *mockGitHubClient, config *mockConfigProvider) {
				// メッセージが見つからない場合
				config.On("GetPhaseMessage", "unknown").Return("", false)

				// ラベル遷移は実行される
				ghClient.On("TransitionLabel", mock.Anything, 222, "status:unknown", "status:unknown").Return(nil)
			},
			wantErr: false, // メッセージが見つからなくてもエラーを返さない
		},
		{
			name:        "カスタムメッセージでの遷移",
			issueNumber: 333,
			phase:       "plan",
			from:        "status:needs-plan",
			to:          "status:planning",
			setupMock: func(ghClient *mockGitHubClient, config *mockConfigProvider) {
				// カスタムメッセージ
				config.On("GetPhaseMessage", "plan").Return("🤖 計画フェーズを開始します...", true)
				ghClient.On("CreateIssueComment", mock.Anything, "owner", "repo", 333, "🤖 計画フェーズを開始します...").Return(nil)

				// ラベル遷移
				ghClient.On("TransitionLabel", mock.Anything, 333, "status:needs-plan", "status:planning").Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックのセットアップ
			mockGH := new(mockGitHubClient)
			mockConfig := new(mockConfigProvider)
			tt.setupMock(mockGH, mockConfig)

			// PhaseTransitionerの作成
			transitioner := &DefaultPhaseTransitioner{
				owner:        "owner",
				repo:         "repo",
				githubClient: mockGH,
				config:       mockConfig,
			}

			// テスト実行
			err := transitioner.TransitionPhase(context.Background(), tt.issueNumber, tt.phase, tt.from, tt.to)

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
			mockGH.AssertExpectations(t)
			mockConfig.AssertExpectations(t)
		})
	}
}

// mockGitHubClient はGitHubクライアントのモック
type mockGitHubClient struct {
	mock.Mock
}

func (m *mockGitHubClient) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, comment string) error {
	args := m.Called(ctx, owner, repo, issueNumber, comment)
	return args.Error(0)
}

func (m *mockGitHubClient) TransitionLabel(ctx context.Context, issueNumber int, from, to string) error {
	args := m.Called(ctx, issueNumber, from, to)
	return args.Error(0)
}

func (m *mockGitHubClient) RemoveLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	return nil
}

func (m *mockGitHubClient) AddLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	return nil
}

// mockConfigProvider は設定プロバイダーのモック
type mockConfigProvider struct {
	mock.Mock
}

func (m *mockConfigProvider) GetPhaseMessage(phase string) (string, bool) {
	args := m.Called(phase)
	return args.String(0), args.Bool(1)
}

package github

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-github/v67/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestLabelManagerWithRetry_TransitionLabelWithRetry(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockLabelService)
		expectedCalls  int
		wantErr        bool
		wantTransition bool
	}{
		{
			name: "正常系: 1回目で成功",
			setupMocks: func(m *MockLabelService) {
				// 現在のラベル取得
				m.On("ListLabelsByIssue", mock.Anything, "owner", "repo", 1, (*github.ListOptions)(nil)).
					Return([]*github.Label{
						{Name: github.String("status:needs-plan")},
					}, &github.Response{}, nil).Once()

				// ラベル削除
				m.On("RemoveLabelForIssue", mock.Anything, "owner", "repo", 1, "status:needs-plan").
					Return(&github.Response{}, nil).Once()

				// ラベル追加
				m.On("AddLabelsToIssue", mock.Anything, "owner", "repo", 1, []string{"status:planning"}).
					Return([]*github.Label{
						{Name: github.String("status:planning")},
					}, &github.Response{}, nil).Once()
			},
			expectedCalls:  1,
			wantErr:        false,
			wantTransition: true,
		},
		{
			name: "リトライ成功: 2回目で成功",
			setupMocks: func(m *MockLabelService) {
				// 1回目: ラベル取得でエラー
				m.On("ListLabelsByIssue", mock.Anything, "owner", "repo", 1, (*github.ListOptions)(nil)).
					Return(nil, &github.Response{}, errors.New("temporary error")).Once()

				// 2回目: 成功
				m.On("ListLabelsByIssue", mock.Anything, "owner", "repo", 1, (*github.ListOptions)(nil)).
					Return([]*github.Label{
						{Name: github.String("status:needs-plan")},
					}, &github.Response{}, nil).Once()

				m.On("RemoveLabelForIssue", mock.Anything, "owner", "repo", 1, "status:needs-plan").
					Return(&github.Response{}, nil).Once()

				m.On("AddLabelsToIssue", mock.Anything, "owner", "repo", 1, []string{"status:planning"}).
					Return([]*github.Label{
						{Name: github.String("status:planning")},
					}, &github.Response{}, nil).Once()
			},
			expectedCalls:  2,
			wantErr:        false,
			wantTransition: true,
		},
		{
			name: "リトライ失敗: 3回全て失敗",
			setupMocks: func(m *MockLabelService) {
				// 3回とも失敗
				m.On("ListLabelsByIssue", mock.Anything, "owner", "repo", 1, (*github.ListOptions)(nil)).
					Return(nil, &github.Response{}, errors.New("persistent error")).Times(3)
			},
			expectedCalls:  3,
			wantErr:        true,
			wantTransition: false,
		},
		{
			name: "部分的失敗でリトライ: ラベル追加で失敗後成功",
			setupMocks: func(m *MockLabelService) {
				// ラベル取得は常に成功
				m.On("ListLabelsByIssue", mock.Anything, "owner", "repo", 1, (*github.ListOptions)(nil)).
					Return([]*github.Label{
						{Name: github.String("status:needs-plan")},
					}, &github.Response{}, nil).Times(2)

				// ラベル削除は常に成功
				m.On("RemoveLabelForIssue", mock.Anything, "owner", "repo", 1, "status:needs-plan").
					Return(&github.Response{}, nil).Times(2)

				// 1回目: ラベル追加で失敗
				m.On("AddLabelsToIssue", mock.Anything, "owner", "repo", 1, []string{"status:planning"}).
					Return(nil, &github.Response{}, errors.New("rate limit")).Once()

				// ロールバック
				m.On("AddLabelsToIssue", mock.Anything, "owner", "repo", 1, []string{"status:needs-plan"}).
					Return([]*github.Label{
						{Name: github.String("status:needs-plan")},
					}, &github.Response{}, nil).Once()

				// 2回目: ラベル追加で成功
				m.On("AddLabelsToIssue", mock.Anything, "owner", "repo", 1, []string{"status:planning"}).
					Return([]*github.Label{
						{Name: github.String("status:planning")},
					}, &github.Response{}, nil).Once()
			},
			expectedCalls:  2,
			wantErr:        false,
			wantTransition: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockLabelService{}
			tt.setupMocks(mockClient)

			manager := NewLabelManagerWithRetry(mockClient, 3, 10*time.Millisecond)
			ctx := context.Background()

			transitioned, err := manager.TransitionLabelWithRetry(ctx, "owner", "repo", 1)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantTransition, transitioned)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestLabelManagerWithRetry_EnsureLabelsExistWithRetry(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*MockLabelService)
		wantErr    bool
	}{
		{
			name: "正常系: 1回目で成功",
			setupMocks: func(m *MockLabelService) {
				// 既存ラベルの取得
				m.On("ListLabels", mock.Anything, "owner", "repo", (*github.ListOptions)(nil)).
					Return([]*github.Label{}, &github.Response{}, nil).Once()

				// 全てのラベルを作成
				labels := []string{
					"status:needs-plan",
					"status:planning",
					"status:ready",
					"status:implementing",
					"status:needs-review",
					"status:reviewing",
				}

				for _, labelName := range labels {
					m.On("CreateLabel", mock.Anything, "owner", "repo", mock.MatchedBy(func(label *github.Label) bool {
						return *label.Name == labelName
					})).Return(&github.Label{Name: github.String(labelName)}, &github.Response{}, nil).Once()
				}
			},
			wantErr: false,
		},
		{
			name: "リトライ成功: ListLabelsで2回目に成功",
			setupMocks: func(m *MockLabelService) {
				// 1回目: 失敗
				m.On("ListLabels", mock.Anything, "owner", "repo", (*github.ListOptions)(nil)).
					Return(nil, &github.Response{}, errors.New("temporary error")).Once()

				// 2回目: 成功
				m.On("ListLabels", mock.Anything, "owner", "repo", (*github.ListOptions)(nil)).
					Return([]*github.Label{
						{Name: github.String("status:needs-plan")},
						{Name: github.String("status:planning")},
						{Name: github.String("status:ready")},
						{Name: github.String("status:implementing")},
						{Name: github.String("status:needs-review")},
						{Name: github.String("status:reviewing")},
					}, &github.Response{}, nil).Once()
			},
			wantErr: false,
		},
		{
			name: "リトライ成功: CreateLabelで一時的に失敗",
			setupMocks: func(m *MockLabelService) {
				// 1回目の試行
				// ListLabels呼び出しは成功（既存のラベルはなし）
				m.On("ListLabels", mock.Anything, "owner", "repo", (*github.ListOptions)(nil)).
					Return([]*github.Label{}, &github.Response{}, nil).Once()

				// 最初のラベル作成（status:needs-plan）は失敗
				m.On("CreateLabel", mock.Anything, "owner", "repo", mock.MatchedBy(func(label *github.Label) bool {
					return *label.Name == "status:needs-plan"
				})).Return(nil, &github.Response{}, errors.New("rate limit")).Once()

				// 2回目の試行（リトライ）
				// ListLabels呼び出し（リトライ時）
				m.On("ListLabels", mock.Anything, "owner", "repo", (*github.ListOptions)(nil)).
					Return([]*github.Label{}, &github.Response{}, nil).Once()

				// 全てのラベルの作成が成功（リトライ時は最初から全て作成し直す）
				// mapの反復順序は不定なので、任意のラベルの作成を受け入れる
				m.On("CreateLabel", mock.Anything, "owner", "repo", mock.MatchedBy(func(label *github.Label) bool {
					validLabels := map[string]bool{
						"status:needs-plan":   true,
						"status:planning":     true,
						"status:ready":        true,
						"status:implementing": true,
						"status:needs-review": true,
						"status:reviewing":    true,
					}
					return validLabels[*label.Name]
				})).Return(&github.Label{}, &github.Response{}, nil).Times(6)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockLabelService{}
			tt.setupMocks(mockClient)

			manager := NewLabelManagerWithRetry(mockClient, 3, 10*time.Millisecond)
			ctx := context.Background()

			err := manager.EnsureLabelsExistWithRetry(ctx, "owner", "repo")

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

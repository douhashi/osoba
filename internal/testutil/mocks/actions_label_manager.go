package mocks

import (
	"context"

	"github.com/douhashi/osoba/internal/github"
	"github.com/stretchr/testify/mock"
)

// MockLabelManager は actions.LabelManager インターフェースのモック実装
type MockLabelManager struct {
	mock.Mock
}

// NewMockLabelManager creates a new instance of MockLabelManager
func NewMockLabelManager() *MockLabelManager {
	return &MockLabelManager{}
}

// TransitionLabel はラベルを遷移させる
func (m *MockLabelManager) TransitionLabel(ctx context.Context, issueNumber int, from, to string) error {
	args := m.Called(ctx, issueNumber, from, to)
	return args.Error(0)
}

// AddLabel はラベルを追加する
func (m *MockLabelManager) AddLabel(ctx context.Context, issueNumber int, label string) error {
	args := m.Called(ctx, issueNumber, label)
	return args.Error(0)
}

// RemoveLabel はラベルを削除する
func (m *MockLabelManager) RemoveLabel(ctx context.Context, issueNumber int, label string) error {
	args := m.Called(ctx, issueNumber, label)
	return args.Error(0)
}

// GetPullRequestForIssue はIssueに関連するPRを取得する
func (m *MockLabelManager) GetPullRequestForIssue(ctx context.Context, issueNumber int) (*github.PullRequest, error) {
	args := m.Called(ctx, issueNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.PullRequest), args.Error(1)
}

// WithSuccessfulTransition は成功するTransitionLabelの期待値を設定
func (m *MockLabelManager) WithSuccessfulTransition(ctx context.Context, issueNumber int, from, to string) *MockLabelManager {
	m.On("TransitionLabel", ctx, issueNumber, from, to).Return(nil)
	return m
}

// WithTransitionError はエラーを返すTransitionLabelの期待値を設定
func (m *MockLabelManager) WithTransitionError(ctx context.Context, issueNumber int, from, to string, err error) *MockLabelManager {
	m.On("TransitionLabel", ctx, issueNumber, from, to).Return(err)
	return m
}

// WithAddLabelSuccess は成功するAddLabelの期待値を設定
func (m *MockLabelManager) WithAddLabelSuccess(ctx context.Context, issueNumber int, label string) *MockLabelManager {
	m.On("AddLabel", ctx, issueNumber, label).Return(nil)
	return m
}

// WithAddLabelError はエラーを返すAddLabelの期待値を設定
func (m *MockLabelManager) WithAddLabelError(ctx context.Context, issueNumber int, label string, err error) *MockLabelManager {
	m.On("AddLabel", ctx, issueNumber, label).Return(err)
	return m
}

// WithRemoveLabelSuccess は成功するRemoveLabelの期待値を設定
func (m *MockLabelManager) WithRemoveLabelSuccess(ctx context.Context, issueNumber int, label string) *MockLabelManager {
	m.On("RemoveLabel", ctx, issueNumber, label).Return(nil)
	return m
}

// WithRemoveLabelError はエラーを返すRemoveLabelの期待値を設定
func (m *MockLabelManager) WithRemoveLabelError(ctx context.Context, issueNumber int, label string, err error) *MockLabelManager {
	m.On("RemoveLabel", ctx, issueNumber, label).Return(err)
	return m
}

// WithDefaultBehavior はデフォルトの動作を設定（すべてのメソッドが成功）
func (m *MockLabelManager) WithDefaultBehavior() *MockLabelManager {
	m.On("TransitionLabel", mock.Anything, mock.AnythingOfType("int"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Maybe().Return(nil)
	m.On("AddLabel", mock.Anything, mock.AnythingOfType("int"), mock.AnythingOfType("string")).Maybe().Return(nil)
	m.On("RemoveLabel", mock.Anything, mock.AnythingOfType("int"), mock.AnythingOfType("string")).Maybe().Return(nil)
	return m
}

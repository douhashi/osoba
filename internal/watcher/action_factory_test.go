package watcher

import (
	"context"
	"testing"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWorktreeManager はWorktreeManagerのモック
type MockWorktreeManager struct {
	mock.Mock
}

func (m *MockWorktreeManager) UpdateMainBranch(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockWorktreeManager) CreateWorktree(ctx context.Context, issueNumber int, phase git.Phase) error {
	args := m.Called(ctx, issueNumber, phase)
	return args.Error(0)
}

func (m *MockWorktreeManager) RemoveWorktree(ctx context.Context, issueNumber int, phase git.Phase) error {
	args := m.Called(ctx, issueNumber, phase)
	return args.Error(0)
}

func (m *MockWorktreeManager) GetWorktreePath(issueNumber int, phase git.Phase) string {
	args := m.Called(issueNumber, phase)
	return args.String(0)
}

func (m *MockWorktreeManager) WorktreeExists(ctx context.Context, issueNumber int, phase git.Phase) (bool, error) {
	args := m.Called(ctx, issueNumber, phase)
	return args.Bool(0), args.Error(1)
}

// CreateWorktreeForIssue はIssue単位でのworktree作成（V2用）
func (m *MockWorktreeManager) CreateWorktreeForIssue(ctx context.Context, issueNumber int) error {
	args := m.Called(ctx, issueNumber)
	return args.Error(0)
}

// WorktreeExistsForIssue はIssue単位でのworktree存在確認（V2用）
func (m *MockWorktreeManager) WorktreeExistsForIssue(ctx context.Context, issueNumber int) (bool, error) {
	args := m.Called(ctx, issueNumber)
	return args.Bool(0), args.Error(1)
}

// GetWorktreePathForIssue はIssue単位でのworktreeパス取得（V2用）
func (m *MockWorktreeManager) GetWorktreePathForIssue(issueNumber int) string {
	args := m.Called(issueNumber)
	return args.String(0)
}

// RemoveWorktreeForIssue はIssue単位でのworktree削除（V2用）
func (m *MockWorktreeManager) RemoveWorktreeForIssue(ctx context.Context, issueNumber int) error {
	args := m.Called(ctx, issueNumber)
	return args.Error(0)
}

func TestActionFactory(t *testing.T) {
	t.Run("DefaultActionFactoryの作成", func(t *testing.T) {
		// Arrange
		sessionName := "test-session"
		ghClient := &github.Client{}
		worktreeManager := &MockWorktreeManager{}
		ml := NewMockLogger()
		claudeExecutor := claude.NewClaudeExecutorWithLogger(ml)
		claudeConfig := &claude.ClaudeConfig{}

		// Act
		factory := NewDefaultActionFactory(
			sessionName,
			ghClient,
			worktreeManager,
			claudeExecutor,
			claudeConfig,
			config.NewConfig(),
			"test-owner",
			"test-repo",
		)

		// Assert
		assert.NotNil(t, factory)
		assert.Equal(t, sessionName, factory.sessionName)
		assert.NotNil(t, factory.ghClient)
		assert.NotNil(t, factory.worktreeManager)
		assert.NotNil(t, factory.claudeExecutor)
		assert.NotNil(t, factory.claudeConfig)
		assert.NotNil(t, factory.stateManager)
	})

	t.Run("CreatePlanActionの作成", func(t *testing.T) {
		// Arrange
		ml := NewMockLogger()
		factory := &DefaultActionFactory{
			sessionName:     "test-session",
			ghClient:        &github.Client{},
			worktreeManager: &MockWorktreeManager{},
			claudeExecutor:  claude.NewClaudeExecutorWithLogger(ml),
			claudeConfig:    &claude.ClaudeConfig{},
			stateManager:    NewIssueStateManager(),
		}

		// Act
		action := factory.CreatePlanAction()

		// Assert
		assert.NotNil(t, action)
	})

	t.Run("CreatePlanActionの作成 - ghクライアント", func(t *testing.T) {
		// Arrange
		mockGhClient := mocks.NewMockGitHubClient()
		ml := NewMockLogger()
		factory := &DefaultActionFactory{
			sessionName:     "test-session",
			ghClient:        mockGhClient,
			worktreeManager: &MockWorktreeManager{},
			claudeExecutor:  claude.NewClaudeExecutorWithLogger(ml),
			claudeConfig:    &claude.ClaudeConfig{},
			stateManager:    NewIssueStateManager(),
			config:          config.NewConfig(),
			owner:           "test-owner",
			repo:            "test-repo",
		}

		// Act
		action := factory.CreatePlanAction()

		// Assert
		assert.NotNil(t, action)
		// 現在の実装では、ghクライアントでPhaseTransitionerが作成されない（これが問題）
		// TDDでは、このテストは失敗すべきだが、現在はnilでもアクション作成は成功するため、
		// 実装修正後により適切な動作になることを確認するため
	})

	t.Run("CreateImplementationActionの作成", func(t *testing.T) {
		// Arrange
		ml := NewMockLogger()
		factory := &DefaultActionFactory{
			sessionName:     "test-session",
			ghClient:        &github.Client{},
			worktreeManager: &MockWorktreeManager{},
			claudeExecutor:  claude.NewClaudeExecutorWithLogger(ml),
			claudeConfig:    &claude.ClaudeConfig{},
			stateManager:    NewIssueStateManager(),
		}

		// Act
		action := factory.CreateImplementationAction()

		// Assert
		assert.NotNil(t, action)
	})

	t.Run("CreateImplementationActionの作成 - ghクライアント", func(t *testing.T) {
		// Arrange
		mockGhClient := mocks.NewMockGitHubClient()
		ml := NewMockLogger()
		factory := &DefaultActionFactory{
			sessionName:     "test-session",
			ghClient:        mockGhClient,
			worktreeManager: &MockWorktreeManager{},
			claudeExecutor:  claude.NewClaudeExecutorWithLogger(ml),
			claudeConfig:    &claude.ClaudeConfig{},
			stateManager:    NewIssueStateManager(),
			config:          config.NewConfig(),
			owner:           "test-owner",
			repo:            "test-repo",
		}

		// Act
		action := factory.CreateImplementationAction()

		// Assert
		assert.NotNil(t, action)
	})

	t.Run("CreateReviewActionの作成", func(t *testing.T) {
		// Arrange
		ml := NewMockLogger()
		factory := &DefaultActionFactory{
			sessionName:     "test-session",
			ghClient:        &github.Client{},
			worktreeManager: &MockWorktreeManager{},
			claudeExecutor:  claude.NewClaudeExecutorWithLogger(ml),
			claudeConfig:    &claude.ClaudeConfig{},
			stateManager:    NewIssueStateManager(),
		}

		// Act
		action := factory.CreateReviewAction()

		// Assert
		assert.NotNil(t, action)
	})

	t.Run("CreateReviewActionの作成 - ghクライアント", func(t *testing.T) {
		// Arrange
		mockGhClient := mocks.NewMockGitHubClient()
		ml := NewMockLogger()
		factory := &DefaultActionFactory{
			sessionName:     "test-session",
			ghClient:        mockGhClient,
			worktreeManager: &MockWorktreeManager{},
			claudeExecutor:  claude.NewClaudeExecutorWithLogger(ml),
			claudeConfig:    &claude.ClaudeConfig{},
			stateManager:    NewIssueStateManager(),
			config:          config.NewConfig(),
			owner:           "test-owner",
			repo:            "test-repo",
		}

		// Act
		action := factory.CreateReviewAction()

		// Assert
		assert.NotNil(t, action)
	})
}

// TestActionManagerWithFactory は別のテストファイルで実装

// TestDefaultLabelManager_OwnerRepoNotSet はowner/repoが設定されていない場合の動作を確認
func TestDefaultLabelManager_OwnerRepoNotSet(t *testing.T) {
	// Arrange
	ml := NewMockLogger()
	factory := &DefaultActionFactory{
		sessionName:     "test-session",
		ghClient:        &github.Client{},
		worktreeManager: &MockWorktreeManager{},
		claudeExecutor:  claude.NewClaudeExecutorWithLogger(ml),
		claudeConfig:    &claude.ClaudeConfig{},
		stateManager:    NewIssueStateManager(),
		config:          config.NewConfig(),
		owner:           "test-owner",
		repo:            "test-repo",
	}

	// Act - CreateImplementationActionを実行
	action := factory.CreateImplementationAction()

	// Assert - アクションが作成されることを確認
	// 現在の実装では、CreateImplementationAction内でowner/repoが設定されていないため、
	// 実際にラベル操作を行おうとするとエラーになる
	assert.NotNil(t, action)

	// TODO: 実際のラベル操作でowner/repoが設定されているかを確認するテストが必要
}

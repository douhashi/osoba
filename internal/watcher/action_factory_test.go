package watcher

import (
	"context"
	"testing"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/github"
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

func TestActionFactory(t *testing.T) {
	t.Run("DefaultActionFactoryの作成", func(t *testing.T) {
		// Arrange
		sessionName := "test-session"
		ghClient := &github.Client{}
		worktreeManager := &MockWorktreeManager{}
		claudeExecutor := claude.NewClaudeExecutor()
		claudeConfig := &claude.ClaudeConfig{}

		// Act
		factory := NewDefaultActionFactory(
			sessionName,
			ghClient,
			worktreeManager,
			claudeExecutor,
			claudeConfig,
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
		factory := &DefaultActionFactory{
			sessionName:     "test-session",
			ghClient:        &github.Client{},
			worktreeManager: &MockWorktreeManager{},
			claudeExecutor:  claude.NewClaudeExecutor(),
			claudeConfig:    &claude.ClaudeConfig{},
			stateManager:    NewIssueStateManager(),
		}

		// Act
		action := factory.CreatePlanAction()

		// Assert
		assert.NotNil(t, action)
	})

	t.Run("CreateImplementationActionの作成", func(t *testing.T) {
		// Arrange
		factory := &DefaultActionFactory{
			sessionName:     "test-session",
			ghClient:        &github.Client{},
			worktreeManager: &MockWorktreeManager{},
			claudeExecutor:  claude.NewClaudeExecutor(),
			claudeConfig:    &claude.ClaudeConfig{},
			stateManager:    NewIssueStateManager(),
		}

		// Act
		action := factory.CreateImplementationAction()

		// Assert
		assert.NotNil(t, action)
	})

	t.Run("CreateReviewActionの作成", func(t *testing.T) {
		// Arrange
		factory := &DefaultActionFactory{
			sessionName:     "test-session",
			ghClient:        &github.Client{},
			worktreeManager: &MockWorktreeManager{},
			claudeExecutor:  claude.NewClaudeExecutor(),
			claudeConfig:    &claude.ClaudeConfig{},
			stateManager:    NewIssueStateManager(),
		}

		// Act
		action := factory.CreateReviewAction()

		// Assert
		assert.NotNil(t, action)
	})
}

// TestActionManagerWithFactory は別のテストファイルで実装

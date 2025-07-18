package mocks

import (
	"context"

	"github.com/douhashi/osoba/internal/git"
	"github.com/stretchr/testify/mock"
)

// MockGitWorktreeManager is a mock implementation of git.WorktreeManager interface
type MockGitWorktreeManager struct {
	mock.Mock
}

// NewMockGitWorktreeManager creates a new instance of MockGitWorktreeManager
func NewMockGitWorktreeManager() *MockGitWorktreeManager {
	return &MockGitWorktreeManager{}
}

// UpdateMainBranch mocks the UpdateMainBranch method
func (m *MockGitWorktreeManager) UpdateMainBranch(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// CreateWorktree mocks the CreateWorktree method
func (m *MockGitWorktreeManager) CreateWorktree(ctx context.Context, issueNumber int, phase git.Phase) error {
	args := m.Called(ctx, issueNumber, phase)
	return args.Error(0)
}

// RemoveWorktree mocks the RemoveWorktree method
func (m *MockGitWorktreeManager) RemoveWorktree(ctx context.Context, issueNumber int, phase git.Phase) error {
	args := m.Called(ctx, issueNumber, phase)
	return args.Error(0)
}

// GetWorktreePath mocks the GetWorktreePath method
func (m *MockGitWorktreeManager) GetWorktreePath(issueNumber int, phase git.Phase) string {
	args := m.Called(issueNumber, phase)
	return args.String(0)
}

// WorktreeExists mocks the WorktreeExists method
func (m *MockGitWorktreeManager) WorktreeExists(ctx context.Context, issueNumber int, phase git.Phase) (bool, error) {
	args := m.Called(ctx, issueNumber, phase)
	return args.Bool(0), args.Error(1)
}

// GetWorktreePathForIssue mocks the GetWorktreePathForIssue method
func (m *MockGitWorktreeManager) GetWorktreePathForIssue(issueNumber int) string {
	args := m.Called(issueNumber)
	return args.String(0)
}

// WorktreeExistsForIssue mocks the WorktreeExistsForIssue method
func (m *MockGitWorktreeManager) WorktreeExistsForIssue(ctx context.Context, issueNumber int) (bool, error) {
	args := m.Called(ctx, issueNumber)
	return args.Bool(0), args.Error(1)
}

// CreateWorktreeForIssue mocks the CreateWorktreeForIssue method
func (m *MockGitWorktreeManager) CreateWorktreeForIssue(ctx context.Context, issueNumber int) error {
	args := m.Called(ctx, issueNumber)
	return args.Error(0)
}

// RemoveWorktreeForIssue mocks the RemoveWorktreeForIssue method
func (m *MockGitWorktreeManager) RemoveWorktreeForIssue(ctx context.Context, issueNumber int) error {
	args := m.Called(ctx, issueNumber)
	return args.Error(0)
}

// ListWorktreesForIssue mocks the ListWorktreesForIssue method
func (m *MockGitWorktreeManager) ListWorktreesForIssue(ctx context.Context, issueNumber int) ([]git.WorktreeInfo, error) {
	args := m.Called(ctx, issueNumber)
	return args.Get(0).([]git.WorktreeInfo), args.Error(1)
}

// ListAllWorktrees mocks the ListAllWorktrees method
func (m *MockGitWorktreeManager) ListAllWorktrees(ctx context.Context) ([]git.WorktreeInfo, error) {
	args := m.Called(ctx)
	return args.Get(0).([]git.WorktreeInfo), args.Error(1)
}

// HasUncommittedChanges mocks the HasUncommittedChanges method
func (m *MockGitWorktreeManager) HasUncommittedChanges(ctx context.Context, worktreePath string) (bool, error) {
	args := m.Called(ctx, worktreePath)
	return args.Bool(0), args.Error(1)
}

// Ensure MockGitWorktreeManager implements git.WorktreeManager interface
var _ git.WorktreeManager = (*MockGitWorktreeManager)(nil)

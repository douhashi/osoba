package actions

import (
	"context"
	"os/exec"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/git"
	"github.com/stretchr/testify/mock"
)

// MockTmuxClient はTmuxClientのモック
type MockTmuxClient struct {
	mock.Mock
}

func (m *MockTmuxClient) CreateWindowForIssue(sessionName string, issueNumber int) error {
	args := m.Called(sessionName, issueNumber)
	return args.Error(0)
}

func (m *MockTmuxClient) SwitchToIssueWindow(sessionName string, issueNumber int) error {
	args := m.Called(sessionName, issueNumber)
	return args.Error(0)
}

func (m *MockTmuxClient) WindowExists(sessionName, windowName string) (bool, error) {
	args := m.Called(sessionName, windowName)
	return args.Bool(0), args.Error(1)
}

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

// MockClaudeExecutor はClaudeExecutorのモック
type MockClaudeExecutor struct {
	mock.Mock
}

func (m *MockClaudeExecutor) ExecuteInTmux(ctx context.Context, phaseConfig *claude.PhaseConfig, templateVars *claude.TemplateVariables, sessionName, windowName, workingDir string) error {
	args := m.Called(ctx, phaseConfig, templateVars, sessionName, windowName, workingDir)
	return args.Error(0)
}

func (m *MockClaudeExecutor) CheckClaudeExists() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockClaudeExecutor) BuildCommand(ctx context.Context, args []string, prompt string, workdir string) *exec.Cmd {
	argList := m.Called(ctx, args, prompt, workdir)
	if cmd := argList.Get(0); cmd != nil {
		return cmd.(*exec.Cmd)
	}
	return nil
}

func (m *MockClaudeExecutor) ExecuteInWorktree(ctx context.Context, config *claude.PhaseConfig, vars *claude.TemplateVariables, workdir string) error {
	args := m.Called(ctx, config, vars, workdir)
	return args.Error(0)
}

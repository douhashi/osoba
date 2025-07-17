package mocks_test

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMockClaudeExecutor_CheckClaudeExists(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mocks.MockClaudeExecutor)
		wantErr   bool
	}{
		{
			name: "claude exists",
			setupMock: func(m *mocks.MockClaudeExecutor) {
				m.On("CheckClaudeExists").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "claude not found",
			setupMock: func(m *mocks.MockClaudeExecutor) {
				m.On("CheckClaudeExists").Return(errors.New("claude not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClaude := mocks.NewMockClaudeExecutor()
			tt.setupMock(mockClaude)

			err := mockClaude.CheckClaudeExists()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockClaude.AssertExpectations(t)
		})
	}
}

func TestMockClaudeExecutor_BuildCommand(t *testing.T) {
	mockClaude := mocks.NewMockClaudeExecutor()

	args := []string{"arg1", "arg2"}
	prompt := "test prompt"
	workdir := "/work/dir"

	expectedCmd := &exec.Cmd{
		Path: "/usr/bin/claude",
		Args: []string{"claude", "arg1", "arg2"},
		Dir:  workdir,
	}

	mockClaude.On("BuildCommand", mock.Anything, args, prompt, workdir).Return(expectedCmd)

	cmd := mockClaude.BuildCommand(context.Background(), args, prompt, workdir)

	assert.NotNil(t, cmd)
	assert.Equal(t, expectedCmd.Dir, cmd.Dir)
	mockClaude.AssertExpectations(t)
}

func TestMockClaudeExecutor_ExecuteInWorktree(t *testing.T) {
	mockClaude := mocks.NewMockClaudeExecutor()

	config := &claude.PhaseConfig{
		Args:   []string{"--arg1", "value1"},
		Prompt: "Implementation prompt for issue {{issue-number}}: {{issue-title}}",
	}

	vars := &claude.TemplateVariables{
		IssueNumber: 123,
		IssueTitle:  "Test Issue",
		RepoName:    "test-repo",
	}

	workdir := "/workspace/test"

	mockClaude.On("ExecuteInWorktree", mock.Anything, config, vars, workdir).Return(nil)

	err := mockClaude.ExecuteInWorktree(context.Background(), config, vars, workdir)

	assert.NoError(t, err)
	mockClaude.AssertExpectations(t)
}

func TestMockClaudeExecutor_ExecuteInTmux(t *testing.T) {
	mockClaude := mocks.NewMockClaudeExecutor()

	config := &claude.PhaseConfig{
		Args:   []string{},
		Prompt: "Plan prompt for issue {{issue-number}}: {{issue-title}}",
	}

	vars := &claude.TemplateVariables{
		IssueNumber: 456,
		IssueTitle:  "Plan Issue",
		RepoName:    "test-repo",
	}

	sessionName := "osoba-test"
	windowName := "issue-456"
	workdir := "/workspace/test"

	mockClaude.On("ExecuteInTmux", mock.Anything, config, vars, sessionName, windowName, workdir).Return(nil)

	err := mockClaude.ExecuteInTmux(context.Background(), config, vars, sessionName, windowName, workdir)

	assert.NoError(t, err)
	mockClaude.AssertExpectations(t)
}

func TestMockClaudeExecutor_WithDefaultBehavior(t *testing.T) {
	mockClaude := mocks.NewMockClaudeExecutor().WithDefaultBehavior()

	// CheckClaudeExists returns no error by default
	err := mockClaude.CheckClaudeExists()
	assert.NoError(t, err)

	// BuildCommand returns a non-nil command
	cmd := mockClaude.BuildCommand(context.Background(), []string{}, "", "")
	assert.NotNil(t, cmd)

	// ExecuteInWorktree succeeds by default
	err = mockClaude.ExecuteInWorktree(context.Background(), &claude.PhaseConfig{}, &claude.TemplateVariables{}, "")
	assert.NoError(t, err)

	// ExecuteInTmux succeeds by default
	err = mockClaude.ExecuteInTmux(context.Background(), &claude.PhaseConfig{}, &claude.TemplateVariables{}, "", "", "")
	assert.NoError(t, err)
}

func TestMockClaudeExecutor_ComplexScenario(t *testing.T) {
	mockClaude := mocks.NewMockClaudeExecutor()

	// 複雑なシナリオのセットアップ
	mockClaude.On("CheckClaudeExists").Return(nil).Once()

	mockClaude.On("BuildCommand", mock.Anything, mock.MatchedBy(func(args []string) bool {
		return len(args) > 0
	}), mock.Anything, mock.Anything).Return(&exec.Cmd{Path: "claude"}).Once()

	mockClaude.On("ExecuteInTmux", mock.Anything, mock.MatchedBy(func(config *claude.PhaseConfig) bool {
		return strings.Contains(config.Prompt, "plan")
	}), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	mockClaude.On("ExecuteInWorktree", mock.Anything, mock.MatchedBy(func(config *claude.PhaseConfig) bool {
		return strings.Contains(config.Prompt, "implementation")
	}), mock.Anything, mock.Anything).Return(nil).Once()

	// 実行
	err := mockClaude.CheckClaudeExists()
	assert.NoError(t, err)

	cmd := mockClaude.BuildCommand(context.Background(), []string{"--help"}, "help", "/tmp")
	assert.NotNil(t, cmd)

	planConfig := &claude.PhaseConfig{Prompt: "plan for issue"}
	err = mockClaude.ExecuteInTmux(context.Background(), planConfig, &claude.TemplateVariables{}, "session", "window", "/work")
	assert.NoError(t, err)

	implConfig := &claude.PhaseConfig{Prompt: "implementation for issue"}
	err = mockClaude.ExecuteInWorktree(context.Background(), implConfig, &claude.TemplateVariables{}, "/work")
	assert.NoError(t, err)

	mockClaude.AssertExpectations(t)
}

package actions

import (
	"context"
	"fmt"

	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
	tmuxpkg "github.com/douhashi/osoba/internal/tmux"
)

// WorkspaceInfo はワークスペース情報を表す構造体
type WorkspaceInfo struct {
	WindowName   string
	WorktreePath string
	PaneIndex    int
	PaneTitle    string
}

// ClaudeCommandBuilder はClaudeコマンドを構築するインターフェース
type ClaudeCommandBuilder interface {
	BuildCommand(promptPath string, outputPath string, workdir string, vars interface{}) string
}

// BaseExecutor は各ActionExecutorの共通機能を提供する構造体
type BaseExecutor struct {
	sessionName     string
	tmuxManager     tmuxpkg.Manager
	worktreeManager git.WorktreeManager
	claudeExecutor  ClaudeCommandBuilder
	logger          logger.Logger
}

// NewBaseExecutor は新しいBaseExecutorを作成する
func NewBaseExecutor(
	sessionName string,
	tmuxManager tmuxpkg.Manager,
	worktreeManager git.WorktreeManager,
	claudeExecutor ClaudeCommandBuilder,
	logger logger.Logger,
) *BaseExecutor {
	return &BaseExecutor{
		sessionName:     sessionName,
		tmuxManager:     tmuxManager,
		worktreeManager: worktreeManager,
		claudeExecutor:  claudeExecutor,
		logger:          logger,
	}
}

// PrepareWorkspace はIssueに対するワークスペースを準備する
func (e *BaseExecutor) PrepareWorkspace(ctx context.Context, issue *github.Issue, phase string) (*WorkspaceInfo, error) {
	if issue == nil || issue.Number == nil {
		return nil, fmt.Errorf("invalid issue: issue or issue number is nil")
	}

	issueNumber := *issue.Number
	windowName := tmuxpkg.GetWindowNameForIssue(int(issueNumber))

	e.logger.Info("Preparing workspace",
		"issue_number", issueNumber,
		"phase", phase,
		"window_name", windowName,
	)

	// 1. Windowの存在確認と作成（新規判定付き）
	isNewWindow := false
	windowExists, err := e.tmuxManager.WindowExists(e.sessionName, windowName)
	if err != nil {
		return nil, fmt.Errorf("failed to check window existence: %w", err)
	}

	if !windowExists {
		e.logger.Info("Creating new window with detection", "window_name", windowName)
		_, isNewWindow, err = e.tmuxManager.CreateWindowForIssueWithNewWindowDetection(e.sessionName, int(issueNumber))
		if err != nil {
			return nil, fmt.Errorf("failed to create window: %w", err)
		}
		e.logger.Info("Window creation result", "is_new_window", isNewWindow)
	}

	// 2. Worktreeの存在確認（なければ作成）
	worktreeExists, err := e.worktreeManager.WorktreeExistsForIssue(ctx, int(issueNumber))
	if err != nil {
		return nil, fmt.Errorf("failed to check worktree existence: %w", err)
	}

	if !worktreeExists {
		e.logger.Info("Creating new worktree", "issue_number", issueNumber)
		if err := e.worktreeManager.CreateWorktreeForIssue(ctx, int(issueNumber)); err != nil {
			return nil, fmt.Errorf("failed to create worktree: %w", err)
		}
	}

	// 3. 適切なpaneの選択または作成
	paneInfo, err := e.ensurePane(windowName, phase, isNewWindow)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure pane: %w", err)
	}

	// 4. WorkspaceInfoの返却
	worktreePath := e.worktreeManager.GetWorktreePathForIssue(int(issueNumber))

	return &WorkspaceInfo{
		WindowName:   windowName,
		WorktreePath: worktreePath,
		PaneIndex:    paneInfo.Index,
		PaneTitle:    paneInfo.Title,
	}, nil
}

// ensurePane は指定されたフェーズ用のpaneを確保する
func (e *BaseExecutor) ensurePane(windowName string, phase string, isNewWindow bool) (*tmuxpkg.PaneInfo, error) {
	// まず既存のpaneを検索
	existingPane, err := e.tmuxManager.GetPaneByTitle(e.sessionName, windowName, phase)
	if err == nil && existingPane != nil {
		e.logger.Info("Using existing pane", "phase", phase, "pane_index", existingPane.Index)
		// 既存のpaneを選択
		if err := e.tmuxManager.SelectPane(e.sessionName, windowName, existingPane.Index); err != nil {
			return nil, fmt.Errorf("failed to select existing pane: %w", err)
		}
		return existingPane, nil
	}

	// 新しいpaneを作成する必要がある
	e.logger.Info("Creating new pane", "phase", phase, "is_new_window", isNewWindow)

	// 新規ウィンドウの場合は、pane分割せずに既存のpane 0を使用
	if isNewWindow {
		e.logger.Info("Using existing pane for new window", "phase", phase)
		// 既存のpane 0のタイトルを設定
		if err := e.tmuxManager.SetPaneTitle(e.sessionName, windowName, 0, phase); err != nil {
			return nil, fmt.Errorf("failed to set pane title: %w", err)
		}
		return &tmuxpkg.PaneInfo{
			Index:  0,
			Title:  phase,
			Active: true,
		}, nil
	}

	// 既存ウィンドウの場合
	// フェーズに応じたpane作成オプション
	opts := tmuxpkg.PaneOptions{
		Split:      "-h", // 水平分割（縦分割）
		Percentage: 50,   // 50%で分割
		Title:      phase,
	}

	// 最初のフェーズ（Plan）の場合は、既存のpane（index 0）を使用
	if phase == "Plan" {
		// 既存のpane 0のタイトルを設定
		if err := e.tmuxManager.SetPaneTitle(e.sessionName, windowName, 0, phase); err != nil {
			return nil, fmt.Errorf("failed to set pane title: %w", err)
		}
		return &tmuxpkg.PaneInfo{
			Index:  0,
			Title:  phase,
			Active: true,
		}, nil
	}

	// Plan以外のフェーズでは新しいpaneを作成
	newPane, err := e.tmuxManager.CreatePane(e.sessionName, windowName, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create pane: %w", err)
	}

	return newPane, nil
}

// ExecuteInWorkspace はワークスペース内でコマンドを実行する
func (e *BaseExecutor) ExecuteInWorkspace(workspace *WorkspaceInfo, command string) error {
	// worktreeディレクトリに移動してコマンドを実行
	cdCommand := fmt.Sprintf("cd %s && %s", workspace.WorktreePath, command)

	// RunInWindowを使用してコマンドを実行（自動的にEnterキーが送信される）
	if err := e.tmuxManager.RunInWindow(e.sessionName, workspace.WindowName, cdCommand); err != nil {
		return fmt.Errorf("failed to execute command in workspace: %w", err)
	}

	return nil
}

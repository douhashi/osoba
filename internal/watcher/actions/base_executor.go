package actions

import (
	"context"
	"fmt"

	"github.com/douhashi/osoba/internal/config"
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

// BaseExecutor は各ActionExecutorの共通機能を提供する構造体
type BaseExecutor struct {
	sessionName     string
	tmuxManager     tmuxpkg.Manager
	worktreeManager git.WorktreeManager
	config          *config.Config
	logger          logger.Logger
}

// NewBaseExecutor は新しいBaseExecutorを作成する
func NewBaseExecutor(
	sessionName string,
	tmuxManager tmuxpkg.Manager,
	worktreeManager git.WorktreeManager,
	cfg *config.Config,
	logger logger.Logger,
) *BaseExecutor {
	return &BaseExecutor{
		sessionName:     sessionName,
		tmuxManager:     tmuxManager,
		worktreeManager: worktreeManager,
		config:          cfg,
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

	// セッションの存在確認と自動作成
	sessionExists, err := e.tmuxManager.SessionExists(e.sessionName)
	if err != nil {
		return nil, fmt.Errorf("failed to check session existence: %w", err)
	}

	if !sessionExists {
		e.logger.Info("Session does not exist, creating new session", "session_name", e.sessionName)
		if err := e.tmuxManager.EnsureSession(e.sessionName); err != nil {
			return nil, fmt.Errorf("failed to ensure session: %w", err)
		}
		e.logger.Info("Session created successfully", "session_name", e.sessionName)
	}

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

		// pane-base-indexを取得
		baseIndex, err := e.tmuxManager.GetPaneBaseIndex()
		if err != nil {
			// エラーの場合はデフォルト値の0を使用
			e.logger.Warn("Failed to get pane-base-index, using default 0", "error", err)
			baseIndex = 0
		}
		e.logger.Info("Got pane-base-index", "baseIndex", baseIndex)

		// 既存のpaneのタイトルを設定
		if err := e.tmuxManager.SetPaneTitle(e.sessionName, windowName, baseIndex, phase); err != nil {
			return nil, fmt.Errorf("failed to set pane title: %w", err)
		}
		return &tmuxpkg.PaneInfo{
			Index:  baseIndex,
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
		// pane-base-indexを取得
		baseIndex, err := e.tmuxManager.GetPaneBaseIndex()
		if err != nil {
			// エラーの場合はデフォルト値の0を使用
			e.logger.Warn("Failed to get pane-base-index, using default 0", "error", err)
			baseIndex = 0
		}
		e.logger.Info("Got pane-base-index", "baseIndex", baseIndex)

		// 既存のpaneのタイトルを設定
		if err := e.tmuxManager.SetPaneTitle(e.sessionName, windowName, baseIndex, phase); err != nil {
			return nil, fmt.Errorf("failed to set pane title: %w", err)
		}
		return &tmuxpkg.PaneInfo{
			Index:  baseIndex,
			Title:  phase,
			Active: true,
		}, nil
	}

	// ペイン数制限のチェック（新規ペイン作成前）
	if e.config != nil && e.config.Tmux.LimitPanesEnabled {
		panes, err := e.tmuxManager.ListPanes(e.sessionName, windowName)
		if err != nil {
			e.logger.Warn("Failed to list panes for limit check", "error", err)
		} else {
			maxPanes := e.config.Tmux.MaxPanesPerWindow
			if maxPanes <= 0 {
				maxPanes = 3 // デフォルト値
			}

			if len(panes) >= maxPanes {
				e.logger.Info("Pane limit reached, removing oldest non-active pane",
					"current_count", len(panes),
					"max_panes", maxPanes)

				// 最古の非アクティブペインを探す
				var oldestNonActiveIndex int = -1
				for _, pane := range panes {
					if !pane.Active {
						if oldestNonActiveIndex == -1 || pane.Index < oldestNonActiveIndex {
							oldestNonActiveIndex = pane.Index
						}
					}
				}

				// 非アクティブペインが見つかった場合は削除
				if oldestNonActiveIndex >= 0 {
					e.logger.Info("Removing pane",
						"pane_index", oldestNonActiveIndex,
						"window", windowName)
					if err := e.tmuxManager.KillPane(e.sessionName, windowName, oldestNonActiveIndex); err != nil {
						e.logger.Warn("Failed to kill pane", "error", err, "pane_index", oldestNonActiveIndex)
					}
				} else {
					e.logger.Warn("All panes are active, skipping removal")
				}
			}
		}
	}

	// Plan以外のフェーズでは新しいpaneを作成
	newPane, err := e.tmuxManager.CreatePane(e.sessionName, windowName, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create pane: %w", err)
	}

	// 自動リサイズが有効の場合、ペイン作成後にリサイズを実行
	if e.config != nil && e.config.Tmux.AutoResizePanes {
		if err := e.tmuxManager.ResizePanesEvenly(e.sessionName, windowName); err != nil {
			// リサイズの失敗はログに記録するが、処理は継続
			e.logger.Warn("Failed to resize panes automatically", "error", err, "window", windowName)
		} else {
			e.logger.Info("Auto-resized panes evenly", "window", windowName, "session", e.sessionName)
		}
	}

	return newPane, nil
}

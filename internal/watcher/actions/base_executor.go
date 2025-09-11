package actions

import (
	"context"
	"fmt"
	"sync"
	"time"

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

const (
	// autoResizeDebounceInterval はリサイズ実行の最小間隔
	autoResizeDebounceInterval = 500 * time.Millisecond
)

// BaseExecutor は各ActionExecutorの共通機能を提供する構造体
type BaseExecutor struct {
	sessionName     string
	tmuxManager     tmuxpkg.Manager
	worktreeManager git.WorktreeManager
	config          *config.Config
	logger          logger.Logger
	// リサイズのデバウンス機能
	lastResizeTime map[string]time.Time
	resizeMutex    sync.Mutex
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
		lastResizeTime:  make(map[string]time.Time),
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

		// 既存ペイン使用時もリサイズを実行
		e.executeAutoResize(windowName)

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

		// 新規ウィンドウでも自動リサイズを実行
		e.executeAutoResize(windowName)

		return &tmuxpkg.PaneInfo{
			Index:  baseIndex,
			Title:  phase,
			Active: true,
		}, nil
	}

	// 既存ウィンドウの場合
	// フェーズに応じたpane作成オプション
	var paneConfig *tmuxpkg.PaneConfig
	if e.config != nil && e.config.Tmux.LimitPanesEnabled {
		paneConfig = &tmuxpkg.PaneConfig{
			LimitPanesEnabled: e.config.Tmux.LimitPanesEnabled,
			MaxPanesPerWindow: e.config.Tmux.MaxPanesPerWindow,
		}
	}

	opts := tmuxpkg.PaneOptions{
		Split:      "-h", // 水平分割（縦分割）
		Percentage: 50,   // 50%で分割
		Title:      phase,
		Config:     paneConfig,
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

		// Planフェーズで既存ペイン使用時もリサイズを実行
		e.executeAutoResize(windowName)

		return &tmuxpkg.PaneInfo{
			Index:  baseIndex,
			Title:  phase,
			Active: true,
		}, nil
	}

	// Plan以外のフェーズでは新しいpaneを作成
	// CreatePane内でペイン数制限とレイアウト調整が行われる
	newPane, err := e.tmuxManager.CreatePane(e.sessionName, windowName, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create pane: %w", err)
	}

	// ペイン作成後に自動リサイズを実行（CreatePane内でも行われるが、デバウンス機能のため追加実行）
	e.executeAutoResize(windowName)

	return newPane, nil
}

// executeAutoResize はデバウンス機能付きでペインの自動リサイズを実行する
func (e *BaseExecutor) executeAutoResize(windowName string) {
	// AutoResizePanesが無効な場合は何もしない
	if e.config == nil || !e.config.Tmux.AutoResizePanes {
		return
	}

	e.resizeMutex.Lock()
	defer e.resizeMutex.Unlock()

	now := time.Now()
	lastTime, exists := e.lastResizeTime[windowName]

	// デバウンス期間内の場合はスキップ
	if exists && now.Sub(lastTime) < autoResizeDebounceInterval {
		e.logger.Debug("Skipping resize due to debounce",
			"window", windowName,
			"time_since_last", now.Sub(lastTime))
		return
	}

	// リサイズを実行
	if err := e.tmuxManager.ResizePanesEvenly(e.sessionName, windowName); err != nil {
		// リサイズの失敗はログに記録するが、処理は継続
		e.logger.Warn("Failed to resize panes automatically", "error", err, "window", windowName)
	} else {
		e.logger.Info("Auto-resized panes evenly", "window", windowName, "session", e.sessionName)
	}

	// 最後のリサイズ時刻を更新
	e.lastResizeTime[windowName] = now
}

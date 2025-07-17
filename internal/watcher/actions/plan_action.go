package actions

import (
	"context"
	"fmt"
	"log"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
	"github.com/douhashi/osoba/internal/tmux"
	"github.com/douhashi/osoba/internal/types"
)

// TmuxClient はtmux操作のインターフェース
type TmuxClient interface {
	CreateWindowForIssue(sessionName string, issueNumber int, phase string) error
	SwitchToIssueWindow(sessionName string, issueNumber int, phase string) error
	WindowExists(sessionName, windowName string) (bool, error)
}

// StateManager は状態管理のインターフェース
type StateManager interface {
	GetState(issueNumber int64) (*types.IssueState, bool)
	SetState(issueNumber int64, phase types.IssuePhase, status types.IssueStatus)
	IsProcessing(issueNumber int64) bool
	HasBeenProcessed(issueNumber int64, phase types.IssuePhase) bool
	MarkAsCompleted(issueNumber int64, phase types.IssuePhase)
	MarkAsFailed(issueNumber int64, phase types.IssuePhase)
}

// DefaultTmuxClient はデフォルトのtmuxクライアント実装
type DefaultTmuxClient struct{}

func (c *DefaultTmuxClient) CreateWindowForIssue(sessionName string, issueNumber int, phase string) error {
	return tmux.CreateWindowForIssueWithExecutor(sessionName, issueNumber, phase, &tmux.DefaultCommandExecutor{})
}

func (c *DefaultTmuxClient) SwitchToIssueWindow(sessionName string, issueNumber int, phase string) error {
	return tmux.SwitchToIssueWindowWithExecutor(sessionName, issueNumber, phase, &tmux.DefaultCommandExecutor{})
}

func (c *DefaultTmuxClient) WindowExists(sessionName, windowName string) (bool, error) {
	return tmux.WindowExists(sessionName, windowName)
}

// PlanAction は計画フェーズのアクション実装
type PlanAction struct {
	types.BaseAction
	sessionName       string
	tmuxClient        TmuxClient
	stateManager      StateManager
	phaseTransitioner PhaseTransitioner
	worktreeManager   git.WorktreeManager
	claudeExecutor    claude.ClaudeExecutor
	claudeConfig      *claude.ClaudeConfig
	logger            logger.Logger
}

// NewPlanAction は新しいPlanActionを作成する
func NewPlanAction(
	sessionName string,
	tmuxClient TmuxClient,
	stateManager StateManager,
	worktreeManager git.WorktreeManager,
	claudeExecutor claude.ClaudeExecutor,
	claudeConfig *claude.ClaudeConfig,
) *PlanAction {
	return &PlanAction{
		BaseAction:      types.BaseAction{Type: types.ActionTypePlan},
		sessionName:     sessionName,
		tmuxClient:      tmuxClient,
		stateManager:    stateManager,
		worktreeManager: worktreeManager,
		claudeExecutor:  claudeExecutor,
		claudeConfig:    claudeConfig,
	}
}

// NewPlanActionWithPhaseTransitioner は新しいPlanActionをPhaseTransitionerと共に作成する
func NewPlanActionWithPhaseTransitioner(
	sessionName string,
	tmuxClient TmuxClient,
	stateManager StateManager,
	phaseTransitioner PhaseTransitioner,
	worktreeManager git.WorktreeManager,
	claudeExecutor claude.ClaudeExecutor,
	claudeConfig *claude.ClaudeConfig,
) *PlanAction {
	return &PlanAction{
		BaseAction:        types.BaseAction{Type: types.ActionTypePlan},
		sessionName:       sessionName,
		tmuxClient:        tmuxClient,
		stateManager:      stateManager,
		phaseTransitioner: phaseTransitioner,
		worktreeManager:   worktreeManager,
		claudeExecutor:    claudeExecutor,
		claudeConfig:      claudeConfig,
	}
}

// NewPlanActionWithLogger はloggerを注入したPlanActionを作成する
func NewPlanActionWithLogger(
	sessionName string,
	tmuxClient TmuxClient,
	stateManager StateManager,
	worktreeManager git.WorktreeManager,
	claudeExecutor claude.ClaudeExecutor,
	claudeConfig *claude.ClaudeConfig,
	logger logger.Logger,
) *PlanAction {
	return &PlanAction{
		BaseAction:      types.BaseAction{Type: types.ActionTypePlan},
		sessionName:     sessionName,
		tmuxClient:      tmuxClient,
		stateManager:    stateManager,
		worktreeManager: worktreeManager,
		claudeExecutor:  claudeExecutor,
		claudeConfig:    claudeConfig,
		logger:          logger,
	}
}

// logInfo はloggerが設定されている場合は構造化ログを、設定されていない場合は標準ログを出力する
func (a *PlanAction) logInfo(msg string, keysAndValues ...interface{}) {
	if a.logger != nil {
		a.logger.Info(msg, keysAndValues...)
	} else {
		// 後方互換性のため、標準ログ出力を維持
		if len(keysAndValues) >= 2 {
			// 特別なケースの処理
			var issueNumber interface{}
			var path interface{}
			for i := 0; i < len(keysAndValues); i += 2 {
				if keysAndValues[i] == "issue_number" {
					issueNumber = keysAndValues[i+1]
				} else if keysAndValues[i] == "path" {
					path = keysAndValues[i+1]
				}
			}

			// pathとissue_numberがある場合
			if path != nil && msg == "Worktree created" {
				log.Printf("%s at: %v", msg, path)
				return
			}

			// issue_numberがある場合は既存のフォーマットを使用
			if issueNumber != nil {
				log.Printf("%s for issue #%v", msg, issueNumber)
				return
			}
		}
		log.Print(msg)
	}
}

// Execute は計画フェーズのアクションを実行する
func (a *PlanAction) Execute(ctx context.Context, issue *github.Issue) error {
	if issue == nil || issue.Number == nil {
		return fmt.Errorf("invalid issue")
	}

	issueNumber := int64(*issue.Number)
	a.logInfo("Executing plan action", "issue_number", issueNumber)

	// 既に処理済みかチェック
	if a.stateManager.HasBeenProcessed(issueNumber, types.IssueStatePlan) {
		a.logInfo("Issue has already been processed for plan phase", "issue_number", issueNumber)
		return nil
	}

	// 処理中かチェック
	if a.stateManager.IsProcessing(issueNumber) {
		return fmt.Errorf("issue #%d is already processing", issueNumber)
	}

	// 処理開始
	a.stateManager.SetState(issueNumber, types.IssueStatePlan, types.IssueStatusProcessing)

	// tmuxウィンドウ作成
	if err := a.tmuxClient.CreateWindowForIssue(a.sessionName, int(issueNumber), "plan"); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStatePlan)
		return fmt.Errorf("failed to create tmux window: %w", err)
	}

	// mainブランチを最新化
	a.logInfo("Updating main branch", "issue_number", issueNumber)
	if err := a.worktreeManager.UpdateMainBranch(ctx); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStatePlan)
		return fmt.Errorf("failed to update main branch: %w", err)
	}

	// worktree作成
	a.logInfo("Creating worktree", "issue_number", issueNumber)
	if err := a.worktreeManager.CreateWorktree(ctx, int(issueNumber), git.PhasePlan); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStatePlan)
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// worktreeパスを取得
	worktreePath := a.worktreeManager.GetWorktreePath(int(issueNumber), git.PhasePlan)
	a.logInfo("Worktree created", "issue_number", issueNumber, "path", worktreePath)

	// Claude実行用の変数を準備
	templateVars := &claude.TemplateVariables{
		IssueNumber: int(issueNumber),
		IssueTitle:  getIssueTitle(issue),
		RepoName:    getRepoName(),
	}

	// Claude設定を取得
	phaseConfig, exists := a.claudeConfig.GetPhase("plan")
	if !exists {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStatePlan)
		return fmt.Errorf("plan phase config not found")
	}

	// tmuxウィンドウ内でClaude実行
	windowName := fmt.Sprintf("%d-plan", issueNumber)
	a.logInfo("Executing Claude in tmux window", "issue_number", issueNumber, "window_name", windowName)
	if err := a.claudeExecutor.ExecuteInTmux(ctx, phaseConfig, templateVars, a.sessionName, windowName, worktreePath); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStatePlan)
		return fmt.Errorf("failed to execute claude: %w", err)
	}

	// 処理完了
	a.stateManager.MarkAsCompleted(issueNumber, types.IssueStatePlan)
	a.logInfo("Successfully completed plan action", "issue_number", issueNumber)

	return nil
}

// CanExecute は計画フェーズのアクションが実行可能かを判定する
func (a *PlanAction) CanExecute(issue *github.Issue) bool {
	return hasLabel(issue, "status:needs-plan")
}

// hasLabel はIssueが指定されたラベルを持っているかを確認する
func hasLabel(issue *github.Issue, labelName string) bool {
	if issue == nil || issue.Labels == nil {
		return false
	}

	for _, label := range issue.Labels {
		if label.Name != nil && *label.Name == labelName {
			return true
		}
	}
	return false
}

// getIssueTitle はIssueのタイトルを取得する
func getIssueTitle(issue *github.Issue) string {
	if issue == nil || issue.Title == nil {
		return ""
	}
	return *issue.Title
}

// getRepoName はリポジトリ名を取得する（現在は固定値）
func getRepoName() string {
	// TODO: 実際のリポジトリ名を動的に取得
	return "douhashi/osoba"
}

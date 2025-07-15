package actions

import (
	"context"
	"fmt"
	"log"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/tmux"
	"github.com/douhashi/osoba/internal/types"
	"github.com/google/go-github/v67/github"
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

// Execute は計画フェーズのアクションを実行する
func (a *PlanAction) Execute(ctx context.Context, issue *github.Issue) error {
	if issue == nil || issue.Number == nil {
		return fmt.Errorf("invalid issue")
	}

	issueNumber := int64(*issue.Number)
	log.Printf("Executing plan action for issue #%d", issueNumber)

	// 既に処理済みかチェック
	if a.stateManager.HasBeenProcessed(issueNumber, types.IssueStatePlan) {
		log.Printf("Issue #%d has already been processed for plan phase", issueNumber)
		return nil
	}

	// 処理中かチェック
	if a.stateManager.IsProcessing(issueNumber) {
		return fmt.Errorf("issue #%d is already processing", issueNumber)
	}

	// 処理開始
	a.stateManager.SetState(issueNumber, types.IssueStatePlan, types.IssueStatusProcessing)

	// ラベル遷移（status:needs-plan → status:planning）
	// PhaseTransitionerがある場合はそれを使用
	if a.phaseTransitioner != nil {
		if err := a.phaseTransitioner.TransitionPhase(ctx, int(issueNumber), "plan", "status:needs-plan", "status:planning"); err != nil {
			a.stateManager.MarkAsFailed(issueNumber, types.IssueStatePlan)
			return fmt.Errorf("failed to transition phase: %w", err)
		}
	}

	// tmuxウィンドウ作成
	if err := a.tmuxClient.CreateWindowForIssue(a.sessionName, int(issueNumber), "plan"); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStatePlan)
		return fmt.Errorf("failed to create tmux window: %w", err)
	}

	// mainブランチを最新化
	log.Printf("Updating main branch for issue #%d", issueNumber)
	if err := a.worktreeManager.UpdateMainBranch(ctx); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStatePlan)
		return fmt.Errorf("failed to update main branch: %w", err)
	}

	// worktree作成
	log.Printf("Creating worktree for issue #%d", issueNumber)
	if err := a.worktreeManager.CreateWorktree(ctx, int(issueNumber), git.PhasePlan); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStatePlan)
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// worktreeパスを取得
	worktreePath := a.worktreeManager.GetWorktreePath(int(issueNumber), git.PhasePlan)
	log.Printf("Worktree created at: %s", worktreePath)

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
	log.Printf("Executing Claude in tmux window for issue #%d", issueNumber)
	if err := a.claudeExecutor.ExecuteInTmux(ctx, phaseConfig, templateVars, a.sessionName, windowName, worktreePath); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStatePlan)
		return fmt.Errorf("failed to execute claude: %w", err)
	}

	// 処理完了
	a.stateManager.MarkAsCompleted(issueNumber, types.IssueStatePlan)
	log.Printf("Successfully completed plan action for issue #%d", issueNumber)

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

package watcher

import (
	"fmt"
	
	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/watcher/actions"
)

// ActionFactory はアクションを作成するファクトリーインターフェース
type ActionFactory interface {
	CreatePlanAction() ActionExecutor
	CreateImplementationAction() ActionExecutor
	CreateReviewAction() ActionExecutor
}

// DefaultActionFactory はデフォルトのActionFactory実装
type DefaultActionFactory struct {
	sessionName     string
	ghClient        github.GitHubClient
	worktreeManager git.WorktreeManager
	claudeExecutor  claude.ClaudeExecutor
	claudeConfig    *claude.ClaudeConfig
	stateManager    *IssueStateManager
	config          *config.Config
	owner           string
	repo            string
}

// NewDefaultActionFactory は新しいDefaultActionFactoryを作成する
func NewDefaultActionFactory(
	sessionName string,
	ghClient github.GitHubClient,
	worktreeManager git.WorktreeManager,
	claudeExecutor claude.ClaudeExecutor,
	claudeConfig *claude.ClaudeConfig,
	cfg *config.Config,
	owner string,
	repo string,
) *DefaultActionFactory {
	return &DefaultActionFactory{
		sessionName:     sessionName,
		ghClient:        ghClient,
		worktreeManager: worktreeManager,
		claudeExecutor:  claudeExecutor,
		claudeConfig:    claudeConfig,
		stateManager:    NewIssueStateManager(),
		config:          cfg,
		owner:           owner,
		repo:            repo,
	}
}

// CreatePlanAction は計画フェーズのアクションを作成する
func (f *DefaultActionFactory) CreatePlanAction() ActionExecutor {
	// GitHubClientインターフェースを使用してLabelTransitionerを作成
	var labelTransitioner github.LabelTransitioner
	if apiClient, ok := f.ghClient.(*github.Client); ok {
		// APIクライアントの場合
		fmt.Printf("DEBUG: Using GitHub API client for PlanAction\n")
		issuesService := apiClient.GetIssuesService()
		if issuesService != nil {
			labelTransitioner = github.NewLabelTransitioner(issuesService, f.owner, f.repo)
		}
	} else {
		// ghクライアントの場合、GitHubClientインターフェースを直接使用
		fmt.Printf("DEBUG: Using gh command client for PlanAction\n")
		labelTransitioner = github.NewLabelTransitionerFromGitHubClient(f.ghClient, f.owner, f.repo)
	}

	// GitHubAdapterを作成（labelTransitionerがnilでも、ghClientを使用するため問題なし）
	githubAdapter := actions.NewGitHubAdapter(f.ghClient, f.owner, f.repo, labelTransitioner)

	// ConfigAdapterを作成
	configAdapter := actions.NewConfigAdapter(f.config)

	// PhaseTransitionerを作成
	phaseTransitioner := actions.NewPhaseTransitioner(f.owner, f.repo, githubAdapter, configAdapter)
	fmt.Printf("DEBUG: PhaseTransitioner created for PlanAction\n")

	return actions.NewPlanActionWithPhaseTransitioner(
		f.sessionName,
		&actions.DefaultTmuxClient{},
		f.stateManager,
		phaseTransitioner,
		f.worktreeManager,
		f.claudeExecutor,
		f.claudeConfig,
	)
}

// CreateImplementationAction は実装フェーズのアクションを作成する
func (f *DefaultActionFactory) CreateImplementationAction() ActionExecutor {
	labelManager := &actions.DefaultLabelManager{
		GitHubClient: f.ghClient,
	}

	// GitHubClientインターフェースを使用してLabelTransitionerを作成
	var labelTransitioner github.LabelTransitioner
	if apiClient, ok := f.ghClient.(*github.Client); ok {
		// APIクライアントの場合
		issuesService := apiClient.GetIssuesService()
		if issuesService != nil {
			labelTransitioner = github.NewLabelTransitioner(issuesService, f.owner, f.repo)
		}
	} else {
		// ghクライアントの場合、GitHubClientインターフェースを直接使用
		labelTransitioner = github.NewLabelTransitionerFromGitHubClient(f.ghClient, f.owner, f.repo)
	}

	// GitHubAdapterを作成
	githubAdapter := actions.NewGitHubAdapter(f.ghClient, f.owner, f.repo, labelTransitioner)

	// ConfigAdapterを作成
	configAdapter := actions.NewConfigAdapter(f.config)

	// PhaseTransitionerを作成
	phaseTransitioner := actions.NewPhaseTransitioner(f.owner, f.repo, githubAdapter, configAdapter)

	return actions.NewImplementationActionWithPhaseTransitioner(
		f.sessionName,
		&actions.DefaultTmuxClient{},
		f.stateManager,
		labelManager,
		phaseTransitioner,
		f.worktreeManager,
		f.claudeExecutor,
		f.claudeConfig,
	)
}

// CreateReviewAction はレビューフェーズのアクションを作成する
func (f *DefaultActionFactory) CreateReviewAction() ActionExecutor {
	labelManager := &actions.DefaultLabelManager{
		GitHubClient: f.ghClient,
	}

	// GitHubClientインターフェースを使用してLabelTransitionerを作成
	var labelTransitioner github.LabelTransitioner
	if apiClient, ok := f.ghClient.(*github.Client); ok {
		// APIクライアントの場合
		issuesService := apiClient.GetIssuesService()
		if issuesService != nil {
			labelTransitioner = github.NewLabelTransitioner(issuesService, f.owner, f.repo)
		}
	} else {
		// ghクライアントの場合、GitHubClientインターフェースを直接使用
		labelTransitioner = github.NewLabelTransitionerFromGitHubClient(f.ghClient, f.owner, f.repo)
	}

	// GitHubAdapterを作成
	githubAdapter := actions.NewGitHubAdapter(f.ghClient, f.owner, f.repo, labelTransitioner)

	// ConfigAdapterを作成
	configAdapter := actions.NewConfigAdapter(f.config)

	// PhaseTransitionerを作成
	phaseTransitioner := actions.NewPhaseTransitioner(f.owner, f.repo, githubAdapter, configAdapter)

	return actions.NewReviewActionWithPhaseTransitioner(
		f.sessionName,
		&actions.DefaultTmuxClient{},
		f.stateManager,
		labelManager,
		phaseTransitioner,
		f.worktreeManager,
		f.claudeExecutor,
		f.claudeConfig,
	)
}

// MockActionFactory はテスト用のモックファクトリー（action_manager_test.go で定義済み）

// NewActionManagerWithFactory はActionFactoryを使用してActionManagerを作成する
func NewActionManagerWithFactory(sessionName string, factory ActionFactory) *ActionManager {
	return &ActionManager{
		sessionName:   sessionName,
		stateManager:  NewIssueStateManager(),
		actionFactory: factory,
	}
}

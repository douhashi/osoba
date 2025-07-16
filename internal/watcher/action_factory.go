package watcher

import (
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
	// 現在の実装ではPhaseTransitionerを使用せず、シンプルな実装を使用
	return actions.NewPlanAction(
		f.sessionName,
		&actions.DefaultTmuxClient{},
		f.stateManager,
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

	// 現在の実装ではPhaseTransitionerを使用せず、シンプルな実装を使用
	return actions.NewImplementationAction(
		f.sessionName,
		&actions.DefaultTmuxClient{},
		f.stateManager,
		labelManager,
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

	// 現在の実装ではPhaseTransitionerを使用せず、シンプルな実装を使用
	return actions.NewReviewAction(
		f.sessionName,
		&actions.DefaultTmuxClient{},
		f.stateManager,
		labelManager,
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

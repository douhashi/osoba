package watcher

import (
	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
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
	logger          logger.Logger
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

// NewDefaultActionFactoryWithLogger はloggerを含む新しいDefaultActionFactoryを作成する
func NewDefaultActionFactoryWithLogger(
	sessionName string,
	ghClient github.GitHubClient,
	worktreeManager git.WorktreeManager,
	claudeExecutor claude.ClaudeExecutor,
	claudeConfig *claude.ClaudeConfig,
	cfg *config.Config,
	owner string,
	repo string,
	logger logger.Logger,
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
		logger:          logger,
	}
}

// CreatePlanAction は計画フェーズのアクションを作成する
func (f *DefaultActionFactory) CreatePlanAction() ActionExecutor {
	if f.logger != nil {
		// loggerが設定されている場合は、logger付きのActionを作成
		return actions.NewPlanActionWithLogger(
			f.sessionName,
			&actions.DefaultTmuxClient{},
			NewStateManagerAdapter(f.stateManager),
			f.worktreeManager,
			f.claudeExecutor,
			f.claudeConfig,
			f.logger.WithFields("component", "PlanAction"),
		)
	}
	// 既存の実装との互換性のため、loggerがない場合は従来の方法で作成
	return actions.NewPlanAction(
		f.sessionName,
		&actions.DefaultTmuxClient{},
		NewStateManagerAdapter(f.stateManager),
		f.worktreeManager,
		f.claudeExecutor,
		f.claudeConfig,
	)
}

// CreateImplementationAction は実装フェーズのアクションを作成する
func (f *DefaultActionFactory) CreateImplementationAction() ActionExecutor {
	labelManager := &actions.DefaultLabelManager{
		GitHubClient: f.ghClient,
		Owner:        f.owner,
		Repo:         f.repo,
	}

	if f.logger != nil {
		// loggerが設定されている場合は、logger付きのActionを作成
		return actions.NewImplementationActionWithLogger(
			f.sessionName,
			&actions.DefaultTmuxClient{},
			NewStateManagerAdapter(f.stateManager),
			labelManager,
			f.worktreeManager,
			f.claudeExecutor,
			f.claudeConfig,
			f.logger.WithFields("component", "ImplementationAction"),
		)
	}
	// 既存の実装との互換性のため、loggerがない場合は従来の方法で作成
	return actions.NewImplementationAction(
		f.sessionName,
		&actions.DefaultTmuxClient{},
		NewStateManagerAdapter(f.stateManager),
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
		Owner:        f.owner,
		Repo:         f.repo,
	}

	if f.logger != nil {
		// loggerが設定されている場合は、logger付きのActionを作成
		return actions.NewReviewActionWithLogger(
			f.sessionName,
			&actions.DefaultTmuxClient{},
			NewStateManagerAdapter(f.stateManager),
			labelManager,
			f.worktreeManager,
			f.claudeExecutor,
			f.claudeConfig,
			f.logger.WithFields("component", "ReviewAction"),
		)
	}
	// 既存の実装との互換性のため、loggerがない場合は従来の方法で作成
	return actions.NewReviewAction(
		f.sessionName,
		&actions.DefaultTmuxClient{},
		NewStateManagerAdapter(f.stateManager),
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

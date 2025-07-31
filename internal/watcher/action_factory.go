package watcher

import (
	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
	"github.com/douhashi/osoba/internal/tmux"
	"github.com/douhashi/osoba/internal/watcher/actions"
)

// ActionFactory はアクションを作成するファクトリーインターフェース
type ActionFactory interface {
	CreatePlanAction() ActionExecutor
	CreateImplementationAction() ActionExecutor
	CreateReviewAction() ActionExecutor
}

// DefaultActionFactory はpane管理方式を使用するActionFactory実装
type DefaultActionFactory struct {
	sessionName     string
	ghClient        github.GitHubClient
	tmuxManager     tmux.Manager
	worktreeManager git.WorktreeManager
	claudeExecutor  claude.ClaudeExecutor
	commandBuilder  *claude.DefaultCommandBuilder
	claudeConfig    *claude.ClaudeConfig
	config          *config.Config
	owner           string
	repo            string
	logger          logger.Logger
}

// NewDefaultActionFactory は新しいDefaultActionFactoryを作成する
func NewDefaultActionFactory(
	sessionName string,
	ghClient github.GitHubClient,
	tmuxManager tmux.Manager,
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
		tmuxManager:     tmuxManager,
		worktreeManager: worktreeManager,
		claudeExecutor:  claudeExecutor,
		commandBuilder:  claude.NewCommandBuilder(),
		claudeConfig:    claudeConfig,
		config:          cfg,
		owner:           owner,
		repo:            repo,
		logger:          logger,
	}
}

// CreatePlanAction は計画フェーズのアクションを作成する
func (f *DefaultActionFactory) CreatePlanAction() ActionExecutor {
	return actions.NewPlanAction(
		f.sessionName,
		f.tmuxManager,
		f.worktreeManager,
		f.commandBuilder,
		f.claudeConfig,
		f.logger.WithFields("component", "PlanAction"),
	)
}

// CreateImplementationAction は実装フェーズのアクションを作成する
func (f *DefaultActionFactory) CreateImplementationAction() ActionExecutor {
	labelManager := &actions.DefaultLabelManager{
		GitHubClient: f.ghClient,
		Owner:        f.owner,
		Repo:         f.repo,
	}

	return actions.NewImplementationAction(
		f.sessionName,
		f.tmuxManager,
		labelManager,
		f.worktreeManager,
		f.commandBuilder,
		f.claudeConfig,
		f.logger.WithFields("component", "ImplementationAction"),
	)
}

// CreateReviewAction はレビューフェーズのアクションを作成する
func (f *DefaultActionFactory) CreateReviewAction() ActionExecutor {
	labelManager := &actions.DefaultLabelManager{
		GitHubClient: f.ghClient,
		Owner:        f.owner,
		Repo:         f.repo,
	}

	return actions.NewReviewAction(
		f.sessionName,
		f.tmuxManager,
		labelManager,
		f.worktreeManager,
		f.commandBuilder,
		f.claudeConfig,
		f.logger.WithFields("component", "ReviewAction"),
	)
}

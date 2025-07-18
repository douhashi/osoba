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

// DefaultActionFactoryV2 は新しいpane管理方式を使用するActionFactory実装
type DefaultActionFactoryV2 struct {
	sessionName     string
	ghClient        github.GitHubClient
	tmuxManager     tmux.Manager
	worktreeManager git.WorktreeManager
	claudeExecutor  claude.ClaudeExecutor
	commandBuilder  *claude.DefaultCommandBuilder
	claudeConfig    *claude.ClaudeConfig
	stateManager    *IssueStateManager
	config          *config.Config
	owner           string
	repo            string
	logger          logger.Logger
}

// NewDefaultActionFactoryV2 は新しいDefaultActionFactoryV2を作成する
func NewDefaultActionFactoryV2(
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
) *DefaultActionFactoryV2 {
	return &DefaultActionFactoryV2{
		sessionName:     sessionName,
		ghClient:        ghClient,
		tmuxManager:     tmuxManager,
		worktreeManager: worktreeManager,
		claudeExecutor:  claudeExecutor,
		commandBuilder:  claude.NewCommandBuilder(),
		claudeConfig:    claudeConfig,
		stateManager:    NewIssueStateManager(),
		config:          cfg,
		owner:           owner,
		repo:            repo,
		logger:          logger,
	}
}

// CreatePlanAction は計画フェーズのアクションを作成する
func (f *DefaultActionFactoryV2) CreatePlanAction() ActionExecutor {
	return actions.NewPlanActionV2(
		f.sessionName,
		f.tmuxManager,
		f.stateManager,
		f.worktreeManager,
		f.commandBuilder,
		f.claudeConfig,
		f.logger.WithFields("component", "PlanActionV2"),
	)
}

// CreateImplementationAction は実装フェーズのアクションを作成する
func (f *DefaultActionFactoryV2) CreateImplementationAction() ActionExecutor {
	labelManager := &actions.DefaultLabelManager{
		GitHubClient: f.ghClient,
		Owner:        f.owner,
		Repo:         f.repo,
	}

	return actions.NewImplementationActionV2(
		f.sessionName,
		f.tmuxManager,
		f.stateManager,
		labelManager,
		f.worktreeManager,
		f.commandBuilder,
		f.claudeConfig,
		f.logger.WithFields("component", "ImplementationActionV2"),
	)
}

// CreateReviewAction はレビューフェーズのアクションを作成する
func (f *DefaultActionFactoryV2) CreateReviewAction() ActionExecutor {
	labelManager := &actions.DefaultLabelManager{
		GitHubClient: f.ghClient,
		Owner:        f.owner,
		Repo:         f.repo,
	}

	return actions.NewReviewActionV2(
		f.sessionName,
		f.tmuxManager,
		f.stateManager,
		labelManager,
		f.worktreeManager,
		f.commandBuilder,
		f.claudeConfig,
		f.logger.WithFields("component", "ReviewActionV2"),
	)
}

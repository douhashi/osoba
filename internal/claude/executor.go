package claude

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"regexp"

	"github.com/douhashi/osoba/internal/logger"
)

// ClaudeExecutor はClaude実行を管理するインターフェース
type ClaudeExecutor interface {
	CheckClaudeExists() error
	BuildCommand(ctx context.Context, args []string, prompt string, workdir string) *exec.Cmd
	ExecuteInWorktree(ctx context.Context, config *PhaseConfig, vars *TemplateVariables, workdir string) error
	ExecuteInTmux(ctx context.Context, config *PhaseConfig, vars *TemplateVariables, sessionName, windowName, workdir string) error
}

// DefaultClaudeExecutor はClaudeExecutorのデフォルト実装
type DefaultClaudeExecutor struct {
	logger logger.Logger
}

// NewClaudeExecutor は新しいClaudeExecutorを作成する
func NewClaudeExecutor() ClaudeExecutor {
	return &DefaultClaudeExecutor{}
}

// NewClaudeExecutorWithLogger はロガーを指定して新しいClaudeExecutorを作成する
func NewClaudeExecutorWithLogger(logger logger.Logger) ClaudeExecutor {
	if logger == nil {
		return nil
	}
	return &DefaultClaudeExecutor{
		logger: logger,
	}
}

// CheckClaudeExists はclaudeコマンドが存在するかチェックする
func (e *DefaultClaudeExecutor) CheckClaudeExists() error {
	_, err := exec.LookPath("claude")
	if err != nil {
		if e.logger != nil {
			e.logger.Error("Claude command not found", "error", err)
		}
		return fmt.Errorf("claude command not found: %w", err)
	}
	return nil
}

// BuildCommand はClaude実行用のコマンドを構築する
func (e *DefaultClaudeExecutor) BuildCommand(ctx context.Context, args []string, prompt string, workdir string) *exec.Cmd {
	// 引数を結合
	cmdArgs := append(args, prompt)

	cmd := exec.CommandContext(ctx, "claude", cmdArgs...)
	cmd.Dir = workdir

	return cmd
}

// ExecuteInWorktree はworktree内でClaudeを実行する
func (e *DefaultClaudeExecutor) ExecuteInWorktree(ctx context.Context, config *PhaseConfig, vars *TemplateVariables, workdir string) error {
	// Claudeコマンドの存在確認
	if err := e.CheckClaudeExists(); err != nil {
		return err
	}

	// プロンプトを展開
	prompt := ExpandTemplate(config.Prompt, vars)

	// コマンドを構築
	cmd := e.BuildCommand(ctx, config.Args, prompt, workdir)

	if e.logger != nil {
		e.logger.Info("Executing Claude in worktree",
			"workdir", workdir,
			"issueNumber", vars.IssueNumber,
		)
		e.logger.Debug("Claude command details",
			"args", config.Args,
			"prompt", e.maskSensitiveData(prompt),
		)
	} else {
		// 互換性のためのフォールバック
		log.Printf("Executing Claude in worktree: %s", workdir)
		log.Printf("Command: claude %v %s", config.Args, prompt)
	}

	// コマンドを実行
	if err := cmd.Run(); err != nil {
		if e.logger != nil {
			e.logger.Error("Failed to execute Claude",
				"error", err,
				"workdir", workdir,
				"issueNumber", vars.IssueNumber,
			)
		}
		return fmt.Errorf("failed to execute claude: %w", err)
	}

	if e.logger != nil {
		e.logger.Info("Claude execution completed successfully",
			"workdir", workdir,
			"issueNumber", vars.IssueNumber,
		)
	} else {
		log.Printf("Claude execution completed successfully")
	}
	return nil
}

// ExecuteInTmux はtmuxウィンドウ内でClaudeを実行する
func (e *DefaultClaudeExecutor) ExecuteInTmux(ctx context.Context, config *PhaseConfig, vars *TemplateVariables, sessionName, windowName, workdir string) error {
	// Claudeコマンドの存在確認
	if err := e.CheckClaudeExists(); err != nil {
		return err
	}

	// プロンプトを展開
	prompt := ExpandTemplate(config.Prompt, vars)

	// tmuxコマンドを構築
	// send-keysを使ってコマンドを送信
	claudeCmd := fmt.Sprintf("cd %s && claude", workdir)
	for _, arg := range config.Args {
		claudeCmd += fmt.Sprintf(" %s", arg)
	}
	claudeCmd += fmt.Sprintf(" '%s'", prompt)

	tmuxCmd := exec.CommandContext(ctx, "tmux", "send-keys", "-t", fmt.Sprintf("%s:%s", sessionName, windowName), claudeCmd, "Enter")

	if e.logger != nil {
		e.logger.Info("Executing Claude in tmux window",
			"session", sessionName,
			"window", windowName,
			"workdir", workdir,
			"issueNumber", vars.IssueNumber,
		)
		e.logger.Debug("Claude command details",
			"command", e.maskSensitiveData(claudeCmd),
			"args", config.Args,
		)
	} else {
		// 互換性のためのフォールバック
		log.Printf("Executing Claude in tmux window: %s:%s", sessionName, windowName)
		log.Printf("Command: %s", claudeCmd)
	}

	// tmuxコマンドを実行
	if err := tmuxCmd.Run(); err != nil {
		if e.logger != nil {
			e.logger.Error("Failed to execute Claude in tmux",
				"error", err,
				"session", sessionName,
				"window", windowName,
				"issueNumber", vars.IssueNumber,
			)
		}
		return fmt.Errorf("failed to execute claude in tmux: %w", err)
	}

	if e.logger != nil {
		e.logger.Info("Claude command sent to tmux window successfully",
			"session", sessionName,
			"window", windowName,
			"issueNumber", vars.IssueNumber,
		)
	} else {
		log.Printf("Claude command sent to tmux window successfully")
	}
	return nil
}

// maskSensitiveData は機密情報をマスクする
func (e *DefaultClaudeExecutor) maskSensitiveData(data string) string {
	// GitHubトークンのマスキング (ghp_, github_pat_, ghs_)
	// ghp_: 36文字
	// github_pat_: 59文字 + 11文字のprefix = 70文字以上
	// ghs_: 36文字
	githubTokenRegex := regexp.MustCompile(`(ghp_[a-zA-Z0-9]{36}|github_pat_[a-zA-Z0-9_]{59,}|ghs_[a-zA-Z0-9]{36})`)
	data = githubTokenRegex.ReplaceAllString(data, "[GITHUB_TOKEN]")

	// APIキーのマスキング (sk-proj-で始まるパターン)
	apiKeyRegex := regexp.MustCompile(`(sk-proj-[a-zA-Z0-9-_]+)`)
	data = apiKeyRegex.ReplaceAllString(data, "[API_KEY]")

	// 一般的なAPIキーパターン (apikey=, api_key=, apiKey= など)
	genericAPIKeyRegex := regexp.MustCompile(`(?i)(api[_-]?key\s*[:=]\s*)(["']?[a-zA-Z0-9-_]{20,}["']?)`)
	data = genericAPIKeyRegex.ReplaceAllString(data, "${1}[MASKED]")

	// Bearerトークン
	bearerTokenRegex := regexp.MustCompile(`(?i)(bearer\s+)([a-zA-Z0-9-_.]+)`)
	data = bearerTokenRegex.ReplaceAllString(data, "${1}[TOKEN]")

	return data
}

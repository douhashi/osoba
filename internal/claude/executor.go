package claude

import (
	"context"
	"fmt"
	"log"
	"os/exec"
)

// ClaudeExecutor はClaude実行を管理するインターフェース
type ClaudeExecutor interface {
	CheckClaudeExists() error
	BuildCommand(ctx context.Context, args []string, prompt string, workdir string) *exec.Cmd
	ExecuteInWorktree(ctx context.Context, config *PhaseConfig, vars *TemplateVariables, workdir string) error
	ExecuteInTmux(ctx context.Context, config *PhaseConfig, vars *TemplateVariables, sessionName, windowName, workdir string) error
}

// DefaultClaudeExecutor はClaudeExecutorのデフォルト実装
type DefaultClaudeExecutor struct{}

// NewClaudeExecutor は新しいClaudeExecutorを作成する
func NewClaudeExecutor() ClaudeExecutor {
	return &DefaultClaudeExecutor{}
}

// CheckClaudeExists はclaudeコマンドが存在するかチェックする
func (e *DefaultClaudeExecutor) CheckClaudeExists() error {
	_, err := exec.LookPath("claude")
	if err != nil {
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

	log.Printf("Executing Claude in worktree: %s", workdir)
	log.Printf("Command: claude %v %s", config.Args, prompt)

	// コマンドを実行
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute claude: %w", err)
	}

	log.Printf("Claude execution completed successfully")
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

	log.Printf("Executing Claude in tmux window: %s:%s", sessionName, windowName)
	log.Printf("Command: %s", claudeCmd)

	// tmuxコマンドを実行
	if err := tmuxCmd.Run(); err != nil {
		return fmt.Errorf("failed to execute claude in tmux: %w", err)
	}

	log.Printf("Claude command sent to tmux window successfully")
	return nil
}

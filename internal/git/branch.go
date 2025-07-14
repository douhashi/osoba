package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/douhashi/osoba/internal/logger"
)

// BranchInfo はブランチの情報を表す構造体
type BranchInfo struct {
	Name      string
	IsCurrent bool
	Commit    string
}

// Branch はgitブランチ操作を管理する構造体
type Branch struct {
	logger  logger.Logger
	command *Command
}

// NewBranch は新しいBranchインスタンスを作成する
func NewBranch(logger logger.Logger) *Branch {
	return &Branch{
		logger:  logger,
		command: NewCommand(logger),
	}
}

// Create は新しいブランチを作成する
func (b *Branch) Create(ctx context.Context, repoPath, branchName, baseBranch string) error {
	logFields := []interface{}{
		"repoPath", repoPath,
		"branchName", branchName,
	}
	if baseBranch != "" {
		logFields = append(logFields, "baseBranch", baseBranch)
	}

	b.logger.Info("Creating git branch", logFields...)

	// ブランチ作成コマンドを構築
	args := []string{"branch", branchName}
	if baseBranch != "" {
		args = append(args, baseBranch)
	}

	// ブランチを作成
	output, err := b.command.Run(ctx, "git", args, repoPath)
	if err != nil {
		errorFields := append(logFields, "error", err.Error())
		b.logger.Error("Failed to create git branch", errorFields...)
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// 成功ログ
	successFields := append(logFields, "output", output)
	b.logger.Info("Git branch created successfully", successFields...)

	return nil
}

// Checkout は指定されたブランチに切り替える
func (b *Branch) Checkout(ctx context.Context, repoPath, branchName string, create bool) error {
	logFields := []interface{}{
		"repoPath", repoPath,
		"branchName", branchName,
		"create", create,
	}

	b.logger.Info("Checking out git branch", logFields...)

	// チェックアウトコマンドを構築
	args := []string{"checkout"}
	if create {
		args = append(args, "-b")
	}
	args = append(args, branchName)

	// ブランチをチェックアウト
	output, err := b.command.Run(ctx, "git", args, repoPath)
	if err != nil {
		errorFields := append(logFields, "error", err.Error())
		b.logger.Error("Failed to checkout git branch", errorFields...)
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	// 成功ログ
	successFields := append(logFields, "output", output)
	b.logger.Info("Git branch checked out successfully", successFields...)

	return nil
}

// List は全てのブランチの情報を取得する
func (b *Branch) List(ctx context.Context, repoPath string, includeRemote bool) ([]BranchInfo, error) {
	logFields := []interface{}{
		"repoPath", repoPath,
		"includeRemote", includeRemote,
	}

	b.logger.Info("Listing git branches", logFields...)

	// ブランチ一覧取得コマンドを構築
	args := []string{"branch", "--format=%(refname:short)|%(HEAD)|%(objectname)"}
	if includeRemote {
		args = append(args, "-a")
	}

	// ブランチ一覧を取得
	output, err := b.command.Run(ctx, "git", args, repoPath)
	if err != nil {
		errorFields := append(logFields, "error", err.Error())
		b.logger.Error("Failed to list git branches", errorFields...)
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	// 出力をパース
	branches := parseBranchList(output)

	// 成功ログ
	successFields := append(logFields, "count", len(branches))
	b.logger.Info("Git branches listed successfully", successFields...)

	// 各ブランチの詳細をデバッグログに出力
	for i, br := range branches {
		b.logger.Debug("Branch info",
			"index", i,
			"name", br.Name,
			"current", br.IsCurrent,
			"commit", br.Commit,
		)
	}

	return branches, nil
}

// parseBranchList はgit branch --formatの出力をパースする
func parseBranchList(output string) []BranchInfo {
	var branches []BranchInfo
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// フォーマット: name|HEAD|commit
		parts := strings.Split(line, "|")
		if len(parts) != 3 {
			continue
		}

		branch := BranchInfo{
			Name:      parts[0],
			IsCurrent: parts[1] == "*",
			Commit:    parts[2],
		}

		branches = append(branches, branch)
	}

	return branches
}

// Delete は指定されたブランチを削除する
func (b *Branch) Delete(ctx context.Context, repoPath, branchName string, force bool) error {
	logFields := []interface{}{
		"repoPath", repoPath,
		"branchName", branchName,
		"force", force,
	}

	b.logger.Info("Deleting git branch", logFields...)

	// 削除コマンドを構築
	args := []string{"branch"}
	if force {
		args = append(args, "-D")
	} else {
		args = append(args, "-d")
	}
	args = append(args, branchName)

	// ブランチを削除
	output, err := b.command.Run(ctx, "git", args, repoPath)
	if err != nil {
		errorFields := append(logFields, "error", err.Error())
		b.logger.Error("Failed to delete git branch", errorFields...)
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	// 成功ログ
	successFields := append(logFields, "output", output)
	b.logger.Info("Git branch deleted successfully", successFields...)

	return nil
}

// GetCurrent は現在のブランチ名を取得する
func (b *Branch) GetCurrent(ctx context.Context, repoPath string) (string, error) {
	// git branch --show-currentを実行
	output, err := b.command.Run(ctx, "git", []string{"branch", "--show-current"}, repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(output), nil
}

// Exists は指定されたブランチが存在するかを確認する
func (b *Branch) Exists(ctx context.Context, repoPath, branchName string) bool {
	// git show-ref --verify refs/heads/<branch>を実行
	args := []string{"show-ref", "--verify", fmt.Sprintf("refs/heads/%s", branchName)}
	_, err := b.command.Run(ctx, "git", args, repoPath)

	// エラーがなければブランチは存在する
	return err == nil
}

// GetUpstream は指定されたブランチの上流ブランチを取得する
func (b *Branch) GetUpstream(ctx context.Context, repoPath, branchName string) (string, error) {
	// git rev-parse --abbrev-ref <branch>@{upstream}を実行
	args := []string{"rev-parse", "--abbrev-ref", fmt.Sprintf("%s@{upstream}", branchName)}
	output, err := b.command.Run(ctx, "git", args, repoPath)
	if err != nil {
		// 上流ブランチが設定されていない場合はエラーではない
		if strings.Contains(err.Error(), "no upstream") {
			return "", nil
		}
		return "", fmt.Errorf("failed to get upstream branch: %w", err)
	}

	return strings.TrimSpace(output), nil
}

// SetUpstream は指定されたブランチの上流ブランチを設定する
func (b *Branch) SetUpstream(ctx context.Context, repoPath, branchName, upstream string) error {
	logFields := []interface{}{
		"repoPath", repoPath,
		"branchName", branchName,
		"upstream", upstream,
	}

	b.logger.Info("Setting upstream branch", logFields...)

	// git branch --set-upstream-to=<upstream> <branch>を実行
	args := []string{"branch", fmt.Sprintf("--set-upstream-to=%s", upstream), branchName}
	output, err := b.command.Run(ctx, "git", args, repoPath)
	if err != nil {
		errorFields := append(logFields, "error", err.Error())
		b.logger.Error("Failed to set upstream branch", errorFields...)
		return fmt.Errorf("failed to set upstream branch: %w", err)
	}

	// 成功ログ
	successFields := append(logFields, "output", output)
	b.logger.Info("Upstream branch set successfully", successFields...)

	return nil
}

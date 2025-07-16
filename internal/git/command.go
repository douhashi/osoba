package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/douhashi/osoba/internal/logger"
)

// Command はgitコマンド実行を管理する構造体
type Command struct {
	logger logger.Logger
}

// NewCommand は新しいCommandインスタンスを作成する
func NewCommand(logger logger.Logger) *Command {
	return &Command{
		logger: logger,
	}
}

// Run は指定されたgitコマンドを実行し、出力を返す
func (c *Command) Run(ctx context.Context, command string, args []string, workDir string) (string, error) {
	// ログフィールドを構築
	logFields := []interface{}{
		"command", command,
		"args", args,
	}
	if workDir != "" {
		logFields = append(logFields, "workDir", workDir)
	}

	// コマンド実行開始をログ出力（DEBUGレベルに変更）
	c.logger.Debug("Executing git command", logFields...)

	// コマンドを作成
	cmd := exec.CommandContext(ctx, command, args...)

	// 作業ディレクトリを設定
	if workDir != "" {
		cmd.Dir = workDir
	}

	// 出力用のバッファを準備
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// コマンドを実行
	err := cmd.Run()

	// 出力を文字列として取得
	stdoutStr := strings.TrimSpace(stdout.String())
	stderrStr := strings.TrimSpace(stderr.String())

	// 実行結果に基づいてログを出力
	if err != nil {
		// エラー時のログフィールド
		errorFields := append(logFields,
			"error", err.Error(),
			"stderr", truncateOutput(stderrStr, 1000),
		)
		if stdoutStr != "" {
			errorFields = append(errorFields, "stdout", truncateOutput(stdoutStr, 1000))
		}

		c.logger.Error("Git command failed", errorFields...)

		// エラーメッセージを構築
		if stderrStr != "" {
			return "", fmt.Errorf("git command failed: %w\nstderr: %s", err, stderrStr)
		}
		return "", fmt.Errorf("git command failed: %w", err)
	}

	// 成功時のログフィールド
	successFields := append(logFields, "duration", cmd.ProcessState.UserTime())

	// 出力がある場合は記録（大量の場合は要約）
	if stdoutStr != "" {
		successFields = append(successFields, "output", truncateOutput(stdoutStr, 500))
	}
	if stderrStr != "" {
		successFields = append(successFields, "stderr", truncateOutput(stderrStr, 500))
	}

	c.logger.Debug("Git command completed successfully", successFields...)

	// 標準出力を返す
	return stdoutStr, nil
}

// truncateOutput は長い出力を指定された長さに切り詰める
func truncateOutput(output string, maxLength int) string {
	if len(output) <= maxLength {
		return output
	}

	lines := strings.Split(output, "\n")
	if len(lines) > 10 {
		// 行数が多い場合は最初と最後の数行を表示
		result := strings.Join(lines[:5], "\n")
		result += fmt.Sprintf("\n... (%d lines omitted) ...\n", len(lines)-10)
		result += strings.Join(lines[len(lines)-5:], "\n")
		return result
	}

	// 行数が少ない場合は単純に切り詰め
	return output[:maxLength] + "... (truncated)"
}

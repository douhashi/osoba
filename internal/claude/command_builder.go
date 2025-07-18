package claude

import (
	"fmt"
	"strings"
)

// DefaultCommandBuilder はClaudeCommandBuilderの実装
type DefaultCommandBuilder struct{}

// NewCommandBuilder は新しいDefaultCommandBuilderを作成する
func NewCommandBuilder() *DefaultCommandBuilder {
	return &DefaultCommandBuilder{}
}

// BuildCommand はClaudeコマンドを構築する
func (b *DefaultCommandBuilder) BuildCommand(promptPath string, outputPath string, workdir string, vars interface{}) string {
	parts := []string{"claude"}

	// worddirが指定されている場合はオプションを追加
	if workdir != "" {
		parts = append(parts, fmt.Sprintf("--workdir=%s", workdir))
	}

	// outputPathが指定されている場合はオプションを追加
	if outputPath != "" {
		parts = append(parts, fmt.Sprintf("--output=%s", outputPath))
	}

	// promptPathを追加
	parts = append(parts, promptPath)

	// varsがTemplateVariablesの場合は変数を展開
	if templateVars, ok := vars.(*TemplateVariables); ok {
		// IssueNumberを追加
		if templateVars.IssueNumber > 0 {
			parts = append(parts, fmt.Sprintf("--issue-number=%d", templateVars.IssueNumber))
		}
		// IssueTitleを追加
		if templateVars.IssueTitle != "" {
			parts = append(parts, fmt.Sprintf("--issue-title=%q", templateVars.IssueTitle))
		}
		// RepoNameを追加
		if templateVars.RepoName != "" {
			parts = append(parts, fmt.Sprintf("--repo-name=%s", templateVars.RepoName))
		}
	}

	return strings.Join(parts, " ")
}

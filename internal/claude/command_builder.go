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
	// プロンプトテンプレートを構築
	promptTemplate := promptPath

	// varsがTemplateVariablesの場合は変数を展開
	if templateVars, ok := vars.(*TemplateVariables); ok {
		// {{issue-number}}を置換
		if templateVars.IssueNumber > 0 {
			promptTemplate = strings.Replace(promptTemplate, "{{issue-number}}", fmt.Sprintf("%d", templateVars.IssueNumber), -1)
		}
	}

	// Claudeコマンドを構築（プロンプトはダブルクォートで囲む）
	command := fmt.Sprintf("claude \"%s\"", promptTemplate)

	return command
}

package claude

import (
	"fmt"
	"strings"
)

// TemplateVariables はテンプレート展開で使用する変数
type TemplateVariables struct {
	IssueNumber int
	IssueTitle  string
	RepoName    string
}

// ExpandTemplate はテンプレート文字列内の変数を実際の値に置換する
func ExpandTemplate(template string, vars *TemplateVariables) string {
	result := template

	// {{issue-number}} の置換
	result = strings.ReplaceAll(result, "{{issue-number}}", fmt.Sprintf("%d", vars.IssueNumber))

	// {{issue-title}} の置換
	result = strings.ReplaceAll(result, "{{issue-title}}", vars.IssueTitle)

	// {{repo-name}} の置換
	result = strings.ReplaceAll(result, "{{repo-name}}", vars.RepoName)

	return result
}

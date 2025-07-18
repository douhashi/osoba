package actions

import (
	"github.com/douhashi/osoba/internal/github"
)

// hasLabel はIssueが指定されたラベルを持っているかを確認する
func hasLabel(issue *github.Issue, labelName string) bool {
	if issue == nil || issue.Labels == nil {
		return false
	}

	for _, label := range issue.Labels {
		if label.Name != nil && *label.Name == labelName {
			return true
		}
	}
	return false
}

// getIssueTitle はIssueのタイトルを取得する
func getIssueTitle(issue *github.Issue) string {
	if issue == nil || issue.Title == nil {
		return ""
	}
	return *issue.Title
}

// getRepoName はリポジトリ名を取得する（現在は固定値）
func getRepoName() string {
	// TODO: 実際のリポジトリ名を動的に取得
	return "douhashi/osoba"
}

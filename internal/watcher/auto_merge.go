package watcher

import (
	"context"
	"fmt"

	"github.com/douhashi/osoba/internal/cleanup"
	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
)

// executeAutoMergeIfLGTM はstatus:lgtmラベルが付いたIssueのPRを自動マージする
func executeAutoMergeIfLGTM(
	ctx context.Context,
	issue *github.Issue,
	cfg *config.Config,
	ghClient github.GitHubClient,
	cleanupManager cleanup.Manager,
) error {
	// auto_merge_lgtm設定が無効な場合はスキップ
	if !cfg.GitHub.AutoMergeLGTM {
		return nil
	}

	// status:lgtmラベルがない場合はスキップ
	if !hasLGTMLabel(issue) {
		return nil
	}

	// Issue番号を取得
	if issue.Number == nil {
		return nil
	}
	issueNumber := *issue.Number

	// IssueとリンクされたPRを取得
	pr, err := ghClient.GetPullRequestForIssue(ctx, issueNumber)
	if err != nil {
		return fmt.Errorf("failed to get pull request for issue #%d: %w", issueNumber, err)
	}

	// PRが存在しない場合はスキップ
	if pr == nil {
		return nil
	}

	// PRがマージ可能かチェック
	if !isMergeable(pr) {
		return nil
	}

	// PRをマージ
	if err := ghClient.MergePullRequest(ctx, pr.Number); err != nil {
		return fmt.Errorf("failed to merge pull request #%d: %w", pr.Number, err)
	}

	// マージ成功後、クリーンアップを実行
	// クリーンアップエラーは警告ログのみで処理を継続
	if err := cleanupManager.CleanupIssueResources(ctx, issueNumber); err != nil {
		// エラーはログに記録するが、処理は継続
		// ロガーが利用可能な場合のみログ出力
		return nil
	}

	return nil
}

// executeAutoMergeIfLGTMWithLogger はログ付きの自動マージ処理
func executeAutoMergeIfLGTMWithLogger(
	ctx context.Context,
	issue *github.Issue,
	cfg *config.Config,
	ghClient github.GitHubClient,
	cleanupManager cleanup.Manager,
	log logger.Logger,
) error {
	// auto_merge_lgtm設定が無効な場合はスキップ
	if !cfg.GitHub.AutoMergeLGTM {
		return nil
	}

	// status:lgtmラベルがない場合はスキップ
	if !hasLGTMLabel(issue) {
		return nil
	}

	// Issue番号を取得
	if issue.Number == nil {
		return nil
	}
	issueNumber := *issue.Number

	log.Info("Auto-merge: Processing LGTM issue",
		"issue_number", issueNumber,
	)

	// IssueとリンクされたPRを取得
	pr, err := ghClient.GetPullRequestForIssue(ctx, issueNumber)
	if err != nil {
		return fmt.Errorf("failed to get pull request for issue #%d: %w", issueNumber, err)
	}

	// PRが存在しない場合はスキップ
	if pr == nil {
		log.Debug("Auto-merge: No pull request found for issue",
			"issue_number", issueNumber,
		)
		return nil
	}

	log.Debug("Auto-merge: Found pull request",
		"issue_number", issueNumber,
		"pr_number", pr.Number,
		"state", pr.State,
		"mergeable", pr.Mergeable,
		"is_draft", pr.IsDraft,
	)

	// PRがマージ可能かチェック
	if !isMergeable(pr) {
		log.Info("Auto-merge: Pull request is not mergeable",
			"pr_number", pr.Number,
			"state", pr.State,
			"mergeable", pr.Mergeable,
			"is_draft", pr.IsDraft,
		)
		return nil
	}

	// PRをマージ
	log.Info("Auto-merge: Merging pull request",
		"pr_number", pr.Number,
	)
	if err := ghClient.MergePullRequest(ctx, pr.Number); err != nil {
		return fmt.Errorf("failed to merge pull request #%d: %w", pr.Number, err)
	}

	log.Info("Auto-merge: Successfully merged pull request",
		"pr_number", pr.Number,
	)

	// マージ成功後、クリーンアップを実行
	// クリーンアップエラーは警告ログのみで処理を継続
	log.Info("Auto-merge: Cleaning up resources",
		"issue_number", issueNumber,
	)
	if err := cleanupManager.CleanupIssueResources(ctx, issueNumber); err != nil {
		log.Warn("Auto-merge: Failed to cleanup resources",
			"issue_number", issueNumber,
			"error", err,
		)
		// エラーはログに記録するが、処理は継続
		return nil
	}

	log.Info("Auto-merge: Successfully cleaned up resources",
		"issue_number", issueNumber,
	)

	return nil
}

// hasLGTMLabel はIssueにstatus:lgtmラベルが付いているかチェック
func hasLGTMLabel(issue *github.Issue) bool {
	if issue == nil || issue.Labels == nil {
		return false
	}

	for _, label := range issue.Labels {
		if label != nil && label.Name != nil && *label.Name == "status:lgtm" {
			return true
		}
	}
	return false
}

// isMergeable はPRがマージ可能かチェック
func isMergeable(pr *github.PullRequest) bool {
	if pr == nil {
		return false
	}

	// PRが開いていること
	if pr.State != "OPEN" {
		return false
	}

	// ドラフトでないこと
	if pr.IsDraft {
		return false
	}

	// マージ可能であること
	// MERGEABLE: マージ可能
	// CONFLICTING: コンフリクトあり
	// UNKNOWN: 判定中
	if pr.Mergeable != "MERGEABLE" {
		return false
	}

	return true
}

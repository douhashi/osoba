package watcher

import (
	"context"
	"fmt"
	"time"

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
	log.Debug("Auto-merge: Configuration check",
		"auto_merge_enabled", cfg != nil && cfg.GitHub.AutoMergeLGTM,
	)

	// auto_merge_lgtm設定が無効な場合はスキップ
	if !cfg.GitHub.AutoMergeLGTM {
		log.Debug("Auto-merge: Configuration disabled")
		return nil
	}

	// status:lgtmラベルがない場合はスキップ
	if !hasLGTMLabel(issue) {
		log.Debug("Auto-merge: No LGTM label found")
		return nil
	}

	// Issue番号を取得
	if issue.Number == nil {
		log.Debug("Auto-merge: Issue number is nil")
		return nil
	}
	issueNumber := *issue.Number

	log.Info("Auto-merge: Processing LGTM issue",
		"issue_number", issueNumber,
	)

	// IssueとリンクされたPRを取得（リトライ機能付き）
	pr, err := getPullRequestForIssueWithRetry(ctx, ghClient, issueNumber, log)
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
		"checks_status", pr.ChecksStatus,
	)

	// PRがマージ可能かチェック（リトライ機能付き）
	mergeable, err := checkMergeableWithRetry(ctx, ghClient, pr, log)
	if err != nil {
		return fmt.Errorf("failed to check mergeable status for PR #%d: %w", pr.Number, err)
	}

	if !mergeable {
		log.Info("Auto-merge: Pull request is not mergeable after retry",
			"pr_number", pr.Number,
			"state", pr.State,
			"mergeable", pr.Mergeable,
			"is_draft", pr.IsDraft,
			"checks_status", pr.ChecksStatus,
		)
		return nil
	}

	// PRをマージ
	log.Info("Auto-merge: Merging pull request",
		"pr_number", pr.Number,
	)
	if err := ghClient.MergePullRequest(ctx, pr.Number); err != nil {
		log.Error("Auto-merge: Failed to merge pull request",
			"pr_number", pr.Number,
			"error", err,
		)
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

// getPullRequestForIssueWithRetry はIssueに関連するPRを取得する（リトライ機能付き）
func getPullRequestForIssueWithRetry(
	ctx context.Context,
	ghClient github.GitHubClient,
	issueNumber int,
	log logger.Logger,
) (*github.PullRequest, error) {
	log.Debug("Auto-merge: Getting pull request for issue",
		"issue_number", issueNumber,
	)

	pr, err := ghClient.GetPullRequestForIssue(ctx, issueNumber)
	if err != nil {
		log.Warn("Auto-merge: Failed to get pull request via linked search",
			"issue_number", issueNumber,
			"error", err,
		)

		// フォールバック: ブランチ名による検索を試行
		log.Debug("Auto-merge: Attempting fallback search by branch name",
			"issue_number", issueNumber,
		)

		// 現時点ではフォールバック機能は未実装
		// 将来的にはブランチ名パターン（issue-123, fix-123等）での検索を追加予定
		return nil, err
	}

	if pr != nil {
		log.Debug("Auto-merge: Successfully found pull request via linked search",
			"issue_number", issueNumber,
			"pr_number", pr.Number,
		)
	}

	return pr, nil
}

// checkMergeableWithRetry はPRのマージ可能性をチェックする（リトライ機能付き）
func checkMergeableWithRetry(
	ctx context.Context,
	ghClient github.GitHubClient,
	pr *github.PullRequest,
	log logger.Logger,
) (bool, error) {
	const maxRetries = 3
	const retryDelay = 2 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Debug("Auto-merge: Checking mergeable status",
			"pr_number", pr.Number,
			"attempt", attempt,
			"max_retries", maxRetries,
		)

		// 最新のPRステータスを取得
		currentPR, err := ghClient.GetPullRequestStatus(ctx, pr.Number)
		if err != nil {
			log.Warn("Auto-merge: Failed to get PR status",
				"pr_number", pr.Number,
				"attempt", attempt,
				"error", err,
			)
			if attempt == maxRetries {
				return false, fmt.Errorf("failed to get PR status after %d attempts: %w", maxRetries, err)
			}
			time.Sleep(retryDelay * time.Duration(attempt))
			continue
		}

		// PRステータスを更新
		*pr = *currentPR

		log.Debug("Auto-merge: Current PR status",
			"pr_number", pr.Number,
			"state", pr.State,
			"mergeable", pr.Mergeable,
			"is_draft", pr.IsDraft,
			"checks_status", pr.ChecksStatus,
			"attempt", attempt,
		)

		// UNKNOWN以外の場合は結果を返す
		if pr.Mergeable != "UNKNOWN" {
			mergeable := isMergeable(pr)
			log.Debug("Auto-merge: Mergeable check completed",
				"pr_number", pr.Number,
				"mergeable", mergeable,
				"attempt", attempt,
			)
			return mergeable, nil
		}

		// UNKNOWNの場合は再試行
		log.Info("Auto-merge: PR mergeable status is UNKNOWN, retrying",
			"pr_number", pr.Number,
			"attempt", attempt,
			"max_retries", maxRetries,
		)

		if attempt < maxRetries {
			time.Sleep(retryDelay * time.Duration(attempt))
		}
	}

	// 最大試行回数に達してもUNKNOWNの場合
	log.Warn("Auto-merge: PR mergeable status remains UNKNOWN after max retries",
		"pr_number", pr.Number,
		"max_retries", maxRetries,
	)
	return false, nil // エラーではなく、マージ不可として扱う
}

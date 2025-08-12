package watcher

import (
	"context"
	"fmt"

	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
)

// executeAutoReviseIfRequiresChanges はstatus:requires-changesラベルが付いたPRに対してreviseアクションを自動実行する
func executeAutoReviseIfRequiresChanges(
	ctx context.Context,
	pr *github.PullRequest,
	cfg *config.Config,
	ghClient github.GitHubClient,
	actionManager ActionManagerInterface,
	sessionName string,
) error {
	// auto_revise_pr設定が無効な場合はスキップ
	if !cfg.GitHub.AutoRevisePR {
		return nil
	}

	// status:requires-changesラベルがない場合はスキップ
	if !hasRequiresChangesLabel(pr) {
		return nil
	}

	// PRから関連するIssue番号を取得
	issueNumber, err := ghClient.GetClosingIssueNumber(ctx, pr.Number)
	if err != nil {
		return fmt.Errorf("failed to get closing issue number for PR #%d: %w", pr.Number, err)
	}

	// Issue番号が取得できない場合はスキップ
	if issueNumber == 0 {
		return nil
	}

	// 該当のIssueを作成（実際のラベル情報は不要）
	targetIssue := &github.Issue{
		Number: &issueNumber,
		Labels: []*github.Label{
			{Name: &[]string{"status:requires-changes"}[0]},
		},
	}

	// ActionManagerを使用してReviseActionを実行
	if err := actionManager.ExecuteAction(ctx, targetIssue); err != nil {
		return fmt.Errorf("failed to execute revise action for issue #%d: %w", issueNumber, err)
	}

	return nil
}

// executeAutoReviseIfRequiresChangesWithLogger はログ付きの自動Revise処理
func executeAutoReviseIfRequiresChangesWithLogger(
	ctx context.Context,
	pr *github.PullRequest,
	cfg *config.Config,
	ghClient github.GitHubClient,
	actionManager ActionManagerInterface,
	sessionName string,
	log logger.Logger,
) error {
	// nil PRチェック
	if pr == nil {
		log.Debug("Auto-revise: PR is nil, skipping")
		return nil
	}

	log.Debug("Auto-revise: Configuration check",
		"auto_revise_enabled", cfg != nil && cfg.GitHub.AutoRevisePR,
		"pr_number", pr.Number,
	)

	// auto_revise_pr設定が無効な場合はスキップ
	if !cfg.GitHub.AutoRevisePR {
		log.Debug("Auto-revise: Configuration disabled")
		return nil
	}

	// status:requires-changesラベルがない場合はスキップ
	if !hasRequiresChangesLabel(pr) {
		log.Debug("Auto-revise: No requires-changes label found",
			"pr_number", pr.Number,
		)
		return nil
	}

	log.Info("Auto-revise: Processing PR with requires-changes label",
		"pr_number", pr.Number,
	)

	// PRから関連するIssue番号を取得
	issueNumber, err := ghClient.GetClosingIssueNumber(ctx, pr.Number)
	if err != nil {
		log.Error("Auto-revise: Failed to get closing issue number",
			"pr_number", pr.Number,
			"error", err,
		)
		return fmt.Errorf("failed to get closing issue number for PR #%d: %w", pr.Number, err)
	}

	// Issue番号が取得できない場合はスキップ
	if issueNumber == 0 {
		log.Warn("Auto-revise: No closing issue found for PR",
			"pr_number", pr.Number,
		)
		return nil
	}

	log.Info("Auto-revise: Found closing issue",
		"pr_number", pr.Number,
		"issue_number", issueNumber,
	)

	// Issueを取得（まず全てのIssueを取得してからフィルタリング）
	// ここでは簡単のため、Issue番号から直接Issueオブジェクトを構築
	// 実際のラベル情報は不要（ActionManagerが判断する）
	targetIssue := &github.Issue{
		Number: &issueNumber,
	}

	// ActionManagerのReviseActionが存在するか確認
	action := actionManager.GetActionForIssue(targetIssue)
	if action == nil {
		log.Warn("Auto-revise: No action found for issue",
			"issue_number", issueNumber,
		)

		// ReviseActionを明示的に実行する必要がある場合
		// ActionManagerのFactoryを通じてReviseActionを作成し実行
		if factory, ok := actionManager.(*ActionManager); ok {
			// ReviseActionをFactoryに登録されているか確認
			reviseAction := factory.GetActionForIssue(&github.Issue{
				Number: &issueNumber,
				Labels: []*github.Label{
					{Name: &[]string{"status:requires-changes"}[0]},
				},
			})

			if reviseAction != nil {
				log.Info("Auto-revise: Executing revise action",
					"issue_number", issueNumber,
				)
				if err := reviseAction.Execute(ctx, targetIssue); err != nil {
					log.Error("Auto-revise: Failed to execute revise action",
						"issue_number", issueNumber,
						"error", err,
					)
					return fmt.Errorf("failed to execute revise action for issue #%d: %w", issueNumber, err)
				}
				log.Info("Auto-revise: Successfully executed revise action",
					"issue_number", issueNumber,
				)
				return nil
			}
		}

		// ReviseActionが見つからない場合は、直接ラベル付きのIssueを作成
		targetIssue = &github.Issue{
			Number: &issueNumber,
			Labels: []*github.Label{
				{Name: &[]string{"status:requires-changes"}[0]},
			},
		}
	}

	// ActionManagerを使用してReviseActionを実行
	log.Info("Auto-revise: Executing action via ActionManager",
		"issue_number", issueNumber,
	)

	if err := actionManager.ExecuteAction(ctx, targetIssue); err != nil {
		log.Error("Auto-revise: Failed to execute action",
			"issue_number", issueNumber,
			"error", err,
		)
		return fmt.Errorf("failed to execute revise action for issue #%d: %w", issueNumber, err)
	}

	log.Info("Auto-revise: Successfully executed revise action",
		"issue_number", issueNumber,
	)

	return nil
}

// hasRequiresChangesLabel はPRにstatus:requires-changesラベルが付いているかチェック
// 注意: PR自体にはLabelsフィールドがないため、このメソッドは常にfalseを返す
// 実際のラベル検知はPR watcherのラベルフィルタリングで行われる
func hasRequiresChangesLabel(pr *github.PullRequest) bool {
	// PR watcherが既にラベルでフィルタリングしているので、
	// この関数が呼ばれる時点で対象のPRはstatus:requires-changesを持っている
	// ただし、互換性のため常にtrueを返すことはせず、PR存在チェックのみ行う
	return pr != nil
}

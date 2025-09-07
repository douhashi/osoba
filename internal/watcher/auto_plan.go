package watcher

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
)

// executeAutoPlanIfNoActiveIssues は処理中のIssueがない場合に自動的に次のIssueをplanフェーズに移行させる
func executeAutoPlanIfNoActiveIssues(
	ctx context.Context,
	cfg *config.Config,
	ghClient GitHubClientInterface,
	owner, repo string,
	log logger.Logger,
) error {
	// auto_plan_issue設定が無効な場合はスキップ
	if !cfg.GitHub.AutoPlanIssue {
		return nil
	}

	// status:*ラベル付きIssueが存在するかチェック
	statusLabels := []string{
		"status:needs-plan",
		"status:planning",
		"status:ready",
		"status:implementing",
		"status:review-requested",
		"status:reviewing",
		"status:lgtm",
		"status:requires-changes",
		"status:revising",
	}

	activeIssues, err := ghClient.ListIssuesByLabels(ctx, owner, repo, statusLabels)
	if err != nil {
		return &AutoPlanError{
			Type:    "api_error",
			Message: "failed to list active issues",
			Cause:   err,
		}
	}

	// 処理中のIssueが存在する場合はスキップ
	if len(activeIssues) > 0 {
		log.Debug("Auto-plan: Skipping because active issues exist",
			"active_count", len(activeIssues),
		)
		return nil
	}

	log.Debug("Auto-plan: No active issues found, searching for unlabeled issues")

	// すべてのオープンIssueを取得
	allIssues, err := ghClient.ListAllOpenIssues(ctx, owner, repo)
	if err != nil {
		return &AutoPlanError{
			Type:    "api_error",
			Message: "failed to list all open issues",
			Cause:   err,
		}
	}

	// status:*ラベルが付いていない最も若い番号のIssueを特定
	targetIssue := findLowestNumberIssueWithoutStatusLabel(allIssues)
	if targetIssue == nil {
		log.Debug("Auto-plan: No unlabeled issues found")
		return nil
	}

	issueNumber := *targetIssue.Number
	log.Info("Auto-plan: Adding status:needs-plan label to issue",
		"issue_number", issueNumber,
		"issue_title", safeStringValue(targetIssue.Title),
	)

	// status:needs-planラベルを付与
	if err := ghClient.AddLabel(ctx, owner, repo, issueNumber, "status:needs-plan"); err != nil {
		return &AutoPlanError{
			Type:        "label_error",
			Message:     "failed to add status:needs-plan label",
			Cause:       err,
			IssueNumber: &issueNumber,
		}
	}

	log.Info("Auto-plan: Successfully added status:needs-plan label",
		"issue_number", issueNumber,
	)

	return nil
}

// findLowestNumberIssueWithoutStatusLabel はstatus:*ラベルが付いていない最も若い番号のIssueを返す
func findLowestNumberIssueWithoutStatusLabel(issues []*github.Issue) *github.Issue {
	var lowestIssue *github.Issue
	lowestNumber := int(^uint(0) >> 1) // int型の最大値

	for _, issue := range issues {
		if issue.Number == nil {
			continue
		}

		if hasStatusLabel(issue) {
			continue
		}

		if *issue.Number < lowestNumber {
			lowestNumber = *issue.Number
			lowestIssue = issue
		}
	}

	return lowestIssue
}

// hasStatusLabel はIssueにstatus:*ラベルが付いているかチェック
func hasStatusLabel(issue *github.Issue) bool {
	if issue == nil || issue.Labels == nil {
		return false
	}

	for _, label := range issue.Labels {
		if label == nil || label.Name == nil {
			continue
		}
		if strings.HasPrefix(*label.Name, "status:") {
			return true
		}
	}
	return false
}

// safeStringValue はstring pointerを安全に文字列に変換する
func safeStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// AutoPlanError は自動計画機能のエラーを表す
type AutoPlanError struct {
	Type        string
	Message     string
	Cause       error
	IssueNumber *int
}

func (e *AutoPlanError) Error() string {
	if e.IssueNumber != nil {
		return e.Message + " (issue #" + string(rune(*e.IssueNumber)) + "): " + e.Cause.Error()
	}
	return e.Message + ": " + e.Cause.Error()
}

func (e *AutoPlanError) Unwrap() error {
	return e.Cause
}

// RaceConditionError は競合状態検出時のエラーを表す
type RaceConditionError struct {
	Type        string
	Message     string
	IssueNumber *int
	Timestamp   time.Time
}

func (e *RaceConditionError) Error() string {
	if e.IssueNumber != nil {
		return fmt.Sprintf("%s (issue #%d): %s", e.Message, *e.IssueNumber, e.Type)
	}
	return fmt.Sprintf("%s: %s", e.Message, e.Type)
}

// executeAutoPlanWithOptimisticLock は楽観的ロック機能付きのauto_plan実行
func executeAutoPlanWithOptimisticLock(
	ctx context.Context,
	cfg *config.Config,
	ghClient GitHubClientInterface,
	owner, repo string,
	log logger.Logger,
) error {
	// auto_plan_issue設定が無効な場合はスキップ
	if !cfg.GitHub.AutoPlanIssue {
		return nil
	}

	statusLabels := []string{
		"status:needs-plan",
		"status:planning",
		"status:ready",
		"status:implementing",
		"status:review-requested",
		"status:reviewing",
		"status:lgtm",
		"status:requires-changes",
		"status:revising",
	}

	// 最初のチェック: アクティブIssueの存在確認
	activeIssues, err := ghClient.ListIssuesByLabels(ctx, owner, repo, statusLabels)
	if err != nil {
		return &AutoPlanError{
			Type:    "api_error",
			Message: "failed to list active issues in optimistic lock",
			Cause:   err,
		}
	}

	if len(activeIssues) > 0 {
		log.Debug("Auto-plan: Skipping because active issues exist (optimistic lock)",
			"active_count", len(activeIssues),
		)
		return nil
	}

	// すべてのオープンIssueを取得
	allIssues, err := ghClient.ListAllOpenIssues(ctx, owner, repo)
	if err != nil {
		return &AutoPlanError{
			Type:    "api_error",
			Message: "failed to list all open issues",
			Cause:   err,
		}
	}

	// status:*ラベルが付いていない最も若い番号のIssueを特定
	targetIssue := findLowestNumberIssueWithoutStatusLabel(allIssues)
	if targetIssue == nil {
		log.Debug("Auto-plan: No unlabeled issues found")
		return nil
	}

	issueNumber := *targetIssue.Number

	// 楽観的ロック: ラベル付与前の再確認
	log.Debug("Auto-plan: Performing optimistic lock check before label assignment",
		"issue_number", issueNumber,
	)

	reconfirmActiveIssues, err := ghClient.ListIssuesByLabels(ctx, owner, repo, statusLabels)
	if err != nil {
		return &AutoPlanError{
			Type:    "api_error",
			Message: "failed to reconfirm active issues during optimistic lock",
			Cause:   err,
		}
	}

	// 他のプロセスが先にラベルを付与していた場合は競合検出
	if len(reconfirmActiveIssues) > 0 {
		log.Info("Auto-plan: Race condition detected - another process added labels",
			"active_count", len(reconfirmActiveIssues),
			"target_issue", issueNumber,
		)
		// 競合は正常な動作なので、エラーではなくnilを返す
		return nil
	}

	log.Info("Auto-plan: Adding status:needs-plan label to issue (optimistic lock)",
		"issue_number", issueNumber,
		"issue_title", safeStringValue(targetIssue.Title),
	)

	// ラベル付与
	if err := ghClient.AddLabel(ctx, owner, repo, issueNumber, "status:needs-plan"); err != nil {
		return &AutoPlanError{
			Type:        "label_error",
			Message:     "failed to add status:needs-plan label (optimistic lock)",
			Cause:       err,
			IssueNumber: &issueNumber,
		}
	}

	log.Info("Auto-plan: Successfully added status:needs-plan label (optimistic lock)",
		"issue_number", issueNumber,
	)

	return nil
}

// executeAutoPlanWithOptimisticLockWithRetry はリトライ機能付きの楽観的ロック実行
func executeAutoPlanWithOptimisticLockWithRetry(
	ctx context.Context,
	cfg *config.Config,
	ghClient GitHubClientInterface,
	owner, repo string,
	log logger.Logger,
) error {
	const maxRetries = 3
	const baseDelay = time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := executeAutoPlanWithOptimisticLock(ctx, cfg, ghClient, owner, repo, log)
		if err == nil {
			return nil
		}

		// 最後の試行でエラーが発生した場合はそのまま返す
		if attempt == maxRetries {
			return err
		}

		// リトライ対象のエラーかどうかを判定
		if shouldRetryAutoPlan(err) {
			delay := time.Duration(attempt) * baseDelay
			log.Warn("Auto-plan: Retrying after error",
				"attempt", attempt,
				"maxRetries", maxRetries,
				"delay", delay,
				"error", err,
			)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// リトライ
			}
		} else {
			// リトライしないエラーはそのまま返す
			return err
		}
	}

	return nil
}

// shouldRetryAutoPlan はリトライすべきエラーかどうかを判定
func shouldRetryAutoPlan(err error) bool {
	if err == nil {
		return false
	}

	// AutoPlanErrorの場合はタイプに応じて判定
	if autoPlanErr, ok := err.(*AutoPlanError); ok {
		return autoPlanErr.Type == "api_error"
	}

	// RaceConditionErrorの場合は通常リトライしない
	if _, ok := err.(*RaceConditionError); ok {
		return false
	}

	// その他のエラーは一般的にリトライする
	return true
}

// GitHubClientInterface はauto_plan機能で使用するGitHubクライアントインターフェース
type GitHubClientInterface interface {
	ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error)
	ListAllOpenIssues(ctx context.Context, owner, repo string) ([]*github.Issue, error)
	AddLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error
}

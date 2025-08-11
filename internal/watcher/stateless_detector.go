package watcher

import (
	"fmt"

	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
)

// トリガーラベルの定数定義
const (
	TriggerLabelNeedsPlan       = "status:needs-plan"
	TriggerLabelReady           = "status:ready"
	TriggerLabelReviewRequested = "status:review-requested"
	TriggerLabelRequiresChanges = "status:requires-changes"
)

// 実行中ラベルの定数定義
const (
	ExecutionLabelPlanning     = "status:planning"
	ExecutionLabelImplementing = "status:implementing"
	ExecutionLabelReviewing    = "status:reviewing"
)

// getTriggerLabelPriority はトリガーラベルの優先順位順に返す
// 優先順位: needs-plan > ready > review-requested > requires-changes
func getTriggerLabelPriority() []string {
	return []string{
		TriggerLabelNeedsPlan,
		TriggerLabelReady,
		TriggerLabelReviewRequested,
		TriggerLabelRequiresChanges,
	}
}

// GetTriggerLabelMapping はトリガーラベルと実行中ラベルの対応関係を返す
func GetTriggerLabelMapping() map[string]string {
	return map[string]string{
		TriggerLabelNeedsPlan:       ExecutionLabelPlanning,
		TriggerLabelReady:           ExecutionLabelImplementing,
		TriggerLabelReviewRequested: ExecutionLabelReviewing,
		TriggerLabelRequiresChanges: "", // requires-changesには対応する実行中ラベルがない（直接readyに遷移）
	}
}

// ShouldProcessIssue はIssueを処理すべきかをラベルベースで判定する
// GitHubのラベル状態を唯一の情報源として、ステートレスに判定を行う
func ShouldProcessIssue(issue *github.Issue) (bool, string) {
	// nilチェックを強化
	if issue == nil {
		return false, "No trigger labels found"
	}

	if issue.Labels == nil {
		return false, "No trigger labels found"
	}

	triggerMapping := GetTriggerLabelMapping()
	issueLabels := make(map[string]bool, len(issue.Labels))

	// Issueのラベルをマップに変換（パフォーマンス最適化）
	for _, label := range issue.Labels {
		if label != nil && label.Name != nil {
			issueLabels[*label.Name] = true
		}
	}

	// トリガーラベルを優先順位順に判定
	triggerPriority := getTriggerLabelPriority()
	for _, trigger := range triggerPriority {
		executionLabel := triggerMapping[trigger]
		hasTrigger := issueLabels[trigger]
		hasExecution := issueLabels[executionLabel]

		if hasTrigger {
			if hasExecution {
				// トリガーラベルはあるが、対応する実行中ラベルもある場合は処理しない
				reason := fmt.Sprintf("Execution label '%s' already exists for trigger '%s'", executionLabel, trigger)
				return false, reason
			}
			// トリガーラベルがあり、対応する実行中ラベルがない場合は処理する
			reason := fmt.Sprintf("Trigger label '%s' found without corresponding execution label", trigger)
			return true, reason
		}
	}

	// トリガーラベルがない場合は処理しない
	return false, "No trigger labels found"
}

// ShouldProcessIssueWithLogger はログ出力機能付きでIssueを処理すべきかを判定する
// GitHubのラベル状態を唯一の情報源として、ステートレスに判定を行う
func ShouldProcessIssueWithLogger(issue *github.Issue, log logger.Logger) (bool, string) {
	// nilチェックを強化
	if issue == nil {
		log.Debug("ShouldProcessIssue: issue is nil")
		return false, "No trigger labels found"
	}

	// Issue番号のログ出力（デバッグ用）
	issueNumber := 0
	if issue.Number != nil {
		issueNumber = *issue.Number
	}

	if issue.Labels == nil {
		log.Debug("ShouldProcessIssue: labels are nil", "issue", issueNumber)
		return false, "No trigger labels found"
	}

	triggerMapping := GetTriggerLabelMapping()
	issueLabels := make(map[string]bool, len(issue.Labels))

	// Issueのラベルをマップに変換（パフォーマンス最適化）
	labelCount := 0
	for _, label := range issue.Labels {
		if label != nil && label.Name != nil {
			issueLabels[*label.Name] = true
			labelCount++
		}
	}

	log.Debug("ShouldProcessIssue: processing issue labels",
		"issue", issueNumber,
		"totalLabels", len(issue.Labels),
		"validLabels", labelCount)

	// トリガーラベルを優先順位順に判定
	triggerPriority := getTriggerLabelPriority()
	for _, trigger := range triggerPriority {
		executionLabel := triggerMapping[trigger]
		hasTrigger := issueLabels[trigger]
		hasExecution := issueLabels[executionLabel]

		if hasTrigger {
			if hasExecution {
				// トリガーラベルはあるが、対応する実行中ラベルもある場合は処理しない
				reason := fmt.Sprintf("Execution label '%s' already exists for trigger '%s'", executionLabel, trigger)
				log.Debug("ShouldProcessIssue: skip processing",
					"issue", issueNumber,
					"trigger", trigger,
					"executionLabel", executionLabel,
					"reason", reason)
				return false, reason
			}
			// トリガーラベルがあり、対応する実行中ラベルがない場合は処理する
			reason := fmt.Sprintf("Trigger label '%s' found without corresponding execution label", trigger)
			log.Debug("ShouldProcessIssue: should process",
				"issue", issueNumber,
				"trigger", trigger,
				"reason", reason)
			return true, reason
		}
	}

	// トリガーラベルがない場合は処理しない
	log.Debug("ShouldProcessIssue: no trigger labels found", "issue", issueNumber)
	return false, "No trigger labels found"
}

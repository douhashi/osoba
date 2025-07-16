package watcher

import (
	"fmt"

	"github.com/douhashi/osoba/internal/github"
)

// トリガーラベルの定数定義
const (
	TriggerLabelNeedsPlan       = "status:needs-plan"
	TriggerLabelReady           = "status:ready"
	TriggerLabelReviewRequested = "status:review-requested"
)

// 実行中ラベルの定数定義
const (
	ExecutionLabelPlanning     = "status:planning"
	ExecutionLabelImplementing = "status:implementing"
	ExecutionLabelReviewing    = "status:reviewing"
)

// getTriggerLabelPriority はトリガーラベルの優先順位順に返す
// 優先順位: needs-plan > ready > review-requested
func getTriggerLabelPriority() []string {
	return []string{
		TriggerLabelNeedsPlan,
		TriggerLabelReady,
		TriggerLabelReviewRequested,
	}
}

// GetTriggerLabelMapping はトリガーラベルと実行中ラベルの対応関係を返す
func GetTriggerLabelMapping() map[string]string {
	return map[string]string{
		TriggerLabelNeedsPlan:       ExecutionLabelPlanning,
		TriggerLabelReady:           ExecutionLabelImplementing,
		TriggerLabelReviewRequested: ExecutionLabelReviewing,
	}
}

// ShouldProcessIssue はIssueを処理すべきかをラベルベースで判定する
// GitHubのラベル状態を唯一の情報源として、ステートレスに判定を行う
func ShouldProcessIssue(issue *github.Issue) (bool, string) {
	if issue == nil || issue.Labels == nil {
		return false, "No trigger labels found"
	}

	triggerMapping := GetTriggerLabelMapping()
	issueLabels := make(map[string]bool)

	// Issueのラベルをマップに変換
	for _, label := range issue.Labels {
		if label.Name != nil {
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
				return false, fmt.Sprintf("Execution label '%s' already exists for trigger '%s'", executionLabel, trigger)
			}
			// トリガーラベルがあり、対応する実行中ラベルがない場合は処理する
			return true, fmt.Sprintf("Trigger label '%s' found without corresponding execution label", trigger)
		}
	}

	// トリガーラベルがない場合は処理しない
	return false, "No trigger labels found"
}

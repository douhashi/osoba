package watcher

import (
	"fmt"
	"time"
)

// EventType はIssueイベントの種類を表す
type EventType string

const (
	// IssueDetected 新しいIssueが検出された
	IssueDetected EventType = "issue_detected"
	// LabelAdded ラベルが追加された
	LabelAdded EventType = "label_added"
	// LabelRemoved ラベルが削除された
	LabelRemoved EventType = "label_removed"
	// LabelChanged ラベルが変更された（同じプレフィックスのラベル間の変更）
	LabelChanged EventType = "label_changed"
)

// IssueEvent はIssueのラベル変更イベントを表す
type IssueEvent struct {
	Type       EventType
	IssueID    int
	IssueTitle string
	Owner      string
	Repo       string
	FromLabel  string // LabelRemoved, LabelChangedで使用
	ToLabel    string // LabelAdded, LabelChangedで使用
	Timestamp  time.Time
}

// String はイベントの文字列表現を返す
func (e IssueEvent) String() string {
	switch e.Type {
	case IssueDetected:
		return fmt.Sprintf("[%s] Issue #%d '%s' (%s/%s) detected at %s",
			e.Type, e.IssueID, e.IssueTitle, e.Owner, e.Repo, e.Timestamp.Format(time.RFC3339))
	case LabelAdded:
		return fmt.Sprintf("[%s] Issue #%d '%s' (%s/%s): Label added '%s' at %s",
			e.Type, e.IssueID, e.IssueTitle, e.Owner, e.Repo, e.ToLabel, e.Timestamp.Format(time.RFC3339))
	case LabelRemoved:
		return fmt.Sprintf("[%s] Issue #%d '%s' (%s/%s): Label removed '%s' at %s",
			e.Type, e.IssueID, e.IssueTitle, e.Owner, e.Repo, e.FromLabel, e.Timestamp.Format(time.RFC3339))
	case LabelChanged:
		return fmt.Sprintf("[%s] Issue #%d '%s' (%s/%s): Label changed from '%s' to '%s' at %s",
			e.Type, e.IssueID, e.IssueTitle, e.Owner, e.Repo, e.FromLabel, e.ToLabel, e.Timestamp.Format(time.RFC3339))
	default:
		return fmt.Sprintf("[%s] Issue #%d '%s' (%s/%s) at %s",
			e.Type, e.IssueID, e.IssueTitle, e.Owner, e.Repo, e.Timestamp.Format(time.RFC3339))
	}
}

// DetectLabelChanges は新旧のラベルリストを比較してイベントを生成する
func DetectLabelChanges(oldLabels, newLabels []string) []IssueEvent {
	events := []IssueEvent{}

	// ラベルをセットに変換
	oldSet := make(map[string]bool)
	for _, label := range oldLabels {
		oldSet[label] = true
	}

	newSet := make(map[string]bool)
	for _, label := range newLabels {
		newSet[label] = true
	}

	// statusプレフィックスのラベルを検出
	var oldStatus, newStatus string
	for label := range oldSet {
		if hasPrefix(label, "status:") {
			oldStatus = label
			break
		}
	}
	for label := range newSet {
		if hasPrefix(label, "status:") {
			newStatus = label
			break
		}
	}

	// statusラベルの変更を検出
	if oldStatus != "" && newStatus != "" && oldStatus != newStatus {
		events = append(events, IssueEvent{
			Type:      LabelChanged,
			FromLabel: oldStatus,
			ToLabel:   newStatus,
		})
		// 変更イベントを生成した場合、個別の追加/削除イベントは生成しない
		delete(oldSet, oldStatus)
		delete(newSet, newStatus)
	}

	// 削除されたラベル
	for label := range oldSet {
		if !newSet[label] {
			events = append(events, IssueEvent{
				Type:      LabelRemoved,
				FromLabel: label,
			})
		}
	}

	// 追加されたラベル
	for label := range newSet {
		if !oldSet[label] {
			events = append(events, IssueEvent{
				Type:    LabelAdded,
				ToLabel: label,
			})
		}
	}

	return events
}

// hasPrefix は文字列が指定されたプレフィックスで始まるかを確認する
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

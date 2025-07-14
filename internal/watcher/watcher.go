package watcher

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/douhashi/osoba/internal/github"
	gh "github.com/google/go-github/v50/github"
)

// IssueCallback はIssue検出時に呼ばれるコールバック関数
type IssueCallback func(issue *gh.Issue)

// IssueWatcher はGitHub Issueを監視する構造体
type IssueWatcher struct {
	client              github.GitHubClient
	owner               string
	repo                string
	labels              []string
	pollInterval        time.Duration
	seenIssues          map[int64]bool // 既に処理したIssueを記録
	actionManager       *ActionManager
	eventNotifier       *EventNotifier     // イベント通知システム
	labelChangeTracking bool               // ラベル変更追跡が有効かどうか
	issueLabels         map[int64][]string // Issue IDとラベルのマッピング
}

// NewIssueWatcher は新しいIssueWatcherを作成する
func NewIssueWatcher(client github.GitHubClient, owner, repo, sessionName string, labels []string) (*IssueWatcher, error) {
	if owner == "" {
		return nil, errors.New("owner is required")
	}
	if repo == "" {
		return nil, errors.New("repo is required")
	}
	if sessionName == "" {
		return nil, errors.New("session name is required")
	}
	if len(labels) == 0 {
		return nil, errors.New("at least one label is required")
	}

	return &IssueWatcher{
		client:              client,
		owner:               owner,
		repo:                repo,
		labels:              labels,
		pollInterval:        5 * time.Second, // デフォルト5秒
		seenIssues:          make(map[int64]bool),
		actionManager:       NewActionManager(sessionName),
		labelChangeTracking: false,
		issueLabels:         make(map[int64][]string),
	}, nil
}

// SetPollInterval はポーリング間隔を設定する
func (w *IssueWatcher) SetPollInterval(interval time.Duration) error {
	if interval < time.Second {
		return errors.New("poll interval must be at least 1 second")
	}
	w.pollInterval = interval
	return nil
}

// GetPollInterval は現在のポーリング間隔を取得する
func (w *IssueWatcher) GetPollInterval() time.Duration {
	return w.pollInterval
}

// Start はIssue監視を開始する
func (w *IssueWatcher) Start(ctx context.Context, callback IssueCallback) {
	log.Printf("Starting issue watcher: owner=%s, repo=%s, labels=%v, interval=%v",
		w.owner, w.repo, w.labels, w.pollInterval)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// 初回実行
	w.checkIssues(ctx, callback)

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping issue watcher")
			return
		case <-ticker.C:
			w.checkIssues(ctx, callback)
		}
	}
}

// StartWithActions はIssue監視を開始し、ラベルに基づいてアクションを実行する
func (w *IssueWatcher) StartWithActions(ctx context.Context) {
	callback := func(issue *gh.Issue) {
		// ActionManagerを使用してアクションを実行
		if err := w.actionManager.ExecuteAction(ctx, issue); err != nil {
			log.Printf("Failed to execute action for issue #%d: %v", issue.GetNumber(), err)
		}
	}

	w.Start(ctx, callback)
}

// checkIssues は現在のIssueをチェックし、新しいIssueがあればコールバックを呼ぶ
func (w *IssueWatcher) checkIssues(ctx context.Context, callback IssueCallback) {
	var issues []*gh.Issue

	// リトライ付きでAPIを呼び出し
	err := RetryWithBackoff(ctx, 3, time.Second, func() error {
		var err error
		issues, err = w.client.ListIssuesByLabels(ctx, w.owner, w.repo, w.labels)
		return err
	})

	if err != nil {
		log.Printf("Failed to list issues: %v", err)
		return
	}

	for _, issue := range issues {
		if issue.Number == nil {
			continue
		}

		issueID := int64(*issue.Number)
		currentLabels := getLabels(issue)

		// 新しいIssueの検出
		if !w.seenIssues[issueID] {
			w.seenIssues[issueID] = true
			log.Printf("New issue detected: #%d - %s (labels: %v)",
				*issue.Number,
				safeString(issue.Title),
				currentLabels)

			// イベント通知
			if w.eventNotifier != nil {
				event := IssueEvent{
					Type:       IssueDetected,
					IssueID:    int(*issue.Number),
					IssueTitle: safeString(issue.Title),
					Owner:      w.owner,
					Repo:       w.repo,
					Timestamp:  time.Now(),
				}
				w.eventNotifier.Send(event)
			}

			callback(issue)
		}

		// ラベル変更の追跡
		if w.labelChangeTracking {
			previousLabels, exists := w.issueLabels[issueID]
			if exists {
				// ラベル変更をチェック
				events := DetectLabelChanges(previousLabels, currentLabels)
				for _, event := range events {
					event.IssueID = int(*issue.Number)
					event.IssueTitle = safeString(issue.Title)
					event.Owner = w.owner
					event.Repo = w.repo
					event.Timestamp = time.Now()

					log.Printf("Label change detected for issue #%d: %s %s -> %s",
						*issue.Number, event.Type, event.FromLabel, event.ToLabel)

					// イベント通知
					if w.eventNotifier != nil {
						w.eventNotifier.Send(event)
					}
				}
			}

			// 現在のラベルを保存
			w.issueLabels[issueID] = currentLabels
		}
	}
}

// GetRateLimit はGitHub APIのレート制限情報を取得する
func (w *IssueWatcher) GetRateLimit(ctx context.Context) (*gh.RateLimits, error) {
	return w.client.GetRateLimit(ctx)
}

// ヘルパー関数
func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func getLabels(issue *gh.Issue) []string {
	var labels []string
	for _, label := range issue.Labels {
		if label.Name != nil {
			labels = append(labels, *label.Name)
		}
	}
	return labels
}

// SetEventNotifier はイベント通知システムを設定する
func (w *IssueWatcher) SetEventNotifier(notifier *EventNotifier) {
	w.eventNotifier = notifier
}

// EnableLabelChangeTracking はラベル変更追跡を有効/無効にする
func (w *IssueWatcher) EnableLabelChangeTracking(enable bool) {
	w.labelChangeTracking = enable
}

// NewIssueWatcherWithLabelTracking はラベル変更追跡機能付きのIssueWatcherを作成する
func NewIssueWatcherWithLabelTracking(client github.GitHubClient, owner, repo, sessionName string, labels []string) (*IssueWatcher, error) {
	watcher, err := NewIssueWatcher(client, owner, repo, sessionName, labels)
	if err != nil {
		return nil, err
	}
	watcher.labelChangeTracking = true
	return watcher, nil
}

package watcher

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"time"

	"github.com/douhashi/osoba/internal/github"
	gh "github.com/google/go-github/v67/github"
)

// IssueCallback はIssue検出時に呼ばれるコールバック関数
type IssueCallback func(issue *gh.Issue)

// HealthStats はwatcherの健全性統計情報
type HealthStats struct {
	TotalExecutions      int
	SuccessfulExecutions int
	FailedExecutions     int
	LastExecutionTime    time.Time
	StartTime            time.Time
}

// HealthStatus はヘルスチェックの結果
type HealthStatus struct {
	IsHealthy bool
	Message   string
}

// IssueWatcher はGitHub Issueを監視する構造体
type IssueWatcher struct {
	client              github.GitHubClient
	owner               string
	repo                string
	labels              []string
	pollInterval        time.Duration
	actionManager       *ActionManager
	eventNotifier       *EventNotifier     // イベント通知システム
	labelChangeTracking bool               // ラベル変更追跡が有効かどうか
	issueLabels         map[int64][]string // Issue IDとラベルのマッピング

	// ヘルスチェック用のフィールド
	lastExecutionTime    time.Time
	totalExecutions      int
	successfulExecutions int
	failedExecutions     int
	startTime            time.Time
	mu                   sync.Mutex // ヘルスチェックフィールドの保護用
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
		actionManager:       NewActionManager(sessionName),
		labelChangeTracking: false,
		issueLabels:         make(map[int64][]string),
		startTime:           time.Now(),
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

// GetActionManager はActionManagerを取得する
func (w *IssueWatcher) GetActionManager() *ActionManager {
	return w.actionManager
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
	// サイクル開始時刻
	startTime := time.Now()
	log.Printf("Starting issue check cycle at %s", startTime.Format(time.RFC3339))

	// 統計情報の更新
	w.mu.Lock()
	w.totalExecutions++
	w.mu.Unlock()

	// 処理統計の記録
	var processedCount, processedIssueCount int
	var executionSuccessful bool
	defer func() {
		elapsed := time.Since(startTime)
		log.Printf("Completed issue check cycle: checked issues: %d, processed issues: %d, time taken: %v",
			processedCount, processedIssueCount, elapsed)

		// ヘルスチェック情報の更新
		w.mu.Lock()
		if executionSuccessful {
			w.successfulExecutions++
		} else {
			w.failedExecutions++
		}
		w.lastExecutionTime = time.Now()
		w.mu.Unlock()
	}()

	// パニックリカバリー
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in checkIssues: %v\nStack trace:\n%s", r, string(debug.Stack()))
		}
	}()

	var issues []*gh.Issue

	// リトライ付きでAPIを呼び出し
	err := RetryWithBackoff(ctx, 3, time.Second, func() error {
		var err error
		issues, err = w.client.ListIssuesByLabels(ctx, w.owner, w.repo, w.labels)
		return err
	})

	if err != nil {
		log.Printf("Failed to list issues: %v (owner=%s, repo=%s, labels=%v)",
			err, w.owner, w.repo, w.labels)
		return
	}

	// API呼び出しが成功
	executionSuccessful = true

	for _, issue := range issues {
		if issue.Number == nil {
			continue
		}

		processedCount++
		issueID := int64(*issue.Number)
		currentLabels := getLabels(issue)

		// ステートレスな判定ロジックを使用してIssueを処理すべきか判断
		shouldProcess, reason := ShouldProcessIssue(issue)

		log.Printf("Issue #%d - %s (labels: %v) - Process: %v - Reason: %s",
			*issue.Number,
			safeString(issue.Title),
			currentLabels,
			shouldProcess,
			reason)

		if shouldProcess {
			processedIssueCount++

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

			// ラベル遷移を試みる
			// TransitionIssueLabelWithInfoメソッドを持つクライアントかチェック
			type labelTransitioner interface {
				TransitionIssueLabelWithInfo(ctx context.Context, owner, repo string, issueNumber int) (bool, *github.TransitionInfo, error)
			}

			if client, ok := w.client.(labelTransitioner); ok {
				transitioned, info, err := client.TransitionIssueLabelWithInfo(ctx, w.owner, w.repo, *issue.Number)
				if err != nil {
					log.Printf("Failed to transition label for issue #%d: %v", *issue.Number, err)
				} else if transitioned && info != nil {
					log.Printf("Issue #%d: %s → %s", *issue.Number, info.From, info.To)
				}
			} else {
				// 後方互換性のため、古いメソッドも試す
				transitioned, err := w.client.TransitionIssueLabel(ctx, w.owner, w.repo, *issue.Number)
				if err != nil {
					log.Printf("Failed to transition label for issue #%d: %v", *issue.Number, err)
				} else if transitioned {
					log.Printf("Successfully transitioned label for issue #%d", *issue.Number)
				}
			}

			// コールバック実行時のパニックを捕捉
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Panic recovered in callback for issue #%d: %v\nStack trace:\n%s",
							*issue.Number, r, string(debug.Stack()))
					}
				}()
				callback(issue)
			}()
		}

		// ラベル変更の追跡
		if w.labelChangeTracking {
			w.mu.Lock()
			previousLabels, exists := w.issueLabels[issueID]
			// 現在のラベルを保存
			w.issueLabels[issueID] = currentLabels
			w.mu.Unlock()

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
		}
	}
}

// GetRateLimit はGitHub APIのレート制限情報を取得する
func (w *IssueWatcher) GetRateLimit(ctx context.Context) (*gh.RateLimits, error) {
	return w.client.GetRateLimit(ctx)
}

// GetLastExecutionTime は最後の実行時刻を取得する
func (w *IssueWatcher) GetLastExecutionTime() time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastExecutionTime
}

// GetHealthStats はヘルスチェック統計情報を取得する
func (w *IssueWatcher) GetHealthStats() HealthStats {
	w.mu.Lock()
	defer w.mu.Unlock()

	return HealthStats{
		TotalExecutions:      w.totalExecutions,
		SuccessfulExecutions: w.successfulExecutions,
		FailedExecutions:     w.failedExecutions,
		LastExecutionTime:    w.lastExecutionTime,
		StartTime:            w.startTime,
	}
}

// CheckHealth はwatcherの健全性をチェックする
func (w *IssueWatcher) CheckHealth(maxInactivity time.Duration) HealthStatus {
	w.mu.Lock()
	lastExecution := w.lastExecutionTime
	totalExecutions := w.totalExecutions
	successRate := float64(0)
	if totalExecutions > 0 {
		successRate = float64(w.successfulExecutions) / float64(totalExecutions) * 100
	}
	w.mu.Unlock()

	// 一度も実行されていない場合
	if lastExecution.IsZero() {
		return HealthStatus{
			IsHealthy: false,
			Message:   "Watcher has never been executed",
		}
	}

	// 最後の実行からの経過時間をチェック
	timeSinceLastExecution := time.Since(lastExecution)
	if timeSinceLastExecution > maxInactivity {
		return HealthStatus{
			IsHealthy: false,
			Message:   fmt.Sprintf("Watcher has been inactive for %v (threshold: %v)", timeSinceLastExecution, maxInactivity),
		}
	}

	// 成功率が極端に低い場合
	if totalExecutions > 10 && successRate < 10 {
		return HealthStatus{
			IsHealthy: false,
			Message: fmt.Sprintf("Success rate is too low: %.2f%% (%d/%d executions)",
				successRate, w.successfulExecutions, totalExecutions),
		}
	}

	return HealthStatus{
		IsHealthy: true,
		Message: fmt.Sprintf("Watcher is healthy (success rate: %.2f%%, last execution: %v ago)",
			successRate, timeSinceLastExecution),
	}
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

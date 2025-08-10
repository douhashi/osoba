package watcher

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/douhashi/osoba/internal/cleanup"
	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/github"
	gh "github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
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

// ActionManagerInterface はActionManagerのインターフェース
type ActionManagerInterface interface {
	ExecuteAction(ctx context.Context, issue *github.Issue) error
	GetActionForIssue(issue *github.Issue) ActionExecutor
	SetActionFactory(factory ActionFactory)
}

// IssueWatcher はGitHub Issueを監視する構造体
type IssueWatcher struct {
	client              github.GitHubClient
	owner               string
	repo                string
	labels              []string
	pollInterval        time.Duration
	actionManager       ActionManagerInterface
	eventNotifier       *EventNotifier     // イベント通知システム
	labelChangeTracking bool               // ラベル変更追跡が有効かどうか
	issueLabels         map[int64][]string // Issue IDとラベルのマッピング
	logger              logger.Logger      // ロガー
	config              *config.Config     // 設定
	cleanupManager      cleanup.Manager    // クリーンアップマネージャー

	// ヘルスチェック用のフィールド
	lastExecutionTime    time.Time
	totalExecutions      int
	successfulExecutions int
	failedExecutions     int
	startTime            time.Time
	mu                   sync.Mutex // ヘルスチェックフィールドの保護用
}

// NewIssueWatcher は新しいIssueWatcherを作成する
func NewIssueWatcher(client github.GitHubClient, owner, repo, sessionName string, labels []string, pollInterval time.Duration, logger logger.Logger) (*IssueWatcher, error) {
	return NewIssueWatcherWithConfig(client, owner, repo, sessionName, labels, pollInterval, logger, nil, nil)
}

// NewIssueWatcherWithConfig は設定付きの新しいIssueWatcherを作成する
func NewIssueWatcherWithConfig(client github.GitHubClient, owner, repo, sessionName string, labels []string, pollInterval time.Duration, logger logger.Logger, cfg *config.Config, cleanupMgr cleanup.Manager) (*IssueWatcher, error) {
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
	if pollInterval < time.Second {
		return nil, errors.New("poll interval must be at least 1 second")
	}
	if logger == nil {
		return nil, errors.New("logger is required")
	}

	// デフォルトのcleanupManagerを作成（必要に応じて）
	if cleanupMgr == nil {
		cleanupMgr = cleanup.NewManager(logger)
	}

	return &IssueWatcher{
		client:              client,
		owner:               owner,
		repo:                repo,
		labels:              labels,
		pollInterval:        pollInterval,
		actionManager:       NewActionManager(sessionName),
		labelChangeTracking: false,
		issueLabels:         make(map[int64][]string),
		startTime:           time.Now(),
		logger:              logger.WithFields("component", "watcher", "owner", owner, "repo", repo),
		config:              cfg,
		cleanupManager:      cleanupMgr,
	}, nil
}

// SetPollInterval はポーリング間隔を設定する
func (w *IssueWatcher) SetPollInterval(interval time.Duration) error {
	if interval < time.Second {
		return errors.New("poll interval must be at least 1 second")
	}
	w.mu.Lock()
	w.pollInterval = interval
	w.mu.Unlock()
	return nil
}

// SetPollIntervalForTest はテスト用にポーリング間隔を設定する（1秒未満も許可）
func (w *IssueWatcher) SetPollIntervalForTest(interval time.Duration) {
	w.mu.Lock()
	w.pollInterval = interval
	w.mu.Unlock()
}

// GetActionManager はActionManagerを取得する
func (w *IssueWatcher) GetActionManager() ActionManagerInterface {
	return w.actionManager
}

// GetPollInterval は現在のポーリング間隔を取得する
func (w *IssueWatcher) GetPollInterval() time.Duration {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.pollInterval
}

// Start はIssue監視を開始する
func (w *IssueWatcher) Start(ctx context.Context, callback IssueCallback) {
	// ポーリング間隔を安全に取得
	pollInterval := w.GetPollInterval()
	w.logger.Info("Starting issue watcher",
		"labels", w.labels,
		"interval", pollInterval)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// 初回実行
	w.checkIssues(ctx, callback)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Stopping issue watcher")
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
			w.logger.Error("Failed to execute action for issue",
				"issueNumber", *issue.Number,
				"error", err)
		}

		// アクション実行後、必ずラベル遷移を実行
		if err := w.executeLabelTransition(ctx, issue); err != nil {
			w.logger.Error("Failed to execute label transition for issue",
				"issueNumber", *issue.Number,
				"error", err)
		}

		// ラベル遷移後、自動マージ処理を実行（設定が有効な場合）
		if w.config != nil && w.config.GitHub.AutoMergeLGTM {
			if err := executeAutoMergeIfLGTMWithLogger(ctx, issue, w.config, w.client, w.cleanupManager, w.logger); err != nil {
				w.logger.Error("Failed to execute auto-merge for issue",
					"issueNumber", *issue.Number,
					"error", err)
			}
		}
	}

	w.Start(ctx, callback)
}

// checkIssues は現在のIssueをチェックし、新しいIssueがあればコールバックを呼ぶ
func (w *IssueWatcher) checkIssues(ctx context.Context, callback IssueCallback) {
	// サイクル開始時刻
	startTime := time.Now()
	w.logger.Debug("Starting issue check cycle",
		"startTime", startTime.Format(time.RFC3339))

	// 統計情報の更新
	w.mu.Lock()
	w.totalExecutions++
	w.mu.Unlock()

	// 処理統計の記録
	var processedCount, processedIssueCount int
	var executionSuccessful bool
	defer func() {
		elapsed := time.Since(startTime)
		w.logger.Debug("Completed issue check cycle",
			"checkedIssues", processedCount,
			"processedIssues", processedIssueCount,
			"duration", elapsed)

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
			w.logger.Error("Panic recovered in checkIssues",
				"panic", r,
				"stackTrace", string(debug.Stack()))
		}
	}()

	var issues []*gh.Issue

	// リトライ付きでAPIを呼び出し
	// テスト環境では短いリトライ間隔を使用
	retryDelay := time.Second
	pollInterval := w.GetPollInterval()
	if pollInterval < time.Second {
		// ポーリング間隔が1秒未満の場合（テスト環境）は短いリトライ間隔を使用
		retryDelay = 100 * time.Millisecond
	}
	err := RetryWithBackoffLogger(ctx, w.logger, 3, retryDelay, func() error {
		var err error
		issues, err = w.client.ListIssuesByLabels(ctx, w.owner, w.repo, w.labels)
		return err
	})

	if err != nil {
		w.logger.Error("Failed to list issues",
			"error", err,
			"labels", w.labels)
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
		shouldProcess, reason := ShouldProcessIssueWithLogger(issue, w.logger)

		w.logger.Debug("Issue check result",
			"issueNumber", *issue.Number,
			"title", safeString(issue.Title),
			"labels", currentLabels,
			"shouldProcess", shouldProcess,
			"reason", reason)

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

			// ラベル遷移はActionでのみ実行される。
			// Issue検知時にはラベル遷移を実行しない。

			// コールバック実行時のパニックを捕捉
			func() {
				defer func() {
					if r := recover(); r != nil {
						w.logger.Error("Panic recovered in callback",
							"issueNumber", *issue.Number,
							"panic", r,
							"stackTrace", string(debug.Stack()))
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

					w.logger.Info("Label change detected",
						"issueNumber", *issue.Number,
						"eventType", event.Type,
						"fromLabel", event.FromLabel,
						"toLabel", event.ToLabel)

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
func NewIssueWatcherWithLabelTracking(client github.GitHubClient, owner, repo, sessionName string, labels []string, pollInterval time.Duration, logger logger.Logger) (*IssueWatcher, error) {
	watcher, err := NewIssueWatcher(client, owner, repo, sessionName, labels, pollInterval, logger)
	if err != nil {
		return nil, err
	}
	watcher.labelChangeTracking = true
	return watcher, nil
}

// executeLabelTransition は現在のラベルに基づいて適切なラベル遷移を実行する
func (w *IssueWatcher) executeLabelTransition(ctx context.Context, issue *gh.Issue) error {
	if issue == nil || issue.Number == nil {
		return fmt.Errorf("invalid issue: nil issue or issue number")
	}

	// 現在のラベルを確認して適切な遷移を実行
	transitions := []struct {
		from string
		to   string
	}{
		{"status:needs-plan", "status:planning"},
		{"status:ready", "status:implementing"},
		{"status:review-requested", "status:reviewing"},
	}

	currentLabels := getLabels(issue)
	for _, transition := range transitions {
		// 現在のラベルが遷移元ラベルを含んでいるか確認
		hasFromLabel := false
		for _, label := range currentLabels {
			if label == transition.from {
				hasFromLabel = true
				break
			}
		}

		if hasFromLabel {
			w.logger.Info("Executing label transition",
				"issueNumber", *issue.Number,
				"from", transition.from,
				"to", transition.to)

			// リトライメカニズムを実装
			const maxRetries = 3
			var lastErr error

			for attempt := 1; attempt <= maxRetries; attempt++ {
				// ラベルを削除
				if err := w.client.RemoveLabel(ctx, w.owner, w.repo, *issue.Number, transition.from); err != nil {
					lastErr = fmt.Errorf("failed to remove label %s (attempt %d/%d): %w", transition.from, attempt, maxRetries, err)
					w.logger.Warn("Failed to remove label, retrying",
						"issueNumber", *issue.Number,
						"label", transition.from,
						"attempt", attempt,
						"error", err)

					if attempt < maxRetries {
						time.Sleep(time.Duration(attempt) * time.Second) // バックオフ付きリトライ
						continue
					}
					return lastErr
				}

				// 新しいラベルを追加
				if err := w.client.AddLabel(ctx, w.owner, w.repo, *issue.Number, transition.to); err != nil {
					lastErr = fmt.Errorf("failed to add label %s (attempt %d/%d): %w", transition.to, attempt, maxRetries, err)
					w.logger.Warn("Failed to add label, retrying",
						"issueNumber", *issue.Number,
						"label", transition.to,
						"attempt", attempt,
						"error", err)

					if attempt < maxRetries {
						time.Sleep(time.Duration(attempt) * time.Second) // バックオフ付きリトライ
						continue
					}
					return lastErr
				}

				// 成功した場合はループを抜ける
				w.logger.Info("Successfully transitioned label",
					"issueNumber", *issue.Number,
					"from", transition.from,
					"to", transition.to,
					"attempt", attempt)
				return nil
			}

			// すべてのリトライが失敗した場合
			return lastErr
		}
	}

	return nil
}

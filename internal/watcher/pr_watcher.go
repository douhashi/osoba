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
	"github.com/douhashi/osoba/internal/logger"
)

// PRCallback はPR検出時に呼ばれるコールバック関数
type PRCallback func(pr *github.PullRequest)

// PRWatcher はGitHub Pull Requestを監視する構造体
type PRWatcher struct {
	client           github.GitHubClient
	owner            string
	repo             string
	labels           []string
	pollInterval     time.Duration
	logger           logger.Logger
	config           *config.Config
	cleanupManager   cleanup.Manager
	autoMergeMetrics *AutoMergeMetrics
	sessionName      string                 // tmuxセッション名（Reviseアクション用）
	actionManager    ActionManagerInterface // ReviseAction実行用

	// ヘルスチェック用のフィールド
	lastExecutionTime    time.Time
	totalExecutions      int
	successfulExecutions int
	failedExecutions     int
	startTime            time.Time
	mu                   sync.Mutex // ヘルスチェックフィールドの保護用
}

// NewPRWatcher は新しいPRWatcherを作成する
func NewPRWatcher(client github.GitHubClient, owner, repo string, labels []string, pollInterval time.Duration, logger logger.Logger) (*PRWatcher, error) {
	return NewPRWatcherWithConfig(client, owner, repo, labels, pollInterval, logger, nil, nil)
}

// NewPRWatcherWithConfig は設定付きの新しいPRWatcherを作成する
func NewPRWatcherWithConfig(client github.GitHubClient, owner, repo string, labels []string, pollInterval time.Duration, logger logger.Logger, cfg *config.Config, cleanupMgr cleanup.Manager) (*PRWatcher, error) {
	if owner == "" {
		return nil, errors.New("owner is required")
	}
	if repo == "" {
		return nil, errors.New("repo is required")
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
	// PRWatcherではsessionNameが取得できないため、空文字を渡す（従来の動作）
	if cleanupMgr == nil {
		cleanupMgr = cleanup.NewManager("", logger)
	}

	return &PRWatcher{
		client:           client,
		owner:            owner,
		repo:             repo,
		labels:           labels,
		pollInterval:     pollInterval,
		startTime:        time.Now(),
		logger:           logger.WithFields("component", "pr_watcher", "owner", owner, "repo", repo),
		config:           cfg,
		cleanupManager:   cleanupMgr,
		autoMergeMetrics: NewAutoMergeMetrics(),
	}, nil
}

// SetPollInterval はポーリング間隔を設定する
func (w *PRWatcher) SetPollInterval(interval time.Duration) error {
	if interval < time.Second {
		return errors.New("poll interval must be at least 1 second")
	}
	w.mu.Lock()
	w.pollInterval = interval
	w.mu.Unlock()
	return nil
}

// SetPollIntervalForTest はテスト用にポーリング間隔を設定する（1秒未満も許可）
func (w *PRWatcher) SetPollIntervalForTest(interval time.Duration) {
	w.mu.Lock()
	w.pollInterval = interval
	w.mu.Unlock()
}

// GetPollInterval は現在のポーリング間隔を取得する
func (w *PRWatcher) GetPollInterval() time.Duration {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.pollInterval
}

// SetActionManager はActionManagerを設定する
func (w *PRWatcher) SetActionManager(am ActionManagerInterface) {
	w.actionManager = am
}

// SetSessionName はtmuxセッション名を設定する
func (w *PRWatcher) SetSessionName(sessionName string) {
	w.sessionName = sessionName
}

// GetAutoMergeMetrics は自動マージメトリクスのスナップショットを取得する
func (w *PRWatcher) GetAutoMergeMetrics() AutoMergeMetricsSnapshot {
	if w.autoMergeMetrics == nil {
		// メトリクスが初期化されていない場合は空のスナップショットを返す
		return AutoMergeMetricsSnapshot{
			TotalAttempts:    0,
			SuccessfulMerges: 0,
			FailedMerges:     0,
			FailureReasons:   make(map[string]int64),
			StartTime:        time.Time{},
			LastAttemptTime:  time.Time{},
			SuccessRate:      0.0,
			UptimeDuration:   0,
		}
	}
	return w.autoMergeMetrics.GetSnapshot()
}

// Start はPR監視を開始する
func (w *PRWatcher) Start(ctx context.Context, callback PRCallback) {
	// ポーリング間隔を安全に取得
	pollInterval := w.GetPollInterval()
	w.logger.Info("Starting PR watcher",
		"labels", w.labels,
		"interval", pollInterval)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// 初回実行
	w.checkPRs(ctx, callback)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Stopping PR watcher")
			return
		case <-ticker.C:
			w.checkPRs(ctx, callback)
		}
	}
}

// StartWithAutoMerge はPR監視を開始し、自動マージを実行する
func (w *PRWatcher) StartWithAutoMerge(ctx context.Context) {
	callback := func(pr *github.PullRequest) {
		// ラベルベースの排他制御: status:lgtm と status:requires-changes は相互排他
		hasLGTM := hasPRLabel(pr, "status:lgtm")
		hasRequiresChanges := hasPRLabel(pr, "status:requires-changes")

		w.logger.Debug("PR label analysis",
			"prNumber", pr.Number,
			"hasLGTM", hasLGTM,
			"hasRequiresChanges", hasRequiresChanges,
			"allLabels", pr.Labels,
		)

		// status:lgtm が優先 - 自動マージを実行
		if hasLGTM {
			if w.config != nil && w.config.GitHub.AutoMergeLGTM {
				w.logger.Info("Executing auto-merge for PR with status:lgtm",
					"prNumber", pr.Number,
				)
				if err := executeAutoMergeForPRWithLogger(ctx, pr, w.config, w.client, w.cleanupManager, w.logger, w.autoMergeMetrics); err != nil {
					w.logger.Error("Failed to execute auto-merge for PR",
						"prNumber", pr.Number,
						"error", err)
				}
			}
			return // status:lgtmが存在する場合は他の処理をスキップ
		}

		// status:requires-changes - auto-revise処理を実行（status:lgtmが無い場合のみ）
		if hasRequiresChanges {
			if w.config != nil && w.config.GitHub.AutoRevisePR && w.actionManager != nil {
				w.logger.Info("Executing auto-revise for PR with status:requires-changes",
					"prNumber", pr.Number,
				)
				if err := executeAutoReviseIfRequiresChangesWithLogger(ctx, pr, w.config, w.client, w.actionManager, w.sessionName, w.logger); err != nil {
					w.logger.Error("Failed to execute auto-revise for PR",
						"prNumber", pr.Number,
						"error", err)
				}
			}
			return
		}

		// 該当するラベルがない場合はログ出力のみ
		w.logger.Debug("PR has no actionable labels (status:lgtm or status:requires-changes)",
			"prNumber", pr.Number,
			"labels", pr.Labels,
		)
	}

	w.Start(ctx, callback)
}

// hasPRLabel はPRが指定されたラベルを持っているかをチェック
func hasPRLabel(pr *github.PullRequest, labelName string) bool {
	if pr == nil || pr.Labels == nil {
		return false
	}

	for _, label := range pr.Labels {
		if label == labelName {
			return true
		}
	}
	return false
}

// checkPRs は現在のPRをチェックし、新しいPRがあればコールバックを呼ぶ
func (w *PRWatcher) checkPRs(ctx context.Context, callback PRCallback) {
	// サイクル開始時刻
	startTime := time.Now()
	w.logger.Debug("Starting PR check cycle",
		"startTime", startTime.Format(time.RFC3339))

	// 統計情報の更新
	w.mu.Lock()
	w.totalExecutions++
	w.mu.Unlock()

	// 処理統計の記録
	var processedCount, processedPRCount int
	var executionSuccessful bool
	defer func() {
		elapsed := time.Since(startTime)
		w.logger.Debug("Completed PR check cycle",
			"checkedPRs", processedCount,
			"processedPRs", processedPRCount,
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
			w.logger.Error("Panic recovered in checkPRs",
				"panic", r,
				"stackTrace", string(debug.Stack()))
		}
	}()

	var prs []*github.PullRequest

	// リトライ付きでAPIを呼び出し
	retryDelay := time.Second
	pollInterval := w.GetPollInterval()
	if pollInterval < time.Second {
		// ポーリング間隔が1秒未満の場合（テスト環境）は短いリトライ間隔を使用
		retryDelay = 100 * time.Millisecond
	}

	err := RetryWithBackoffLogger(ctx, w.logger, 3, retryDelay, func() error {
		var err error
		prs, err = w.client.ListPullRequestsByLabels(ctx, w.owner, w.repo, w.labels)
		return err
	})

	if err != nil {
		w.logger.Error("Failed to list pull requests",
			"error", err,
			"labels", w.labels)
		return
	}

	// API呼び出しが成功
	executionSuccessful = true

	for _, pr := range prs {
		if pr == nil || pr.Number == 0 {
			continue
		}

		processedCount++

		w.logger.Debug("PR check result",
			"prNumber", pr.Number,
			"title", pr.Title,
			"state", pr.State,
			"mergeable", pr.Mergeable,
			"isDraft", pr.IsDraft,
			"checksStatus", pr.ChecksStatus)

		// PRを処理対象と判定（Open状態かつDraftでない）
		if pr.State == "OPEN" && !pr.IsDraft {
			processedPRCount++

			// コールバック実行時のパニックを捕捉
			func() {
				defer func() {
					if r := recover(); r != nil {
						w.logger.Error("Panic recovered in callback",
							"prNumber", pr.Number,
							"panic", r,
							"stackTrace", string(debug.Stack()))
					}
				}()
				callback(pr)
			}()
		}
	}
}

// GetRateLimit はGitHub APIのレート制限情報を取得する
func (w *PRWatcher) GetRateLimit(ctx context.Context) (*github.RateLimits, error) {
	return w.client.GetRateLimit(ctx)
}

// GetLastExecutionTime は最後の実行時刻を取得する
func (w *PRWatcher) GetLastExecutionTime() time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastExecutionTime
}

// GetHealthStats はヘルスチェック統計情報を取得する
func (w *PRWatcher) GetHealthStats() HealthStats {
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
func (w *PRWatcher) CheckHealth(maxInactivity time.Duration) HealthStatus {
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
			Message:   "PR Watcher has never been executed",
		}
	}

	// 最後の実行からの経過時間をチェック
	timeSinceLastExecution := time.Since(lastExecution)
	if timeSinceLastExecution > maxInactivity {
		return HealthStatus{
			IsHealthy: false,
			Message:   fmt.Sprintf("PR Watcher has been inactive for %v (threshold: %v)", timeSinceLastExecution, maxInactivity),
		}
	}

	// 成功率が極端に低い場合
	if totalExecutions > 10 && successRate < 10 {
		return HealthStatus{
			IsHealthy: false,
			Message: fmt.Sprintf("PR Watcher success rate is too low: %.2f%% (%d/%d executions)",
				successRate, w.successfulExecutions, totalExecutions),
		}
	}

	return HealthStatus{
		IsHealthy: true,
		Message: fmt.Sprintf("PR Watcher is healthy (success rate: %.2f%%, last execution: %v ago)",
			successRate, timeSinceLastExecution),
	}
}

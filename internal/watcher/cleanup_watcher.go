package watcher

import (
	"context"
	"errors"
	"time"

	"github.com/douhashi/osoba/internal/cleanup"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
)

// CleanupWatcher は定期的にクリーンアップを実行するウォッチャー
type CleanupWatcher struct {
	client         github.GitHubClient
	owner          string
	repo           string
	interval       time.Duration
	cleanupManager cleanup.Manager
	logger         logger.Logger
}

// NewCleanupWatcher は新しいCleanupWatcherを作成する
func NewCleanupWatcher(
	client github.GitHubClient,
	owner, repo string,
	interval time.Duration,
	cleanupManager cleanup.Manager,
	logger logger.Logger,
) (*CleanupWatcher, error) {
	if client == nil {
		return nil, errors.New("github client is required")
	}
	if owner == "" {
		return nil, errors.New("owner is required")
	}
	if repo == "" {
		return nil, errors.New("repo is required")
	}
	if interval <= 0 {
		return nil, errors.New("interval must be positive")
	}
	if cleanupManager == nil {
		return nil, errors.New("cleanup manager is required")
	}

	return &CleanupWatcher{
		client:         client,
		owner:          owner,
		repo:           repo,
		interval:       interval,
		cleanupManager: cleanupManager,
		logger:         logger,
	}, nil
}

// Start はクリーンアップウォッチャーを開始する
func (w *CleanupWatcher) Start(ctx context.Context) {
	if w.logger != nil {
		w.logger.Info("Starting cleanup watcher",
			"owner", w.owner,
			"repo", w.repo,
			"interval", w.interval,
		)
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// 初回実行
	w.performCleanup(ctx)

	for {
		select {
		case <-ctx.Done():
			if w.logger != nil {
				w.logger.Info("Cleanup watcher stopped")
			}
			return
		case <-ticker.C:
			w.performCleanup(ctx)
		}
	}
}

// performCleanup は実際のクリーンアップ処理を実行する
func (w *CleanupWatcher) performCleanup(ctx context.Context) {
	if w.logger != nil {
		w.logger.Debug("Performing cleanup check")
	}

	// 閉じられたIssueを取得
	closedIssues, err := w.client.ListClosedIssues(ctx, w.owner, w.repo)
	if err != nil {
		if w.logger != nil {
			w.logger.Error("Failed to list closed issues",
				"error", err,
			)
		}
		return
	}

	if len(closedIssues) == 0 {
		if w.logger != nil {
			w.logger.Debug("No closed issues found")
		}
		return
	}

	if w.logger != nil {
		w.logger.Info("Found closed issues for cleanup",
			"count", len(closedIssues),
		)
	}

	// 各Issueに対してクリーンアップを実行
	for _, issue := range closedIssues {
		if issue.Number == nil {
			if w.logger != nil {
				w.logger.Warn("Issue without number, skipping")
			}
			continue
		}

		issueNumber := *issue.Number

		// クリーンアップ実行
		if err := w.cleanupManager.CleanupIssueResources(ctx, issueNumber); err != nil {
			if w.logger != nil {
				w.logger.Error("Failed to cleanup issue resources",
					"issue_number", issueNumber,
					"error", err,
				)
			}
			// エラーがあっても他のIssueの処理は続ける
			continue
		}

		if w.logger != nil {
			w.logger.Info("Successfully cleaned up issue resources",
				"issue_number", issueNumber,
			)
		}
	}
}

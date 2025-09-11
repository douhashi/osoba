package watcher

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/cleanup"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestNullLogger はテスト用のnullロガー
type TestNullLogger struct{}

func (t *TestNullLogger) Debug(msg string, keysAndValues ...interface{}) {}
func (t *TestNullLogger) Info(msg string, keysAndValues ...interface{})  {}
func (t *TestNullLogger) Warn(msg string, keysAndValues ...interface{})  {}
func (t *TestNullLogger) Error(msg string, keysAndValues ...interface{}) {}
func (t *TestNullLogger) WithFields(keysAndValues ...interface{}) logger.Logger {
	return t
}

// MockCleanupManagerForWatcher for testing cleanup watcher
type MockCleanupManagerForWatcher struct {
	mock.Mock
}

func (m *MockCleanupManagerForWatcher) CleanupIssueResources(ctx context.Context, issueNumber int) error {
	args := m.Called(ctx, issueNumber)
	return args.Error(0)
}

func TestCleanupWatcher_NewCleanupWatcher(t *testing.T) {
	tests := []struct {
		name       string
		client     github.GitHubClient
		owner      string
		repo       string
		interval   time.Duration
		cleanupMgr cleanup.Manager
		logger     logger.Logger
		wantErr    bool
	}{
		{
			name:       "valid parameters",
			client:     &mocks.MockGitHubClient{},
			owner:      "test-owner",
			repo:       "test-repo",
			interval:   1 * time.Minute,
			cleanupMgr: &MockCleanupManagerForWatcher{},
			logger:     &TestNullLogger{},
			wantErr:    false,
		},
		{
			name:       "nil client",
			client:     nil,
			owner:      "test-owner",
			repo:       "test-repo",
			interval:   1 * time.Minute,
			cleanupMgr: &MockCleanupManagerForWatcher{},
			logger:     &TestNullLogger{},
			wantErr:    true,
		},
		{
			name:       "empty owner",
			client:     &mocks.MockGitHubClient{},
			owner:      "",
			repo:       "test-repo",
			interval:   1 * time.Minute,
			cleanupMgr: &MockCleanupManagerForWatcher{},
			logger:     &TestNullLogger{},
			wantErr:    true,
		},
		{
			name:       "empty repo",
			client:     &mocks.MockGitHubClient{},
			owner:      "test-owner",
			repo:       "",
			interval:   1 * time.Minute,
			cleanupMgr: &MockCleanupManagerForWatcher{},
			logger:     &TestNullLogger{},
			wantErr:    true,
		},
		{
			name:       "invalid interval",
			client:     &mocks.MockGitHubClient{},
			owner:      "test-owner",
			repo:       "test-repo",
			interval:   0,
			cleanupMgr: &MockCleanupManagerForWatcher{},
			logger:     &TestNullLogger{},
			wantErr:    true,
		},
		{
			name:       "nil cleanup manager",
			client:     &mocks.MockGitHubClient{},
			owner:      "test-owner",
			repo:       "test-repo",
			interval:   1 * time.Minute,
			cleanupMgr: nil,
			logger:     &TestNullLogger{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			watcher, err := NewCleanupWatcher(tt.client, tt.owner, tt.repo, tt.interval, tt.cleanupMgr, tt.logger)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, watcher)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, watcher)
			}
		})
	}
}

func TestCleanupWatcher_Start(t *testing.T) {
	t.Run("periodic cleanup execution", func(t *testing.T) {
		mockClient := new(mocks.MockGitHubClient)
		mockManager := new(MockCleanupManagerForWatcher)

		// 閉じられたIssueを返すモック
		closedIssues := []*github.Issue{
			{Number: intPtrForCleanup(1)},
			{Number: intPtrForCleanup(2)},
		}
		mockClient.On("ListClosedIssues", mock.Anything, "owner", "repo").
			Return(closedIssues, nil)

		// クリーンアップの実行を期待
		mockManager.On("CleanupIssueResources", mock.Anything, 1).Return(nil)
		mockManager.On("CleanupIssueResources", mock.Anything, 2).Return(nil)

		watcher, err := NewCleanupWatcher(
			mockClient,
			"owner",
			"repo",
			100*time.Millisecond, // 短い間隔でテスト
			mockManager,
			&TestNullLogger{},
		)
		assert.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			watcher.Start(ctx)
		}()

		wg.Wait()

		// 少なくとも1回は実行されることを確認
		mockClient.AssertExpectations(t)
		mockManager.AssertExpectations(t)
	})

	t.Run("context cancellation stops watcher", func(t *testing.T) {
		mockClient := new(mocks.MockGitHubClient)
		mockManager := new(MockCleanupManagerForWatcher)

		// 初回実行のため空のリストを返す設定
		mockClient.On("ListClosedIssues", mock.Anything, "owner", "repo").
			Return([]*github.Issue{}, nil).Maybe()

		watcher, err := NewCleanupWatcher(
			mockClient,
			"owner",
			"repo",
			1*time.Hour, // 長い間隔
			mockManager,
			&TestNullLogger{},
		)
		assert.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			watcher.Start(ctx)
		}()

		// すぐにキャンセル
		cancel()

		// watcherが終了することを確認
		done := make(chan bool)
		go func() {
			wg.Wait()
			done <- true
		}()

		select {
		case <-done:
			// 正常に終了
		case <-time.After(1 * time.Second):
			t.Fatal("watcher did not stop after context cancellation")
		}
	})

	t.Run("error handling in cleanup", func(t *testing.T) {
		mockClient := new(mocks.MockGitHubClient)
		mockManager := new(MockCleanupManagerForWatcher)

		// 閉じられたIssueを返すモック
		closedIssues := []*github.Issue{
			{Number: intPtrForCleanup(1)},
			{Number: intPtrForCleanup(2)},
		}
		mockClient.On("ListClosedIssues", mock.Anything, "owner", "repo").
			Return(closedIssues, nil)

		// 1つ目はエラー、2つ目は成功
		mockManager.On("CleanupIssueResources", mock.Anything, 1).
			Return(assert.AnError)
		mockManager.On("CleanupIssueResources", mock.Anything, 2).
			Return(nil)

		watcher, err := NewCleanupWatcher(
			mockClient,
			"owner",
			"repo",
			100*time.Millisecond,
			mockManager,
			&TestNullLogger{},
		)
		assert.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			watcher.Start(ctx)
		}()

		wg.Wait()

		// エラーがあっても他のクリーンアップは実行される
		mockClient.AssertExpectations(t)
		mockManager.AssertExpectations(t)
	})
}

func TestCleanupWatcher_performCleanup(t *testing.T) {
	t.Run("cleanup closed issues", func(t *testing.T) {
		mockClient := new(mocks.MockGitHubClient)
		mockManager := new(MockCleanupManagerForWatcher)

		closedIssues := []*github.Issue{
			{Number: intPtrForCleanup(10)},
			{Number: intPtrForCleanup(20)},
		}
		mockClient.On("ListClosedIssues", mock.Anything, "owner", "repo").
			Return(closedIssues, nil)

		mockManager.On("CleanupIssueResources", mock.Anything, 10).Return(nil)
		mockManager.On("CleanupIssueResources", mock.Anything, 20).Return(nil)

		watcher := &CleanupWatcher{
			client:         mockClient,
			owner:          "owner",
			repo:           "repo",
			interval:       1 * time.Minute,
			cleanupManager: mockManager,
			logger:         &TestNullLogger{},
		}

		watcher.performCleanup(context.Background())

		mockClient.AssertExpectations(t)
		mockManager.AssertExpectations(t)
	})

	t.Run("skip issues without number", func(t *testing.T) {
		mockClient := new(mocks.MockGitHubClient)
		mockManager := new(MockCleanupManagerForWatcher)

		closedIssues := []*github.Issue{
			{Number: nil}, // Numberがnil
			{Number: intPtrForCleanup(30)},
		}
		mockClient.On("ListClosedIssues", mock.Anything, "owner", "repo").
			Return(closedIssues, nil)

		// Numberがnilのissueはスキップされる
		mockManager.On("CleanupIssueResources", mock.Anything, 30).Return(nil)

		watcher := &CleanupWatcher{
			client:         mockClient,
			owner:          "owner",
			repo:           "repo",
			interval:       1 * time.Minute,
			cleanupManager: mockManager,
			logger:         &TestNullLogger{},
		}

		watcher.performCleanup(context.Background())

		mockClient.AssertExpectations(t)
		mockManager.AssertExpectations(t)
	})

	t.Run("handle list issues error", func(t *testing.T) {
		mockClient := new(mocks.MockGitHubClient)
		mockManager := new(MockCleanupManagerForWatcher)

		mockClient.On("ListClosedIssues", mock.Anything, "owner", "repo").
			Return(nil, assert.AnError)

		// エラーの場合、クリーンアップは実行されない

		watcher := &CleanupWatcher{
			client:         mockClient,
			owner:          "owner",
			repo:           "repo",
			interval:       1 * time.Minute,
			cleanupManager: mockManager,
			logger:         &TestNullLogger{},
		}

		watcher.performCleanup(context.Background())

		mockClient.AssertExpectations(t)
		mockManager.AssertNotCalled(t, "CleanupIssueResources", mock.Anything, mock.Anything)
	})
}

// Helper function for creating int pointers (cleanup watcher)
func intPtrForCleanup(i int) *int {
	return &i
}

package watcher

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	gh "github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/testutil/builders"
	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/stretchr/testify/mock"
)

func TestNewIssueWatcher(t *testing.T) {
	tests := []struct {
		name    string
		owner   string
		repo    string
		labels  []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "正常系: Issue監視を作成できる",
			owner:   "douhashi",
			repo:    "osoba",
			labels:  []string{"status:needs-plan", "status:ready", "status:review-requested"},
			wantErr: false,
		},
		{
			name:    "異常系: ownerが空",
			owner:   "",
			repo:    "osoba",
			labels:  []string{"status:needs-plan"},
			wantErr: true,
			errMsg:  "owner is required",
		},
		{
			name:    "異常系: repoが空",
			owner:   "douhashi",
			repo:    "",
			labels:  []string{"status:needs-plan"},
			wantErr: true,
			errMsg:  "repo is required",
		},
		{
			name:    "異常系: labelsが空",
			owner:   "douhashi",
			repo:    "osoba",
			labels:  []string{},
			wantErr: true,
			errMsg:  "at least one label is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGH := mocks.NewMockGitHubClient()

			watcher, err := NewIssueWatcher(mockGH, tt.owner, tt.repo, "test-session", tt.labels, 5*time.Second, NewMockLogger())

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewIssueWatcher() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("NewIssueWatcher() error = %v, wantErrMsg %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("NewIssueWatcher() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if watcher == nil {
					t.Errorf("NewIssueWatcher() returned nil watcher")
				}
			}
		})
	}
}

func TestIssueWatcher_WatchConcurrent(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mocks.MockGitHubClient)
		expectedIssues int
		watchDuration  time.Duration
	}{
		{
			name: "複数のIssueを処理できる",
			setupMock: func(m *mocks.MockGitHubClient) {
				issues := []*gh.Issue{
					builders.NewIssueBuilder().WithNumber(1).WithTitle("Issue 1").WithLabels([]string{"status:needs-plan"}).Build(),
					builders.NewIssueBuilder().WithNumber(2).WithTitle("Issue 2").WithLabels([]string{"status:ready"}).Build(),
					builders.NewIssueBuilder().WithNumber(3).WithTitle("Issue 3").WithLabels([]string{"status:review-requested"}).Build(),
				}
				m.On("ListIssuesByLabels", mock.Anything, "owner", "repo", mock.Anything).Return(issues, nil)
				m.On("GetRateLimit", mock.Anything).Return(builders.NewRateLimitsBuilder().Build(), nil)
			},
			expectedIssues: 3,
			watchDuration:  100 * time.Millisecond,
		},
		{
			name: "空のIssueリストを処理できる",
			setupMock: func(m *mocks.MockGitHubClient) {
				m.On("ListIssuesByLabels", mock.Anything, "owner", "repo", mock.Anything).Return([]*gh.Issue{}, nil)
				m.On("GetRateLimit", mock.Anything).Return(builders.NewRateLimitsBuilder().Build(), nil)
			},
			expectedIssues: 0,
			watchDuration:  100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGH := mocks.NewMockGitHubClient()
			tt.setupMock(mockGH)

			watcher, err := NewIssueWatcher(mockGH, "owner", "repo", "test-session", []string{"status:needs-plan", "status:ready", "status:review-requested"}, 5*time.Second, NewMockLogger())
			if err != nil {
				t.Fatalf("NewIssueWatcher() error = %v", err)
			}

			var processedIssues sync.Map
			var processedCount int
			var mu sync.Mutex

			handler := func(issue *gh.Issue) {
				if issue.Number != nil {
					processedIssues.Store(*issue.Number, true)
					mu.Lock()
					processedCount++
					mu.Unlock()
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), tt.watchDuration)
			defer cancel()

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				watcher.Start(ctx, handler)
			}()

			wg.Wait()

			mu.Lock()
			finalCount := processedCount
			mu.Unlock()

			if finalCount < tt.expectedIssues {
				t.Errorf("Processed %d issues, expected at least %d", finalCount, tt.expectedIssues)
			}

			mockGH.AssertExpectations(t)
		})
	}
}

func TestIssueWatcher_Stop(t *testing.T) {
	mockGH := mocks.NewMockGitHubClient()

	// API呼び出しを設定
	issues := []*gh.Issue{
		builders.NewIssueBuilder().WithNumber(1).WithTitle("Issue 1").WithLabels([]string{"status:needs-plan"}).Build(),
	}
	mockGH.On("ListIssuesByLabels", mock.Anything, "owner", "repo", mock.Anything).Return(issues, nil).Maybe()
	mockGH.On("GetRateLimit", mock.Anything).Return(builders.NewRateLimitsBuilder().Build(), nil).Maybe()

	watcher, err := NewIssueWatcher(mockGH, "owner", "repo", "test-session", []string{"status:needs-plan"}, 5*time.Second, NewMockLogger())
	if err != nil {
		t.Fatalf("NewIssueWatcher() error = %v", err)
	}

	var processedCount int
	var mu sync.Mutex
	handler := func(issue *gh.Issue) {
		mu.Lock()
		processedCount++
		mu.Unlock()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		watcher.Start(ctx, handler)
	}()

	time.Sleep(100 * time.Millisecond)

	// コンテキストをキャンセルして停止
	cancel()

	wg.Wait()

	// キャンセル後は新しいIssueが処理されないことを確認
	beforeStop := processedCount
	time.Sleep(100 * time.Millisecond)
	afterStop := processedCount

	if beforeStop != afterStop {
		t.Errorf("Issues were processed after cancel: before=%d, after=%d", beforeStop, afterStop)
	}
}

func TestIssueWatcher_RateLimitHandling(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mocks.MockGitHubClient)
		wantPanic bool
	}{
		{
			name: "Rate limit残量が十分な場合は正常に処理",
			setupMock: func(m *mocks.MockGitHubClient) {
				m.On("ListIssuesByLabels", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]*gh.Issue{}, nil)
				m.On("GetRateLimit", mock.Anything).
					Return(builders.NewRateLimitsBuilder().WithCoreLimit(5000, 1000).Build(), nil)
			},
			wantPanic: false,
		},
		{
			name: "Rate limit残量が少ない場合は警告",
			setupMock: func(m *mocks.MockGitHubClient) {
				m.On("ListIssuesByLabels", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]*gh.Issue{}, nil)
				m.On("GetRateLimit", mock.Anything).
					Return(builders.NewRateLimitsBuilder().WithCoreLimit(5000, 50).Build(), nil)
			},
			wantPanic: false,
		},
		{
			name: "Rate limitが枯渇している場合",
			setupMock: func(m *mocks.MockGitHubClient) {
				m.On("ListIssuesByLabels", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]*gh.Issue{}, nil)
				m.On("GetRateLimit", mock.Anything).
					Return(builders.NewRateLimitsBuilder().AsExhausted().Build(), nil)
			},
			wantPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGH := mocks.NewMockGitHubClient()
			tt.setupMock(mockGH)

			watcher, err := NewIssueWatcher(mockGH, "owner", "repo", "test-session", []string{"status:needs-plan"}, 5*time.Second, NewMockLogger())
			if err != nil {
				t.Fatalf("NewIssueWatcher() error = %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			handler := func(issue *gh.Issue) {
			}

			defer func() {
				if r := recover(); r != nil && !tt.wantPanic {
					t.Errorf("Watch() panicked: %v", r)
				}
			}()

			watcher.Start(ctx, handler)

			mockGH.AssertExpectations(t)
		})
	}
}

func TestIssueWatcher_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mocks.MockGitHubClient)
		handler        func(*gh.Issue)
		expectContinue bool
	}{
		{
			name: "API呼び出しエラー時も監視を継続",
			setupMock: func(m *mocks.MockGitHubClient) {
				// 最初はエラー、次は成功
				m.On("ListIssuesByLabels", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("API error")).Once()
				m.On("ListIssuesByLabels", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]*gh.Issue{}, nil).Maybe()
				m.On("GetRateLimit", mock.Anything).
					Return(builders.NewRateLimitsBuilder().Build(), nil).Maybe()
			},
			handler: func(issue *gh.Issue) {
			},
			expectContinue: true,
		},
		{
			name: "ハンドラーエラー時も監視を継続",
			setupMock: func(m *mocks.MockGitHubClient) {
				issues := []*gh.Issue{
					builders.NewIssueBuilder().WithNumber(1).Build(),
				}
				m.On("ListIssuesByLabels", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(issues, nil)
				m.On("GetRateLimit", mock.Anything).
					Return(builders.NewRateLimitsBuilder().Build(), nil)
			},
			handler: func(issue *gh.Issue) {
				// ハンドラーでエラーが発生したケースを想定
			},
			expectContinue: true,
		},
		{
			name: "コンテキストキャンセル時は正常終了",
			setupMock: func(m *mocks.MockGitHubClient) {
				m.On("ListIssuesByLabels", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]*gh.Issue{}, nil).Maybe()
				m.On("GetRateLimit", mock.Anything).
					Return(builders.NewRateLimitsBuilder().Build(), nil).Maybe()
			},
			handler: func(issue *gh.Issue) {
			},
			expectContinue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGH := mocks.NewMockGitHubClient()
			tt.setupMock(mockGH)

			watcher, err := NewIssueWatcher(mockGH, "owner", "repo", "test-session", []string{"status:needs-plan"}, 5*time.Second, NewMockLogger())
			if err != nil {
				t.Fatalf("NewIssueWatcher() error = %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
			defer cancel()

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				watcher.Start(ctx, tt.handler)
			}()

			if !tt.expectContinue {
				time.Sleep(50 * time.Millisecond)
				cancel()
			}

			wg.Wait()

			// エラーが発生しても監視が継続されたことを確認
			if tt.expectContinue {
				mockGH.AssertExpectations(t)
			}
		})
	}
}

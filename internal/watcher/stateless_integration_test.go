//go:build integration
// +build integration

package watcher

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/testutil/builders"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStatelessLabelReprocessing はラベルを手動で戻した際の再処理動作を検証する
func TestStatelessLabelReprocessing(t *testing.T) {
	tests := []struct {
		name                 string
		scenario             string
		initialLabels        []string
		manualLabelChange    func(*statelessMockClient, int)
		expectedActions      []string
		expectedLabelChanges []labelChange
	}{
		{
			name:          "implementing から ready に戻した場合の再処理",
			scenario:      "開発者が実装を一時中断し、再度実装を開始する場合",
			initialLabels: []string{"status:implementing"},
			manualLabelChange: func(client *statelessMockClient, issueNumber int) {
				// implementing を削除し、ready を追加（手動でラベルを戻す）
				client.simulateManualLabelChange(issueNumber, []string{"status:ready"})
			},
			expectedActions: []string{"implementation"},
			expectedLabelChanges: []labelChange{
				{operation: "remove", label: "status:ready"},
				{operation: "add", label: "status:implementing"},
			},
		},
		{
			name:          "reviewing から review-requested に戻した場合の再処理",
			scenario:      "レビュアーがレビューを中断し、再度レビューを開始する場合",
			initialLabels: []string{"status:reviewing"},
			manualLabelChange: func(client *statelessMockClient, issueNumber int) {
				client.simulateManualLabelChange(issueNumber, []string{"status:review-requested"})
			},
			expectedActions: []string{"review"},
			expectedLabelChanges: []labelChange{
				{operation: "remove", label: "status:review-requested"},
				{operation: "add", label: "status:reviewing"},
			},
		},
		{
			name:          "planning から needs-plan に戻した場合の再処理",
			scenario:      "計画を見直すために一時的に戻す場合",
			initialLabels: []string{"status:planning"},
			manualLabelChange: func(client *statelessMockClient, issueNumber int) {
				client.simulateManualLabelChange(issueNumber, []string{"status:needs-plan"})
			},
			expectedActions: []string{"plan"},
			expectedLabelChanges: []labelChange{
				{operation: "remove", label: "status:needs-plan"},
				{operation: "add", label: "status:planning"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			issue := builders.NewIssueBuilder().
				WithNumber(123).
				WithTitle("Test Issue").
				WithLabels(tt.initialLabels).
				Build()

			mockClient := newStatelessMockClient([]*github.Issue{issue})
			mockLogger := NewMockLogger()
			executedActions := []string{}
			actionMu := sync.Mutex{}

			// ActionFactoryのモック設定
			factory := &mockActionFactory{
				planAction:           createMockAction("plan", "status:needs-plan", &executedActions, &actionMu),
				implementationAction: createMockAction("implementation", "status:ready", &executedActions, &actionMu),
				reviewAction:         createMockAction("review", "status:review-requested", &executedActions, &actionMu),
			}

			// IssueWatcherを作成
			watcher, err := NewIssueWatcher(
				mockClient,
				"owner",
				"repo",
				"test-session",
				[]string{"status:needs-plan", "status:ready", "status:review-requested"},
				1*time.Second,
				mockLogger,
			)
			require.NoError(t, err)
			watcher.GetActionManager().SetActionFactory(factory)

			// Act
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// osobaを起動
			go watcher.StartWithActions(ctx)

			// 初回の処理が完了するまで待機
			time.Sleep(1500 * time.Millisecond)

			// 手動でラベルを変更
			tt.manualLabelChange(mockClient, 123)

			// 再処理が完了するまで待機
			time.Sleep(2000 * time.Millisecond)

			// Assert
			actionMu.Lock()
			assert.Equal(t, tt.expectedActions, executedActions, "期待されるアクションが実行されていない")
			actionMu.Unlock()

			// ラベル変更の検証
			actualChanges := mockClient.getLabelChanges()
			assert.Equal(t, len(tt.expectedLabelChanges), len(actualChanges), "ラベル変更の回数が異なる")

			for i, expected := range tt.expectedLabelChanges {
				if i < len(actualChanges) {
					assert.Equal(t, expected.operation, actualChanges[i].operation, "ラベル操作が異なる")
					assert.Equal(t, expected.label, actualChanges[i].label, "ラベル名が異なる")
				}
			}
		})
	}
}

// TestStatelessMultipleInstanceHandling は複数のosobaインスタンスでの同一Issue処理を検証する
func TestStatelessMultipleInstanceHandling(t *testing.T) {
	// Arrange
	issue := builders.NewIssueBuilder().
		WithNumber(456).
		WithTitle("Concurrent Test Issue").
		WithLabels([]string{"status:ready"}).
		Build()

	mockClient := newStatelessMockClient([]*github.Issue{issue})
	mockLogger := NewMockLogger()
	executionCount := 0
	executionMu := sync.Mutex{}

	// 実行回数をカウントするアクション
	countingAction := &mockAction{
		canExecute: func(issue *github.Issue) bool {
			return hasLabel(issue, "status:ready")
		},
		execute: func(ctx context.Context, issue *github.Issue) error {
			executionMu.Lock()
			executionCount++
			executionMu.Unlock()
			// 処理に時間がかかることをシミュレート
			time.Sleep(50 * time.Millisecond)
			return nil
		},
	}

	factory := &mockActionFactory{
		implementationAction: countingAction,
	}

	// 3つのosobaインスタンスを作成
	watchers := make([]*IssueWatcher, 3)
	for i := 0; i < 3; i++ {
		watcher, err := NewIssueWatcher(
			mockClient,
			"owner",
			"repo",
			"test-session-"+string(rune(i)),
			[]string{"status:ready"},
			1*time.Second,
			mockLogger,
		)
		require.NoError(t, err)
		watcher.GetActionManager().SetActionFactory(factory)
		watchers[i] = watcher
	}

	// Act
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 全インスタンスを同時に起動
	for _, watcher := range watchers {
		go watcher.StartWithActions(ctx)
	}

	// 処理が完了するまで待機
	time.Sleep(2 * time.Second)

	// Assert
	executionMu.Lock()
	actualCount := executionCount
	executionMu.Unlock()

	// 複数インスタンスが動作していても、アクションは1回だけ実行される
	assert.Equal(t, 1, actualCount, "アクションが複数回実行されている（多重実行の防止が機能していない）")

	// ラベル遷移も1回だけ実行されることを確認
	labelChanges := mockClient.getLabelChanges()
	removeCount := 0
	addCount := 0
	for _, change := range labelChanges {
		if change.operation == "remove" && change.label == "status:ready" {
			removeCount++
		}
		if change.operation == "add" && change.label == "status:implementing" {
			addCount++
		}
	}
	assert.Equal(t, 1, removeCount, "status:ready の削除が複数回実行されている")
	assert.Equal(t, 1, addCount, "status:implementing の追加が複数回実行されている")
}

// TestStatelessErrorRecovery はエラー時のリカバリー動作を検証する
func TestStatelessErrorRecovery(t *testing.T) {
	tests := []struct {
		name               string
		errorScenario      func(*statelessMockClient)
		expectedRetries    int
		expectFinalSuccess bool
	}{
		{
			name: "一時的なネットワークエラーからのリカバリー",
			errorScenario: func(client *statelessMockClient) {
				// 最初の2回はエラー、3回目で成功
				client.setErrorCount(2)
			},
			expectedRetries:    3, // 初回 + リトライ2回
			expectFinalSuccess: true,
		},
		{
			name: "永続的なエラーでの最大リトライ",
			errorScenario: func(client *statelessMockClient) {
				// 常にエラーを返す
				client.setErrorCount(999)
			},
			expectedRetries:    3, // 最大リトライ回数
			expectFinalSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			issue := builders.NewIssueBuilder().
				WithNumber(789).
				WithTitle("Error Recovery Test").
				WithLabels([]string{"status:ready"}).
				Build()

			mockClient := newStatelessMockClient([]*github.Issue{issue})
			mockLogger := NewMockLogger()
			tt.errorScenario(mockClient)

			attemptCount := 0
			attemptMu := sync.Mutex{}

			// リトライ回数をカウントするアクション
			retryCountingAction := &mockAction{
				canExecute: func(issue *github.Issue) bool {
					return hasLabel(issue, "status:ready")
				},
				execute: func(ctx context.Context, issue *github.Issue) error {
					attemptMu.Lock()
					attemptCount++
					attemptMu.Unlock()
					return nil
				},
			}

			factory := &mockActionFactory{
				implementationAction: retryCountingAction,
			}

			watcher, err := NewIssueWatcher(
				mockClient,
				"owner",
				"repo",
				"test-session",
				[]string{"status:ready"},
				1*time.Second,
				mockLogger,
			)
			require.NoError(t, err)
			watcher.GetActionManager().SetActionFactory(factory)

			// Act
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			go watcher.StartWithActions(ctx)

			// リトライが完了するまで待機
			time.Sleep(3 * time.Second)

			// Assert
			attemptMu.Lock()
			actualAttempts := attemptCount
			attemptMu.Unlock()

			// アクション実行回数の検証
			assert.Equal(t, 1, actualAttempts, "アクションの実行回数が期待値と異なる")

			// ラベル遷移のリトライ回数を検証
			removeLabelAttempts := mockClient.getOperationCount("RemoveLabel")
			assert.LessOrEqual(t, removeLabelAttempts, tt.expectedRetries, "RemoveLabelのリトライ回数が多すぎる")

			if tt.expectFinalSuccess {
				// 最終的に成功した場合、ラベル遷移が完了していることを確認
				labelChanges := mockClient.getLabelChanges()
				hasSuccessfulTransition := false
				for _, change := range labelChanges {
					if change.operation == "add" && change.label == "status:implementing" {
						hasSuccessfulTransition = true
						break
					}
				}
				assert.True(t, hasSuccessfulTransition, "最終的にラベル遷移が成功していない")
			}
		})
	}
}

// statelessMockClient はステートレステスト用のモッククライアント
type statelessMockClient struct {
	mu              sync.Mutex
	issues          map[int]*github.Issue
	labelChanges    []labelChange
	errorCount      int
	currentErrors   int
	operationCounts map[string]int
}

type labelChange struct {
	issueNumber int
	operation   string // "add" or "remove"
	label       string
}

func newStatelessMockClient(issues []*github.Issue) *statelessMockClient {
	client := &statelessMockClient{
		issues:          make(map[int]*github.Issue),
		labelChanges:    []labelChange{},
		operationCounts: make(map[string]int),
	}
	for _, issue := range issues {
		if issue.Number != nil {
			client.issues[*issue.Number] = issue
		}
	}
	return client
}

func (m *statelessMockClient) simulateManualLabelChange(issueNumber int, newLabels []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if issue, exists := m.issues[issueNumber]; exists {
		// 新しいラベルセットを作成
		issue.Labels = []*github.Label{}
		for _, label := range newLabels {
			issue.Labels = append(issue.Labels, &github.Label{
				Name: github.String(label),
			})
		}
	}
}

func (m *statelessMockClient) setErrorCount(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCount = count
	m.currentErrors = 0
}

func (m *statelessMockClient) getLabelChanges() []labelChange {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]labelChange{}, m.labelChanges...)
}

func (m *statelessMockClient) getOperationCount(operation string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.operationCounts[operation]
}

// GitHubClient インターフェースの実装
func (m *statelessMockClient) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	return &github.Repository{
		Name:  github.String(repo),
		Owner: &github.User{Login: github.String(owner)},
	}, nil
}

func (m *statelessMockClient) ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []*github.Issue
	for _, issue := range m.issues {
		for _, label := range labels {
			if hasLabel(issue, label) {
				result = append(result, issue)
				break
			}
		}
	}
	return result, nil
}

func (m *statelessMockClient) RemoveLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.operationCounts["RemoveLabel"]++

	if m.currentErrors < m.errorCount {
		m.currentErrors++
		return &github.GitHubError{
			Type:    github.ErrorTypeRateLimit,
			Message: "API rate limit exceeded",
		}
	}

	m.labelChanges = append(m.labelChanges, labelChange{
		issueNumber: issueNumber,
		operation:   "remove",
		label:       label,
	})

	// 実際にラベルを削除
	if issue, exists := m.issues[issueNumber]; exists {
		newLabels := []*github.Label{}
		for _, l := range issue.Labels {
			if l.Name != nil && *l.Name != label {
				newLabels = append(newLabels, l)
			}
		}
		issue.Labels = newLabels
	}

	return nil
}

func (m *statelessMockClient) AddLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.operationCounts["AddLabel"]++

	if m.currentErrors < m.errorCount {
		m.currentErrors++
		return &github.GitHubError{
			Type:    github.ErrorTypeRateLimit,
			Message: "API rate limit exceeded",
		}
	}

	m.labelChanges = append(m.labelChanges, labelChange{
		issueNumber: issueNumber,
		operation:   "add",
		label:       label,
	})

	// 実際にラベルを追加
	if issue, exists := m.issues[issueNumber]; exists {
		issue.Labels = append(issue.Labels, &github.Label{
			Name: github.String(label),
		})
	}

	return nil
}

func (m *statelessMockClient) GetRateLimit(ctx context.Context) (*github.RateLimits, error) {
	return &github.RateLimits{
		Core: &github.RateLimit{
			Limit:     5000,
			Remaining: 4999,
		},
	}, nil
}

func (m *statelessMockClient) TransitionIssueLabel(ctx context.Context, owner, repo string, issueNumber int) (bool, error) {
	return false, nil
}

func (m *statelessMockClient) TransitionIssueLabelWithInfo(ctx context.Context, owner, repo string, issueNumber int) (bool, *github.TransitionInfo, error) {
	return false, nil, nil
}

func (m *statelessMockClient) EnsureLabelsExist(ctx context.Context, owner, repo string) error {
	return nil
}

func (m *statelessMockClient) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, comment string) error {
	return nil
}

// ヘルパー関数
func createMockAction(name string, triggerLabel string, executedActions *[]string, mu *sync.Mutex) ActionExecutor {
	return &mockAction{
		canExecute: func(issue *github.Issue) bool {
			return hasLabel(issue, triggerLabel)
		},
		execute: func(ctx context.Context, issue *github.Issue) error {
			mu.Lock()
			*executedActions = append(*executedActions, name)
			mu.Unlock()
			return nil
		},
	}
}

package watcher

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIssueStateManager(t *testing.T) {
	t.Run("新しいStateManagerの作成", func(t *testing.T) {
		// Act
		manager := NewIssueStateManager()

		// Assert
		assert.NotNil(t, manager)
	})
}

func TestGetState(t *testing.T) {
	t.Run("存在しないIssueの状態を取得", func(t *testing.T) {
		// Arrange
		manager := NewIssueStateManager()

		// Act
		state, exists := manager.GetState(123)

		// Assert
		assert.False(t, exists)
		assert.Nil(t, state)
	})

	t.Run("存在するIssueの状態を取得", func(t *testing.T) {
		// Arrange
		manager := NewIssueStateManager()
		manager.SetState(123, IssueStatePlan, IssueStatusPending)

		// Act
		state, exists := manager.GetState(123)

		// Assert
		assert.True(t, exists)
		assert.NotNil(t, state)
		assert.Equal(t, int64(123), state.IssueNumber)
		assert.Equal(t, IssueStatePlan, state.Phase)
		assert.Equal(t, IssueStatusPending, state.Status)
	})
}

func TestSetState(t *testing.T) {
	t.Run("新しい状態の設定", func(t *testing.T) {
		// Arrange
		manager := NewIssueStateManager()

		// Act
		manager.SetState(123, IssueStatePlan, IssueStatusProcessing)

		// Assert
		state, exists := manager.GetState(123)
		assert.True(t, exists)
		assert.Equal(t, IssueStatePlan, state.Phase)
		assert.Equal(t, IssueStatusProcessing, state.Status)
		assert.WithinDuration(t, time.Now(), state.LastAction, time.Second)
	})

	t.Run("既存の状態の更新", func(t *testing.T) {
		// Arrange
		manager := NewIssueStateManager()
		manager.SetState(123, IssueStatePlan, IssueStatusPending)
		time.Sleep(10 * time.Millisecond)

		// Act
		manager.SetState(123, IssueStateImplementation, IssueStatusProcessing)

		// Assert
		state, exists := manager.GetState(123)
		assert.True(t, exists)
		assert.Equal(t, IssueStateImplementation, state.Phase)
		assert.Equal(t, IssueStatusProcessing, state.Status)
	})
}

func TestIsProcessing(t *testing.T) {
	t.Run("処理中の状態", func(t *testing.T) {
		// Arrange
		manager := NewIssueStateManager()
		manager.SetState(123, IssueStatePlan, IssueStatusProcessing)

		// Act
		isProcessing := manager.IsProcessing(123)

		// Assert
		assert.True(t, isProcessing)
	})

	t.Run("処理中でない状態", func(t *testing.T) {
		// Arrange
		manager := NewIssueStateManager()
		manager.SetState(123, IssueStatePlan, IssueStatusPending)

		// Act
		isProcessing := manager.IsProcessing(123)

		// Assert
		assert.False(t, isProcessing)
	})

	t.Run("存在しないIssue", func(t *testing.T) {
		// Arrange
		manager := NewIssueStateManager()

		// Act
		isProcessing := manager.IsProcessing(999)

		// Assert
		assert.False(t, isProcessing)
	})
}

func TestHasBeenProcessed(t *testing.T) {
	t.Run("処理済みの状態", func(t *testing.T) {
		// Arrange
		manager := NewIssueStateManager()
		manager.SetState(123, IssueStatePlan, IssueStatusCompleted)

		// Act
		hasBeenProcessed := manager.HasBeenProcessed(123, IssueStatePlan)

		// Assert
		assert.True(t, hasBeenProcessed)
	})

	t.Run("処理済みでない状態", func(t *testing.T) {
		// Arrange
		manager := NewIssueStateManager()
		manager.SetState(123, IssueStatePlan, IssueStatusPending)

		// Act
		hasBeenProcessed := manager.HasBeenProcessed(123, IssueStatePlan)

		// Assert
		assert.False(t, hasBeenProcessed)
	})

	t.Run("異なるフェーズ", func(t *testing.T) {
		// Arrange
		manager := NewIssueStateManager()
		manager.SetState(123, IssueStatePlan, IssueStatusCompleted)

		// Act
		hasBeenProcessed := manager.HasBeenProcessed(123, IssueStateImplementation)

		// Assert
		assert.False(t, hasBeenProcessed)
	})

	t.Run("存在しないIssue", func(t *testing.T) {
		// Arrange
		manager := NewIssueStateManager()

		// Act
		hasBeenProcessed := manager.HasBeenProcessed(999, IssueStatePlan)

		// Assert
		assert.False(t, hasBeenProcessed)
	})
}

func TestMarkAsCompleted(t *testing.T) {
	t.Run("完了状態への遷移", func(t *testing.T) {
		// Arrange
		manager := NewIssueStateManager()
		manager.SetState(123, IssueStatePlan, IssueStatusProcessing)

		// Act
		manager.MarkAsCompleted(123, IssueStatePlan)

		// Assert
		state, exists := manager.GetState(123)
		assert.True(t, exists)
		assert.Equal(t, IssueStatusCompleted, state.Status)
	})
}

func TestMarkAsFailed(t *testing.T) {
	t.Run("失敗状態への遷移", func(t *testing.T) {
		// Arrange
		manager := NewIssueStateManager()
		manager.SetState(123, IssueStatePlan, IssueStatusProcessing)

		// Act
		manager.MarkAsFailed(123, IssueStatePlan)

		// Assert
		state, exists := manager.GetState(123)
		assert.True(t, exists)
		assert.Equal(t, IssueStatusFailed, state.Status)
	})
}

func TestConcurrentAccess(t *testing.T) {
	t.Run("並行アクセスの安全性", func(t *testing.T) {
		// Arrange
		manager := NewIssueStateManager()
		done := make(chan bool)

		// Act - 複数のgoroutineから同時にアクセス
		go func() {
			for i := 0; i < 100; i++ {
				manager.SetState(int64(i), IssueStatePlan, IssueStatusProcessing)
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 100; i++ {
				manager.GetState(int64(i))
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 100; i++ {
				manager.IsProcessing(int64(i))
			}
			done <- true
		}()

		// Assert - デッドロックやパニックが発生しないことを確認
		for i := 0; i < 3; i++ {
			select {
			case <-done:
				// 正常終了
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout: possible deadlock")
			}
		}
	})
}

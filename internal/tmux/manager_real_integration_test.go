//go:build integration
// +build integration

package tmux

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestTmuxManagerRealIntegration はtmuxコマンドとの実際の統合テスト
// 外部プロセス（tmux）との連携をテストし、内部コンポーネントは実際のものを使用
func TestTmuxManagerRealIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// tmuxコマンドが利用可能かチェック
	if err := exec.Command("tmux", "list-sessions").Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// tmuxサーバーが起動していない場合は正常（セッションがない）
		} else {
			t.Skip("tmux command not available, skipping tmux integration test")
		}
	}

	// 安全性チェック：本番セッションの存在確認
	if err := SafetyCheckBeforeTests(); err != nil {
		t.Logf("Safety check warning: %v", err)
	}

	// テスト用のセッション名（test-osoba-プレフィックスを使用）
	testSessionName := "test-osoba-session-" + time.Now().Format("20060102-150405")

	// クリーンアップ関数
	cleanup := func() {
		// テストセッションが存在する場合は削除
		if err := exec.Command("tmux", "kill-session", "-t", testSessionName).Run(); err != nil {
			// セッションが存在しない場合はエラーを無視
		}
	}
	defer cleanup()

	t.Run("tmuxマネージャーとの実際の連携", func(t *testing.T) {
		// 実際のコマンド実行を使用するマネージャーを作成
		manager := NewDefaultManager()

		t.Run("セッション作成", func(t *testing.T) {
			err := manager.CreateSession(testSessionName)
			assert.NoError(t, err)

			// セッションが実際に作成されたことを確認
			output, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
			assert.NoError(t, err)

			sessions := strings.Split(strings.TrimSpace(string(output)), "\n")
			assert.Contains(t, sessions, testSessionName)
		})

		t.Run("セッション存在確認", func(t *testing.T) {
			exists, err := manager.SessionExists(testSessionName)
			assert.NoError(t, err)
			assert.True(t, exists)

			// 存在しないセッションのテスト
			exists, err = manager.SessionExists("non-existent-session-12345")
			assert.NoError(t, err)
			assert.False(t, exists)
		})

		t.Run("ウィンドウ作成", func(t *testing.T) {
			testWindowName := "test-window"
			err := manager.CreateWindow(testSessionName, testWindowName)
			assert.NoError(t, err)

			// ウィンドウが実際に作成されたことを確認
			output, err := exec.Command("tmux", "list-windows", "-t", testSessionName, "-F", "#{window_name}").Output()
			assert.NoError(t, err)

			windows := strings.Split(strings.TrimSpace(string(output)), "\n")
			assert.Contains(t, windows, testWindowName)
		})

		t.Run("セッション一覧取得", func(t *testing.T) {
			sessions, err := manager.ListSessions("test-osoba")
			assert.NoError(t, err)
			assert.NotNil(t, sessions)

			// テストセッションが含まれていることを確認
			found := false
			for _, session := range sessions {
				if session == testSessionName {
					found = true
					break
				}
			}
			assert.True(t, found, "Test session should be in the list")
		})

		t.Run("セッション削除", func(t *testing.T) {
			// DefaultManagerにはKillSessionがないため、手動削除
			err := exec.Command("tmux", "kill-session", "-t", testSessionName).Run()
			assert.NoError(t, err)

			// セッションが実際に削除されたことを確認
			exists, err := manager.SessionExists(testSessionName)
			assert.NoError(t, err)
			assert.False(t, exists)
		})
	})
}

// TestTmuxManagerErrorHandling はエラーハンドリングの統合テスト
func TestTmuxManagerErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// tmuxコマンドが利用可能かチェック
	if err := exec.Command("tmux", "list-sessions").Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// tmuxサーバーが起動していない場合は正常（セッションがない）
		} else {
			t.Skip("tmux command not available")
		}
	}

	// 安全性チェック：本番セッションの存在確認
	if err := SafetyCheckBeforeTests(); err != nil {
		t.Logf("Safety check warning: %v", err)
	}

	manager := NewDefaultManager()

	t.Run("存在しないセッションでのエラーハンドリング", func(t *testing.T) {
		nonExistentSession := "non-existent-session-12345"

		// 存在しないセッションでのウィンドウ作成
		err := manager.CreateWindow(nonExistentSession, "test-window")
		assert.Error(t, err)
		t.Logf("Expected error for non-existent session: %v", err)
	})

	t.Run("重複セッション作成でのエラーハンドリング", func(t *testing.T) {
		testSessionName := "test-osoba-duplicate-" + time.Now().Format("20060102-150405")

		// クリーンアップ
		defer func() {
			exec.Command("tmux", "kill-session", "-t", testSessionName).Run()
		}()

		// 最初のセッション作成
		err := manager.CreateSession(testSessionName)
		assert.NoError(t, err)

		// 同名セッションの重複作成
		err = manager.CreateSession(testSessionName)
		assert.Error(t, err)
		t.Logf("Expected error for duplicate session: %v", err)
	})
}

// TestTmuxManagerConcurrentAccess は並行アクセスでの統合テスト
func TestTmuxManagerConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// tmuxコマンドが利用可能かチェック
	if err := exec.Command("tmux", "list-sessions").Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// tmuxサーバーが起動していない場合は正常（セッションがない）
		} else {
			t.Skip("tmux command not available")
		}
	}

	// 安全性チェック：本番セッションの存在確認
	if err := SafetyCheckBeforeTests(); err != nil {
		t.Logf("Safety check warning: %v", err)
	}

	manager := NewDefaultManager()

	t.Run("複数セッションの同時作成", func(t *testing.T) {
		const numSessions = 3
		sessionNames := make([]string, numSessions)
		errors := make(chan error, numSessions)

		// クリーンアップ
		defer func() {
			for _, sessionName := range sessionNames {
				if sessionName != "" {
					exec.Command("tmux", "kill-session", "-t", sessionName).Run()
				}
			}
		}()

		// 複数のgoroutineでセッションを作成
		for i := 0; i < numSessions; i++ {
			sessionNames[i] = "test-osoba-concurrent-" + time.Now().Format("20060102-150405") + "-" + string(rune('a'+i))

			go func(sessionName string) {
				err := manager.CreateSession(sessionName)
				errors <- err
			}(sessionNames[i])
		}

		// 結果を収集
		successCount := 0
		for i := 0; i < numSessions; i++ {
			err := <-errors
			if err == nil {
				successCount++
			} else {
				t.Logf("Session creation error: %v", err)
			}
		}

		// 全て成功することを期待
		assert.Equal(t, numSessions, successCount, "All concurrent session creations should succeed")

		// セッションが実際に作成されたことを確認
		sessions, err := manager.ListSessions("test-osoba")
		assert.NoError(t, err)

		for _, sessionName := range sessionNames {
			found := false
			for _, session := range sessions {
				if session == sessionName {
					found = true
					break
				}
			}
			assert.True(t, found, "Session %s should exist", sessionName)
		}
	})
}

// TestTmuxManagerPerformance はパフォーマンスの統合テスト
func TestTmuxManagerPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// tmuxコマンドが利用可能かチェック
	if err := exec.Command("tmux", "list-sessions").Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// tmuxサーバーが起動していない場合は正常（セッションがない）
		} else {
			t.Skip("tmux command not available")
		}
	}

	// 安全性チェック：本番セッションの存在確認
	if err := SafetyCheckBeforeTests(); err != nil {
		t.Logf("Safety check warning: %v", err)
	}

	manager := NewDefaultManager()

	t.Run("セッション作成のレスポンス時間", func(t *testing.T) {
		testSessionName := "test-osoba-perf-" + time.Now().Format("20060102-150405")

		defer func() {
			exec.Command("tmux", "kill-session", "-t", testSessionName).Run()
		}()

		start := time.Now()
		err := manager.CreateSession(testSessionName)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.Less(t, duration, 2*time.Second, "Session creation should be within 2 seconds")

		t.Logf("Session creation time: %v", duration)
	})

	t.Run("セッション一覧取得のレスポンス時間", func(t *testing.T) {
		start := time.Now()
		sessions, err := manager.ListSessions("test-osoba")
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.NotNil(t, sessions)
		assert.Less(t, duration, 1*time.Second, "Session listing should be within 1 second")

		t.Logf("Session listing time: %v (found %d sessions)", duration, len(sessions))
	})
}

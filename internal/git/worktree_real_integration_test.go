//go:build integration
// +build integration

package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGitWorktreeRealIntegration はgit worktreeコマンドとの実際の統合テスト
// 外部プロセス（git）との連携をテストし、内部コンポーネントは実際のものを使用
func TestGitWorktreeRealIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// gitコマンドが利用可能かチェック
	if err := exec.Command("git", "--version").Run(); err != nil {
		t.Skip("git command not available, skipping git worktree integration test")
	}

	// テスト用のリポジトリディレクトリを作成
	testDir := t.TempDir()
	repoDir := filepath.Join(testDir, "test-repo")

	// テスト用のGitリポジトリを初期化
	err := exec.Command("git", "init", repoDir).Run()
	require.NoError(t, err)

	// 初期コミットを作成
	err = os.Chdir(repoDir)
	require.NoError(t, err)

	// git config設定
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	// 初期ファイル作成とコミット
	err = os.WriteFile("README.md", []byte("# Test Repository"), 0644)
	require.NoError(t, err)

	err = exec.Command("git", "add", "README.md").Run()
	require.NoError(t, err)

	err = exec.Command("git", "commit", "-m", "Initial commit").Run()
	require.NoError(t, err)

	t.Run("git worktreeマネージャーとの実際の連携", func(t *testing.T) {
		// 実際のコマンド実行を使用するWorktreeを作成
		log, err := logger.New(logger.WithLevel("error"))
		require.NoError(t, err)

		worktree := NewWorktree(log)

		testBranchName := "feature/test-branch-" + time.Now().Format("20060102-150405")
		worktreePath := filepath.Join(testDir, "worktree-"+testBranchName)

		// クリーンアップ
		defer func() {
			// worktreeを削除
			exec.Command("git", "worktree", "remove", "--force", worktreePath).Run()
			// ブランチを削除
			exec.Command("git", "branch", "-D", testBranchName).Run()
		}()

		ctx := context.Background()

		t.Run("worktreeの作成", func(t *testing.T) {
			err := worktree.Create(ctx, repoDir, worktreePath, testBranchName)
			assert.NoError(t, err)

			// worktreeが実際に作成されたことを確認
			assert.DirExists(t, worktreePath)

			// ブランチが作成されたことを確認
			output, err := exec.Command("git", "branch", "--list", testBranchName).Output()
			assert.NoError(t, err)
			assert.Contains(t, string(output), testBranchName)
		})

		t.Run("worktree一覧の取得", func(t *testing.T) {
			worktrees, err := worktree.List(ctx, repoDir)
			assert.NoError(t, err)
			assert.NotEmpty(t, worktrees)

			// テスト用worktreeが含まれていることを確認
			found := false
			for _, wt := range worktrees {
				if strings.Contains(wt.Path, testBranchName) {
					found = true
					assert.Equal(t, testBranchName, wt.Branch)
					break
				}
			}
			assert.True(t, found, "Test worktree should be in the list")
		})

		t.Run("worktreeの存在確認", func(t *testing.T) {
			// 一覧から存在確認
			worktrees, err := worktree.List(ctx, repoDir)
			assert.NoError(t, err)

			found := false
			for _, wt := range worktrees {
				if strings.Contains(wt.Path, testBranchName) {
					found = true
					break
				}
			}
			assert.True(t, found, "Test worktree should exist")

			// ディレクトリの存在確認
			assert.DirExists(t, worktreePath)
		})

		t.Run("worktreeでの変更とコミット", func(t *testing.T) {
			// worktreeディレクトリに移動
			originalDir, _ := os.Getwd()
			defer os.Chdir(originalDir)

			err := os.Chdir(worktreePath)
			require.NoError(t, err)

			// ファイルを作成
			testFile := "test-file.txt"
			err = os.WriteFile(testFile, []byte("Test content"), 0644)
			require.NoError(t, err)

			// ファイルをステージング
			err = exec.Command("git", "add", testFile).Run()
			assert.NoError(t, err)

			// コミット
			err = exec.Command("git", "commit", "-m", "Add test file").Run()
			assert.NoError(t, err)

			// コミットが作成されたことを確認
			output, err := exec.Command("git", "log", "--oneline", "-1").Output()
			assert.NoError(t, err)
			assert.Contains(t, string(output), "Add test file")
		})

		t.Run("worktreeの削除", func(t *testing.T) {
			err := worktree.Remove(ctx, repoDir, worktreePath)
			assert.NoError(t, err)

			// worktreeディレクトリが削除されたことを確認
			assert.NoDirExists(t, worktreePath)

			// worktreeが一覧から削除されたことを確認
			worktrees, err := worktree.List(ctx, repoDir)
			assert.NoError(t, err)

			found := false
			for _, wt := range worktrees {
				if strings.Contains(wt.Path, testBranchName) {
					found = true
					break
				}
			}
			assert.False(t, found, "Test worktree should be removed from the list")
		})
	})
}

// TestGitWorktreeErrorHandling はエラーハンドリングの統合テスト
func TestGitWorktreeErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// gitコマンドが利用可能かチェック
	if err := exec.Command("git", "--version").Run(); err != nil {
		t.Skip("git command not available")
	}

	// テスト用のリポジトリディレクトリを作成
	testDir := t.TempDir()
	repoDir := filepath.Join(testDir, "test-repo")

	// テスト用のGitリポジトリを初期化
	err := exec.Command("git", "init", repoDir).Run()
	require.NoError(t, err)

	err = os.Chdir(repoDir)
	require.NoError(t, err)

	// git config設定
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	// 初期コミットを作成
	err = os.WriteFile("README.md", []byte("# Test Repository"), 0644)
	require.NoError(t, err)
	exec.Command("git", "add", "README.md").Run()
	exec.Command("git", "commit", "-m", "Initial commit").Run()

	log, err := logger.New(logger.WithLevel("error"))
	require.NoError(t, err)
	worktree := NewWorktree(log)
	ctx := context.Background()

	t.Run("存在しないリポジトリでのエラーハンドリング", func(t *testing.T) {
		nonExistentRepo := "/non/existent/repo"

		err := worktree.Create(ctx, nonExistentRepo, "/tmp/test-worktree", "test-branch")
		assert.Error(t, err)
		t.Logf("Expected error for non-existent repo: %v", err)
	})

	t.Run("重複ブランチでのエラーハンドリング", func(t *testing.T) {
		testBranchName := "duplicate-branch-test"
		worktreePath1 := filepath.Join(testDir, "worktree1")
		worktreePath2 := filepath.Join(testDir, "worktree2")

		defer func() {
			exec.Command("git", "worktree", "remove", "--force", worktreePath1).Run()
			exec.Command("git", "worktree", "remove", "--force", worktreePath2).Run()
			exec.Command("git", "branch", "-D", testBranchName).Run()
		}()

		// 最初のworktree作成
		err := worktree.Create(ctx, repoDir, worktreePath1, testBranchName)
		assert.NoError(t, err)

		// 同名ブランチでの重複作成
		err = worktree.Create(ctx, repoDir, worktreePath2, testBranchName)
		assert.Error(t, err)
		t.Logf("Expected error for duplicate branch: %v", err)
	})

	t.Run("無効なパスでのエラーハンドリング", func(t *testing.T) {
		invalidPath := "/root/cannot-write-here"

		err := worktree.Create(ctx, repoDir, invalidPath, "test-invalid-path")
		assert.Error(t, err)
		t.Logf("Expected error for invalid path: %v", err)
	})
}

// TestGitWorktreeConcurrentAccess は並行アクセスでの統合テスト
func TestGitWorktreeConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// gitコマンドが利用可能かチェック
	if err := exec.Command("git", "--version").Run(); err != nil {
		t.Skip("git command not available")
	}

	// テスト用のリポジトリディレクトリを作成
	testDir := t.TempDir()
	repoDir := filepath.Join(testDir, "test-repo")

	// テスト用のGitリポジトリを初期化
	err := exec.Command("git", "init", repoDir).Run()
	require.NoError(t, err)

	err = os.Chdir(repoDir)
	require.NoError(t, err)

	// git config設定
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	// 初期コミットを作成
	err = os.WriteFile("README.md", []byte("# Test Repository"), 0644)
	require.NoError(t, err)
	exec.Command("git", "add", "README.md").Run()
	exec.Command("git", "commit", "-m", "Initial commit").Run()

	log, err := logger.New(logger.WithLevel("error"))
	require.NoError(t, err)
	worktree := NewWorktree(log)
	ctx := context.Background()

	t.Run("複数worktreeの同時作成", func(t *testing.T) {
		const numWorktrees = 3
		branchNames := make([]string, numWorktrees)
		worktreePaths := make([]string, numWorktrees)
		errors := make(chan error, numWorktrees)

		// クリーンアップ
		defer func() {
			for i := 0; i < numWorktrees; i++ {
				if worktreePaths[i] != "" {
					exec.Command("git", "worktree", "remove", "--force", worktreePaths[i]).Run()
					exec.Command("git", "branch", "-D", branchNames[i]).Run()
				}
			}
		}()

		// 複数のgoroutineでworktreeを作成
		for i := 0; i < numWorktrees; i++ {
			branchNames[i] = "concurrent-test-" + time.Now().Format("20060102-150405") + "-" + string(rune('a'+i))
			worktreePaths[i] = filepath.Join(testDir, "worktree-"+branchNames[i])

			go func(branchName, worktreePath string) {
				err := worktree.Create(ctx, repoDir, worktreePath, branchName)
				errors <- err
			}(branchNames[i], worktreePaths[i])
		}

		// 結果を収集
		successCount := 0
		for i := 0; i < numWorktrees; i++ {
			err := <-errors
			if err == nil {
				successCount++
			} else {
				t.Logf("Worktree creation error: %v", err)
			}
		}

		// 全て成功することを期待
		assert.Equal(t, numWorktrees, successCount, "All concurrent worktree creations should succeed")

		// worktreeが実際に作成されたことを確認
		worktrees, err := worktree.List(ctx, repoDir)
		assert.NoError(t, err)

		for _, branchName := range branchNames {
			found := false
			for _, wt := range worktrees {
				if wt.Branch == branchName {
					found = true
					break
				}
			}
			assert.True(t, found, "Worktree for branch %s should exist", branchName)
		}
	})
}

// TestGitWorktreePerformance はパフォーマンスの統合テスト
func TestGitWorktreePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// gitコマンドが利用可能かチェック
	if err := exec.Command("git", "--version").Run(); err != nil {
		t.Skip("git command not available")
	}

	// テスト用のリポジトリディレクトリを作成
	testDir := t.TempDir()
	repoDir := filepath.Join(testDir, "test-repo")

	// テスト用のGitリポジトリを初期化
	err := exec.Command("git", "init", repoDir).Run()
	require.NoError(t, err)

	err = os.Chdir(repoDir)
	require.NoError(t, err)

	// git config設定
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	// 初期コミットを作成
	err = os.WriteFile("README.md", []byte("# Test Repository"), 0644)
	require.NoError(t, err)
	exec.Command("git", "add", "README.md").Run()
	exec.Command("git", "commit", "-m", "Initial commit").Run()

	log, err := logger.New(logger.WithLevel("error"))
	require.NoError(t, err)
	worktree := NewWorktree(log)
	ctx := context.Background()

	t.Run("worktree作成のレスポンス時間", func(t *testing.T) {
		testBranchName := "perf-test-" + time.Now().Format("20060102-150405")
		worktreePath := filepath.Join(testDir, "perf-worktree")

		defer func() {
			exec.Command("git", "worktree", "remove", "--force", worktreePath).Run()
			exec.Command("git", "branch", "-D", testBranchName).Run()
		}()

		start := time.Now()
		err := worktree.Create(ctx, repoDir, worktreePath, testBranchName)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.Less(t, duration, 3*time.Second, "Worktree creation should be within 3 seconds")

		t.Logf("Worktree creation time: %v", duration)
	})

	t.Run("worktree一覧取得のレスポンス時間", func(t *testing.T) {
		start := time.Now()
		worktrees, err := worktree.List(ctx, repoDir)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.NotNil(t, worktrees)
		assert.Less(t, duration, 1*time.Second, "Worktree listing should be within 1 second")

		t.Logf("Worktree listing time: %v (found %d worktrees)", duration, len(worktrees))
	})
}

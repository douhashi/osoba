//go:build !windows
// +build !windows

package daemon

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestDaemonManager_IsRunning(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		pidFile  string
		setup    func() string
		expected bool
	}{
		{
			name:    "running process",
			pidFile: filepath.Join(tmpDir, "running.pid"),
			setup: func() string {
				// 現在のプロセスのPIDを使用
				pid := os.Getpid()
				content := []byte(strconv.Itoa(pid) + "\n" + time.Now().Format(time.RFC3339) + "\n/path/to/repo")
				path := filepath.Join(tmpDir, "running.pid")
				os.WriteFile(path, content, 0600)
				return path
			},
			expected: true,
		},
		{
			name:    "dead process",
			pidFile: filepath.Join(tmpDir, "dead.pid"),
			setup: func() string {
				// 存在しないPIDを使用
				content := []byte("99999\n" + time.Now().Format(time.RFC3339) + "\n/path/to/repo")
				path := filepath.Join(tmpDir, "dead.pid")
				os.WriteFile(path, content, 0600)
				return path
			},
			expected: false,
		},
		{
			name:     "non-existent pid file",
			pidFile:  filepath.Join(tmpDir, "nonexistent.pid"),
			setup:    func() string { return filepath.Join(tmpDir, "nonexistent.pid") },
			expected: false,
		},
	}

	dm := NewDaemonManager()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pidFile := tt.setup()
			if got := dm.IsRunning(pidFile); got != tt.expected {
				t.Errorf("IsRunning() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDaemonManager_Status(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	// 現在のプロセスのPIDを使用してテスト
	pid := os.Getpid()
	now := time.Now()
	content := []byte(strconv.Itoa(pid) + "\n" + now.Format(time.RFC3339) + "\n/path/to/repo")
	if err := os.WriteFile(pidFile, content, 0600); err != nil {
		t.Fatalf("Failed to create test PID file: %v", err)
	}

	dm := NewDaemonManager()
	status, err := dm.Status(pidFile)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if status.PID != pid {
		t.Errorf("Status.PID = %d, want %d", status.PID, pid)
	}
	if !status.Running {
		t.Error("Status.Running = false, want true")
	}
	if status.RepoPath != "/path/to/repo" {
		t.Errorf("Status.RepoPath = %s, want /path/to/repo", status.RepoPath)
	}
}

func TestDaemonManager_CleanupStalePIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "stale.pid")

	// 存在しないPIDでファイルを作成
	content := []byte("99999\n" + time.Now().Format(time.RFC3339) + "\n/path/to/repo")
	if err := os.WriteFile(pidFile, content, 0600); err != nil {
		t.Fatalf("Failed to create test PID file: %v", err)
	}

	err := cleanupStalePIDFile(pidFile)
	if err != nil {
		t.Errorf("cleanupStalePIDFile() error = %v", err)
	}

	// ファイルが削除されたことを確認
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Error("Stale PID file was not removed")
	}
}

func TestProcessInfo_IsRunning(t *testing.T) {
	tests := []struct {
		name     string
		pid      int
		expected bool
	}{
		{
			name:     "current process",
			pid:      os.Getpid(),
			expected: true,
		},
		{
			name:     "non-existent process",
			pid:      99999,
			expected: false,
		},
		{
			name:     "invalid pid",
			pid:      -1,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &ProcessInfo{PID: tt.pid}
			if got := info.isRunning(); got != tt.expected {
				t.Errorf("isRunning() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDaemonManager_Start(t *testing.T) {
	// Start メソッドのテストは統合テストで行う
	// ここでは基本的な構造のみ確認
	dm := NewDaemonManager()
	ctx := context.Background()

	// テストモードを設定
	os.Setenv("GO_TEST", "1")
	defer os.Unsetenv("GO_TEST")

	// 環境変数が設定されていない場合のテスト
	if os.Getenv("OSOBA_DAEMON_MODE") == "" {
		// 実際の再起動は行わないが、メソッドが存在することを確認
		err := dm.Start(ctx, []string{"test"})
		if err != nil {
			t.Errorf("Start() error = %v", err)
		}
	}
}

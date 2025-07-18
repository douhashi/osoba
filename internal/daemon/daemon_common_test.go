package daemon

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestNewDaemonManager(t *testing.T) {
	dm := NewDaemonManager()
	if dm == nil {
		t.Error("NewDaemonManager() returned nil")
	}
}

func TestDaemonManager_WritePIDFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping PID file test on Windows")
	}

	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	info := &ProcessInfo{
		PID:       12345,
		StartTime: time.Now(),
		RepoPath:  "/path/to/repo",
	}

	err := WritePIDFile(pidFile, info)
	if err != nil {
		t.Fatalf("writePIDFile() error = %v", err)
	}

	// ファイルの存在確認
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		t.Error("PID file was not created")
	}

	// 権限の確認
	fileInfo, err := os.Stat(pidFile)
	if err != nil {
		t.Fatalf("Failed to stat PID file: %v", err)
	}
	if fileInfo.Mode().Perm() != 0600 {
		t.Errorf("PID file permissions = %v, want 0600", fileInfo.Mode().Perm())
	}
}

func TestDaemonManager_ReadPIDFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping PID file test on Windows")
	}

	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	// テスト用のPIDファイルを作成
	now := time.Now()
	content := []byte(`12345
` + now.Format(time.RFC3339) + `
/path/to/repo`)
	if err := os.WriteFile(pidFile, content, 0600); err != nil {
		t.Fatalf("Failed to create test PID file: %v", err)
	}

	info, err := readPIDFile(pidFile)
	if err != nil {
		t.Fatalf("readPIDFile() error = %v", err)
	}

	if info.PID != 12345 {
		t.Errorf("PID = %d, want 12345", info.PID)
	}
	if info.RepoPath != "/path/to/repo" {
		t.Errorf("RepoPath = %s, want /path/to/repo", info.RepoPath)
	}
}

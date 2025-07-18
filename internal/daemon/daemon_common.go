package daemon

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// DaemonManager はバックグラウンドプロセスを管理するインターフェース
type DaemonManager interface {
	Start(ctx context.Context, args []string) error
	Stop(pidFile string) error
	Status(pidFile string) (*ProcessStatus, error)
	IsRunning(pidFile string) bool
}

// ProcessStatus はプロセスの状態を表す構造体
type ProcessStatus struct {
	PID       int
	StartTime time.Time
	RepoPath  string
	Running   bool
}

// ProcessInfo はPIDファイルに保存する情報
type ProcessInfo struct {
	PID       int
	StartTime time.Time
	RepoPath  string
}

// WritePIDFile は外部から呼び出し可能なPIDファイル作成関数
func WritePIDFile(pidFile string, info *ProcessInfo) error {
	content := fmt.Sprintf("%d\n%s\n%s",
		info.PID,
		info.StartTime.Format(time.RFC3339),
		info.RepoPath)

	if err := os.WriteFile(pidFile, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	return nil
}

// readPIDFile はPIDファイルを読み込みます
func readPIDFile(pidFile string) (*ProcessInfo, error) {
	content, err := os.ReadFile(pidFile)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) < 3 {
		return nil, fmt.Errorf("invalid PID file format")
	}

	pid, err := strconv.Atoi(lines[0])
	if err != nil {
		return nil, fmt.Errorf("invalid PID: %w", err)
	}

	startTime, err := time.Parse(time.RFC3339, lines[1])
	if err != nil {
		return nil, fmt.Errorf("invalid start time: %w", err)
	}

	return &ProcessInfo{
		PID:       pid,
		StartTime: startTime,
		RepoPath:  lines[2],
	}, nil
}

// cleanupStalePIDFile は古いPIDファイルを削除します
func cleanupStalePIDFile(pidFile string) error {
	return os.Remove(pidFile)
}

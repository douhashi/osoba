//go:build windows
// +build windows

package daemon

import (
	"context"
	"fmt"
	"os"
	"runtime"
)

type daemonManager struct{}

// NewDaemonManager は新しいDaemonManagerを作成します
func NewDaemonManager() DaemonManager {
	return &daemonManager{}
}

// Start はプロセスをバックグラウンドで起動します（Windows未対応）
func (dm *daemonManager) Start(ctx context.Context, args []string) error {
	return fmt.Errorf("バックグラウンド実行はWindows環境では未対応です。--foreground フラグを使用してください")
}

// Stop はプロセスを停止します（Windows未対応）
func (dm *daemonManager) Stop(pidFile string) error {
	return fmt.Errorf("プロセス停止機能はWindows環境では未対応です")
}

// Status はプロセスの状態を取得します
func (dm *daemonManager) Status(pidFile string) (*ProcessStatus, error) {
	// Windows環境では常に実行されていない状態を返す
	return &ProcessStatus{Running: false}, nil
}

// IsRunning はプロセスが実行中かどうかを確認します
func (dm *daemonManager) IsRunning(pidFile string) bool {
	// Windows環境では常にfalseを返す
	return false
}

// isRunning はプロセスが実行中かどうかを確認します（Windows用スタブ）
func (info *ProcessInfo) isRunning() bool {
	// Windows環境では常にfalseを返す
	return false
}

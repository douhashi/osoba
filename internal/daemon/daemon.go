//go:build !windows
// +build !windows

package daemon

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type daemonManager struct{}

// NewDaemonManager は新しいDaemonManagerを作成します
func NewDaemonManager() DaemonManager {
	return &daemonManager{}
}

// Start はプロセスをバックグラウンドで起動します
func (dm *daemonManager) Start(ctx context.Context, args []string) error {
	// 環境変数でデーモンモードかどうかを判定
	if os.Getenv("OSOBA_DAEMON_MODE") == "1" {
		// すでにデーモンモードで起動している
		return nil
	}

	// 自分自身をバックグラウンドで再起動
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(), "OSOBA_DAEMON_MODE=1")

	// 標準入出力をリダイレクト
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// プロセスグループを分離
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// テスト中はos.Exitを呼ばない
	if os.Getenv("GO_TEST") != "1" {
		// 親プロセスは終了
		os.Exit(0)
	}
	return nil
}

// Stop はプロセスを停止します
func (dm *daemonManager) Stop(pidFile string) error {
	info, err := readPIDFile(pidFile)
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	// プロセスが実行中か確認
	if !info.isRunning() {
		// すでに停止している場合はPIDファイルを削除
		os.Remove(pidFile)
		return nil
	}

	// SIGTERMを送信
	proc, err := os.FindProcess(info.PID)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// プロセスの終了を待機（最大10秒）
	for i := 0; i < 100; i++ {
		if !info.isRunning() {
			os.Remove(pidFile)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	// タイムアウトした場合はSIGKILLを送信
	if err := proc.Signal(syscall.SIGKILL); err != nil {
		return fmt.Errorf("failed to send SIGKILL: %w", err)
	}

	os.Remove(pidFile)
	return nil
}

// Status はプロセスの状態を取得します
func (dm *daemonManager) Status(pidFile string) (*ProcessStatus, error) {
	info, err := readPIDFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &ProcessStatus{Running: false}, nil
		}
		return nil, err
	}

	status := &ProcessStatus{
		PID:       info.PID,
		StartTime: info.StartTime,
		RepoPath:  info.RepoPath,
		Running:   info.isRunning(),
	}

	// プロセスが死んでいる場合はPIDファイルをクリーンアップ
	if !status.Running {
		cleanupStalePIDFile(pidFile)
	}

	return status, nil
}

// IsRunning はプロセスが実行中かどうかを確認します
func (dm *daemonManager) IsRunning(pidFile string) bool {
	info, err := readPIDFile(pidFile)
	if err != nil {
		return false
	}
	return info.isRunning()
}

// isRunning はプロセスが実行中かどうかを確認します
func (info *ProcessInfo) isRunning() bool {
	if info.PID <= 0 {
		return false
	}

	proc, err := os.FindProcess(info.PID)
	if err != nil {
		return false
	}

	// プロセスの存在確認（UNIX系）
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

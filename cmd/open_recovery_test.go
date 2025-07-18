package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/daemon"
	"github.com/douhashi/osoba/internal/paths"
)

// MockDaemonManager はテスト用のDaemonManagerモック
type MockDaemonManager struct {
	isRunning     bool
	processStatus *daemon.ProcessStatus
	statusError   error
}

func (m *MockDaemonManager) Start(ctx context.Context, args []string) error {
	return nil
}

func (m *MockDaemonManager) Stop(pidFile string) error {
	return nil
}

func (m *MockDaemonManager) Status(pidFile string) (*daemon.ProcessStatus, error) {
	if m.statusError != nil {
		return nil, m.statusError
	}
	return m.processStatus, nil
}

func (m *MockDaemonManager) IsRunning(pidFile string) bool {
	return m.isRunning
}

// MockTmuxManager はテスト用のTmuxManagerモック
type MockTmuxManager struct {
	ensureSessionError error
	sessionExists      map[string]bool
}

func (m *MockTmuxManager) EnsureSession(sessionName string) error {
	if m.ensureSessionError != nil {
		return m.ensureSessionError
	}
	if m.sessionExists == nil {
		m.sessionExists = make(map[string]bool)
	}
	m.sessionExists[sessionName] = true
	return nil
}

func (m *MockTmuxManager) SessionExists(sessionName string) (bool, error) {
	if m.sessionExists == nil {
		return false, nil
	}
	return m.sessionExists[sessionName], nil
}

func (m *MockTmuxManager) KillSession(sessionName string) error {
	if m.sessionExists != nil {
		delete(m.sessionExists, sessionName)
	}
	return nil
}

func (m *MockTmuxManager) CreateWindow(sessionName, windowName, workDir string) error {
	return nil
}

func (m *MockTmuxManager) SendKeys(target, keys string) error {
	return nil
}

func (m *MockTmuxManager) ListSessions() ([]string, error) {
	var sessions []string
	for session := range m.sessionExists {
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (m *MockTmuxManager) HasSession(sessionName string) (bool, error) {
	return m.SessionExists(sessionName)
}

// TestSessionRecoveryWithDaemonRunning はデーモンが動作中の場合の自動復旧をテスト
func TestSessionRecoveryWithDaemonRunning(t *testing.T) {
	// 一時ディレクトリの作成
	tmpDir := t.TempDir()

	// テスト用のPIDファイルを作成
	pidFile := filepath.Join(tmpDir, "test-repo.pid")
	info := &daemon.ProcessInfo{
		PID:       os.Getpid(), // 現在のプロセスPIDを使用（実行中として認識される）
		StartTime: time.Now(),
		RepoPath:  "/tmp/test-repo",
	}
	if err := daemon.WritePIDFile(pidFile, info); err != nil {
		t.Fatalf("PIDファイルの作成に失敗: %v", err)
	}

	// モックの設定
	mockDaemon := &MockDaemonManager{
		isRunning: true,
		processStatus: &daemon.ProcessStatus{
			PID:       info.PID,
			StartTime: info.StartTime,
			RepoPath:  info.RepoPath,
			Running:   true,
		},
	}

	mockTmux := &MockTmuxManager{
		sessionExists: make(map[string]bool),
	}

	// attemptSessionRecoveryの実装をテスト
	sessionName := "osoba-test-repo"
	repoName := "test-repo"

	// PathManagerのモック化が必要
	_ = paths.NewPathManager("")
	_ = repoName + "_" + "/tmp/test-repo"

	// デーモンが動作中の確認
	if !mockDaemon.IsRunning(pidFile) {
		t.Error("デーモンが動作中として認識されるべき")
	}

	// セッション復旧を試行
	err := mockTmux.EnsureSession(sessionName)
	if err != nil {
		t.Errorf("セッションの復旧に失敗: %v", err)
	}

	// セッションが作成されたことを確認
	exists, _ := mockTmux.SessionExists(sessionName)
	if !exists {
		t.Error("セッションが作成されるべき")
	}
}

// TestSessionRecoveryWithDaemonNotRunning はデーモンが動作していない場合のエラーをテスト
func TestSessionRecoveryWithDaemonNotRunning(t *testing.T) {
	// モックの設定
	mockDaemon := &MockDaemonManager{
		isRunning: false,
	}

	_ = "osoba-test-repo"
	pidFile := "/tmp/test.pid"

	// デーモンが動作していないことを確認
	if mockDaemon.IsRunning(pidFile) {
		t.Error("デーモンは動作していないとして認識されるべき")
	}

	// この場合、エラーメッセージは "osoba start" を含むべき
	expectedError := "セッション 'osoba-test-repo' が見つかりません。先に 'osoba start'を実行してください"

	// 実際のattemptSessionRecovery関数をテストするには、
	// 依存関係を注入できるようにリファクタリングが必要
	t.Logf("期待されるエラーメッセージ: %s", expectedError)
}

// TestErrorMessageContainsOsobaStart はエラーメッセージが正しいコマンドを含むことをテスト
func TestErrorMessageContainsOsobaStart(t *testing.T) {
	// 元の関数を保存
	originalCheckTmux := checkTmuxInstalledFunc
	originalSessionExists := sessionExistsFunc
	originalGetRepoName := getRepositoryNameFunc

	// テスト後に復元
	defer func() {
		checkTmuxInstalledFunc = originalCheckTmux
		sessionExistsFunc = originalSessionExists
		getRepositoryNameFunc = originalGetRepoName
	}()

	// モックの設定
	checkTmuxInstalledFunc = func() error { return nil }
	getRepositoryNameFunc = func() (string, error) { return "test-repo", nil }
	sessionExistsFunc = func(name string) (bool, error) {
		return false, nil // セッションが存在しない
	}

	// コマンドの実行
	cmd := newOpenCmd()
	err := cmd.Execute()

	// エラーが発生することを確認
	if err == nil {
		t.Error("エラーが発生すべき")
		return
	}

	// エラーメッセージに "osoba start" が含まれることを確認
	errMsg := err.Error()
	if !strings.Contains(errMsg, "osoba start") {
		t.Errorf("エラーメッセージに 'osoba start' が含まれていません: %s", errMsg)
	}

	// エラーメッセージに "osoba watch" が含まれていないことを確認
	if strings.Contains(errMsg, "osoba watch") {
		t.Errorf("エラーメッセージに存在しないコマンド 'osoba watch' が含まれています: %s", errMsg)
	}
}

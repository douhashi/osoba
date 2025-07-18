package tmux

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// MockManager はテスト用のモックマネージャー
type MockManager struct {
	SessionManager
	WindowManager
	PaneManager
}

// createTestMockManager はテスト用のモックマネージャーを作成
func createTestMockManager() Manager {
	return &MockManager{
		SessionManager: &testSessionManager{},
		WindowManager:  &testWindowManager{},
		PaneManager:    &testPaneManager{},
	}
}

// testSessionManager はテスト用のSessionManager実装
type testSessionManager struct{}

func (m *testSessionManager) CheckTmuxInstalled() error {
	// テスト環境では常に成功
	return nil
}

func (m *testSessionManager) SessionExists(sessionName string) (bool, error) {
	// テスト環境では常にfalse
	return false, nil
}

func (m *testSessionManager) CreateSession(sessionName string) error {
	// テスト環境では常に成功
	return nil
}

func (m *testSessionManager) EnsureSession(sessionName string) error {
	// テスト環境では常に成功
	return nil
}

func (m *testSessionManager) ListSessions(prefix string) ([]string, error) {
	// テスト環境では空のリストを返す
	return []string{}, nil
}

// testWindowManager はテスト用のWindowManager実装
type testWindowManager struct{}

func (m *testWindowManager) CreateWindow(sessionName, windowName string) error {
	return nil
}

func (m *testWindowManager) SwitchToWindow(sessionName, windowName string) error {
	return nil
}

func (m *testWindowManager) WindowExists(sessionName, windowName string) (bool, error) {
	return false, nil
}

func (m *testWindowManager) KillWindow(sessionName, windowName string) error {
	return nil
}

func (m *testWindowManager) CreateOrReplaceWindow(sessionName, windowName string) error {
	return nil
}

func (m *testWindowManager) ListWindows(sessionName string) ([]string, error) {
	return []string{}, nil
}

func (m *testWindowManager) SendKeys(sessionName, windowName, keys string) error {
	return nil
}

func (m *testWindowManager) ClearWindow(sessionName, windowName string) error {
	return nil
}

func (m *testWindowManager) RunInWindow(sessionName, windowName, command string) error {
	return nil
}

func (m *testWindowManager) GetIssueWindow(issueNumber int) string {
	return "issue-" + strconv.Itoa(issueNumber)
}

func (m *testWindowManager) MatchIssueWindow(windowName string) bool {
	return strings.HasPrefix(windowName, "issue-")
}

func (m *testWindowManager) FindIssueWindow(windowName string) (int, bool) {
	if strings.HasPrefix(windowName, "issue-") {
		// 簡単な実装
		return 1, true
	}
	return 0, false
}

// testPaneManager はテスト用のPaneManager実装
type testPaneManager struct{}

func (m *testPaneManager) CreatePane(sessionName, windowName string, opts PaneOptions) (*PaneInfo, error) {
	return &PaneInfo{
		Index:  1,
		Title:  opts.Title,
		Active: true,
		Width:  80,
		Height: 40,
	}, nil
}

func (m *testPaneManager) SelectPane(sessionName, windowName string, paneIndex int) error {
	return nil
}

func (m *testPaneManager) SetPaneTitle(sessionName, windowName string, paneIndex int, title string) error {
	return nil
}

func (m *testPaneManager) ListPanes(sessionName, windowName string) ([]*PaneInfo, error) {
	return []*PaneInfo{
		{Index: 0, Title: "Plan", Active: true, Width: 80, Height: 40},
	}, nil
}

func (m *testPaneManager) GetPaneByTitle(sessionName, windowName string, title string) (*PaneInfo, error) {
	if title == "Plan" {
		return &PaneInfo{Index: 0, Title: "Plan", Active: true, Width: 80, Height: 40}, nil
	}
	return nil, fmt.Errorf("pane with title '%s' not found", title)
}

// init はパッケージ初期化時に実行される
func init() {
	// テスト環境ではモックマネージャーを使用
	if os.Getenv("OSOBA_TEST_MODE") == "true" {
		// テスト用のモックマネージャーを設定
		SetGlobalManager(createTestMockManager())
		return
	}

	// 本番環境ではデフォルトマネージャーを使用
	// globalManager は global_manager.go で定義済み
}

package tmux

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// MockManager はテスト用のモックマネージャー
type MockManager struct {
	SessionManager
	WindowManager
	PaneManager
	DiagnosticManager
}

// GetPaneBaseIndex テスト用のpane-base-index取得
func (m *MockManager) GetPaneBaseIndex() (int, error) {
	// テスト環境では0を返す
	return 0, nil
}

// createTestMockManager はテスト用のモックマネージャーを作成
func createTestMockManager() Manager {
	return &MockManager{
		SessionManager:    &testSessionManager{},
		WindowManager:     &testWindowManager{},
		PaneManager:       &testPaneManager{},
		DiagnosticManager: &testDiagnosticManager{},
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

func (m *testWindowManager) CreateWindowForIssueWithNewWindowDetection(sessionName string, issueNumber int) (string, bool, error) {
	// テスト環境では常に新規ウィンドウとして扱う
	windowName := fmt.Sprintf("issue-%d", issueNumber)
	return windowName, true, nil
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

func (m *testPaneManager) ResizePanesEvenly(sessionName, windowName string) error {
	// テスト環境では常に成功
	return nil
}

func (m *testPaneManager) ResizePanesEvenlyWithRetry(sessionName, windowName string) error {
	// テスト環境では常に成功
	return nil
}

func (m *testPaneManager) GetWindowSize(sessionName, windowName string) (width, height int, err error) {
	// テスト環境ではデフォルトサイズを返す
	return 120, 40, nil
}

func (m *testPaneManager) KillPane(sessionName, windowName string, paneIndex int) error {
	// テスト環境では常に成功
	return nil
}

// testDiagnosticManager はテスト用のDiagnosticManager実装
type testDiagnosticManager struct{}

func (m *testDiagnosticManager) DiagnoseSession(sessionName string) (*SessionDiagnostics, error) {
	return &SessionDiagnostics{
		Name:      sessionName,
		Windows:   1,
		Attached:  false,
		Created:   "1641641600",
		Errors:    []string{},
		Metadata:  map[string]string{"exists": "true", "test": "true"},
		Timestamp: time.Now(),
	}, nil
}

func (m *testDiagnosticManager) DiagnoseWindow(sessionName, windowName string) (*WindowDiagnostics, error) {
	return &WindowDiagnostics{
		Name:        windowName,
		SessionName: sessionName,
		Index:       0,
		Exists:      true,
		Active:      false,
		Panes:       1,
		IssueNumber: 0,
		Phase:       "",
		Errors:      []string{},
		Metadata:    map[string]string{"exists": "true", "test": "true"},
		Timestamp:   time.Now(),
	}, nil
}

func (m *testDiagnosticManager) ListSessionDiagnostics(prefix string) ([]*SessionDiagnostics, error) {
	return []*SessionDiagnostics{
		{
			Name:      "test-osoba-session",
			Windows:   2,
			Attached:  false,
			Created:   "1641641600",
			Errors:    []string{},
			Metadata:  map[string]string{"exists": "true", "test": "true"},
			Timestamp: time.Now(),
		},
	}, nil
}

func (m *testDiagnosticManager) ListWindowDiagnostics(sessionName string) ([]*WindowDiagnostics, error) {
	return []*WindowDiagnostics{
		{
			Name:        "330-plan",
			SessionName: sessionName,
			Index:       0,
			Exists:      true,
			Active:      true,
			Panes:       1,
			IssueNumber: 330,
			Phase:       "plan",
			Errors:      []string{},
			Metadata:    map[string]string{"exists": "true", "test": "true", "issue_window": "true"},
			Timestamp:   time.Now(),
		},
		{
			Name:        "general",
			SessionName: sessionName,
			Index:       1,
			Exists:      true,
			Active:      false,
			Panes:       1,
			IssueNumber: 0,
			Phase:       "",
			Errors:      []string{},
			Metadata:    map[string]string{"exists": "true", "test": "true"},
			Timestamp:   time.Now(),
		},
	}, nil
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

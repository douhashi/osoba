package tmux

// SessionManager はtmuxセッション操作のインターフェース
type SessionManager interface {
	// CheckTmuxInstalled tmuxがインストールされているか確認
	CheckTmuxInstalled() error

	// SessionExists 指定された名前のtmuxセッションが存在するか確認
	SessionExists(sessionName string) (bool, error)

	// CreateSession 新しいtmuxセッションを作成
	CreateSession(sessionName string) error

	// EnsureSession tmuxセッションが存在しない場合は作成
	EnsureSession(sessionName string) error

	// ListSessions 指定されたプレフィックスで始まるセッション一覧を取得
	ListSessions(prefix string) ([]string, error)
}

// WindowManager はtmuxウィンドウ操作のインターフェース
type WindowManager interface {
	// CreateWindow 新しいtmuxウィンドウを作成
	CreateWindow(sessionName, windowName string) error

	// SwitchToWindow 指定されたウィンドウに切り替え
	SwitchToWindow(sessionName, windowName string) error

	// WindowExists 指定されたウィンドウが存在するか確認
	WindowExists(sessionName, windowName string) (bool, error)

	// KillWindow 指定されたウィンドウを削除
	KillWindow(sessionName, windowName string) error

	// CreateOrReplaceWindow ウィンドウが存在する場合は削除してから新規作成
	CreateOrReplaceWindow(sessionName, windowName string) error

	// ListWindows 指定されたセッションのウィンドウ一覧を取得
	ListWindows(sessionName string) ([]string, error)

	// SendKeys 指定されたウィンドウにキーを送信
	SendKeys(sessionName, windowName, keys string) error

	// ClearWindow ウィンドウの内容をクリア
	ClearWindow(sessionName, windowName string) error

	// RunInWindow ウィンドウ内でコマンドを実行
	RunInWindow(sessionName, windowName, command string) error

	// GetIssueWindow Issue番号に対応するウィンドウ名を取得
	GetIssueWindow(issueNumber int) string

	// MatchIssueWindow ウィンドウ名がIssueパターンにマッチするか確認
	MatchIssueWindow(windowName string) bool

	// FindIssueWindow ウィンドウ名からIssue番号を抽出
	FindIssueWindow(windowName string) (int, bool)
}

// Manager はtmuxの全操作を統合したインターフェース
type Manager interface {
	SessionManager
	WindowManager
	PaneManager
}

// DefaultManager はManagerインターフェースのデフォルト実装
type DefaultManager struct {
	executor CommandExecutor
}

// NewDefaultManager はDefaultManagerの新しいインスタンスを作成
func NewDefaultManager() *DefaultManager {
	return &DefaultManager{
		executor: &DefaultCommandExecutor{},
	}
}

// NewDefaultManagerWithExecutor は指定されたExecutorを使用してDefaultManagerを作成
func NewDefaultManagerWithExecutor(executor CommandExecutor) *DefaultManager {
	return &DefaultManager{
		executor: executor,
	}
}

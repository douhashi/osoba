package tmux

// globalManager はパッケージレベルのデフォルトマネージャー
var globalManager Manager = NewDefaultManager()

// SetGlobalManager はグローバルマネージャーを設定する（主にテスト用）
func SetGlobalManager(m Manager) {
	globalManager = m
}

// GetGlobalManager はグローバルマネージャーを取得する
func GetGlobalManager() Manager {
	return globalManager
}

// 以下、既存の関数をグローバルマネージャーを使用するように置き換えるラッパー

// CheckTmuxInstalled tmuxがインストールされているか確認
func CheckTmuxInstalled() error {
	return globalManager.CheckTmuxInstalled()
}

// SessionExists 指定された名前のtmuxセッションが存在するか確認
func SessionExists(sessionName string) (bool, error) {
	return globalManager.SessionExists(sessionName)
}

// CreateSession 新しいtmuxセッションを作成
func CreateSession(sessionName string) error {
	return globalManager.CreateSession(sessionName)
}

// EnsureSession tmuxセッションが存在しない場合は作成
func EnsureSession(sessionName string) error {
	return globalManager.EnsureSession(sessionName)
}

// ListSessions 指定されたプレフィックスで始まるセッション一覧を取得
func ListSessions(prefix string) ([]string, error) {
	return globalManager.ListSessions(prefix)
}

// CreateWindow 新しいtmuxウィンドウを作成
func CreateWindow(sessionName, windowName string) error {
	return globalManager.CreateWindow(sessionName, windowName)
}

// SwitchToWindow 指定されたウィンドウに切り替え
func SwitchToWindow(sessionName, windowName string) error {
	return globalManager.SwitchToWindow(sessionName, windowName)
}

// WindowExists 指定されたウィンドウが存在するか確認
func WindowExists(sessionName, windowName string) (bool, error) {
	return globalManager.WindowExists(sessionName, windowName)
}

// KillWindow 指定されたウィンドウを削除
func KillWindow(sessionName, windowName string) error {
	return globalManager.KillWindow(sessionName, windowName)
}

// CreateOrReplaceWindow ウィンドウが存在する場合は削除してから新規作成
func CreateOrReplaceWindow(sessionName, windowName string) error {
	return globalManager.CreateOrReplaceWindow(sessionName, windowName)
}

// ListWindowsAsStrings 指定されたセッションのウィンドウ一覧を文字列配列で取得
func ListWindowsAsStrings(sessionName string) ([]string, error) {
	return globalManager.ListWindows(sessionName)
}

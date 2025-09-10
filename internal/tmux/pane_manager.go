package tmux

// PaneManager はtmuxペイン操作のインターフェース
type PaneManager interface {
	// CreatePane 新しいペインを作成
	CreatePane(sessionName, windowName string, opts PaneOptions) (*PaneInfo, error)

	// SelectPane 指定されたペインを選択
	SelectPane(sessionName, windowName string, paneIndex int) error

	// SetPaneTitle ペインのタイトルを設定
	SetPaneTitle(sessionName, windowName string, paneIndex int, title string) error

	// ListPanes ウィンドウ内のペイン一覧を取得
	ListPanes(sessionName, windowName string) ([]*PaneInfo, error)

	// GetPaneByTitle タイトルでペインを検索
	GetPaneByTitle(sessionName, windowName string, title string) (*PaneInfo, error)

	// ResizePanesEvenly ペインを均等にリサイズ
	ResizePanesEvenly(sessionName, windowName string) error

	// KillPane 指定されたペインを削除
	KillPane(sessionName, windowName string, paneIndex int) error
}

// PaneOptions ペイン作成時のオプション
type PaneOptions struct {
	Split      string // "-v" (vertical) or "-h" (horizontal)
	Percentage int    // split percentage
	Title      string // pane title for border
}

// PaneInfo ペイン情報
type PaneInfo struct {
	Index  int
	Title  string
	Active bool
	Width  int
	Height int
}

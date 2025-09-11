package tmux

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ペインリサイズ機能の安定性向上のための定数
const (
	// ウィンドウサイズの最小要件
	MinWindowWidth  = 80
	MinWindowHeight = 24

	// リトライ設定
	MaxRetries        = 3
	BaseRetryInterval = 100 * time.Millisecond

	// リサイズ後の安定化待機時間
	LayoutStabilizationDelay = 100 * time.Millisecond
)

// CreatePane 新しいペインを作成
func (m *DefaultManager) CreatePane(sessionName, windowName string, opts PaneOptions) (*PaneInfo, error) {
	// ペイン数制限のチェック（ペイン作成前）
	if opts.Config != nil && opts.Config.LimitPanesEnabled {
		if err := m.enforcePaneLimit(sessionName, windowName, opts.Config.MaxPanesPerWindow); err != nil {
			// ペイン制限のエラーはログに記録するが、処理は継続（ベストエフォート）
			// 実際の運用環境ではログを出力する
			// log.Printf("Warning: Failed to enforce pane limit: %v", err)
		}
	}

	// デフォルト値の設定
	percentage := opts.Percentage
	if percentage == 0 {
		percentage = 50
	}

	// split-windowコマンドの実行
	args := []string{"split-window", opts.Split, "-p", strconv.Itoa(percentage), "-t", fmt.Sprintf("%s:%s", sessionName, windowName)}
	if _, err := m.executor.Execute("tmux", args...); err != nil {
		return nil, fmt.Errorf("failed to create pane: %w", err)
	}

	// 作成されたペインの情報を取得
	panes, err := m.ListPanes(sessionName, windowName)
	if err != nil {
		return nil, fmt.Errorf("failed to list panes after creation: %w", err)
	}

	// 最後のペイン（新しく作成されたもの）を取得
	if len(panes) == 0 {
		return nil, fmt.Errorf("no panes found after creation")
	}
	newPane := panes[len(panes)-1]

	// タイトルを設定
	if opts.Title != "" {
		if err := m.SetPaneTitle(sessionName, windowName, newPane.Index, opts.Title); err != nil {
			return nil, fmt.Errorf("failed to set pane title: %w", err)
		}
		newPane.Title = opts.Title
	}

	// ペイン作成後の自動レイアウト調整
	// エラーが発生してもペイン作成は成功として扱う（ベストエフォート）
	if err := m.ResizePanesEvenlyWithRetry(sessionName, windowName); err != nil {
		// レイアウト調整エラーは記録するが、ペイン作成の結果には影響しない
		// 実際の運用環境ではログを出力する
		// log.Printf("Warning: Failed to adjust layout after pane creation: %v", err)
	}

	return newPane, nil
}

// enforcePaneLimit ペイン数が制限を超えている場合、最古の非アクティブペインを削除
func (m *DefaultManager) enforcePaneLimit(sessionName, windowName string, maxPanes int) error {
	if maxPanes <= 0 {
		maxPanes = 3 // デフォルト値
	}

	panes, err := m.ListPanes(sessionName, windowName)
	if err != nil {
		return fmt.Errorf("failed to list panes: %w", err)
	}

	// 現在のペイン数が上限以下の場合は何もしない
	if len(panes) < maxPanes {
		return nil
	}

	// 最古の非アクティブペインを探す
	var oldestNonActiveIndex int = -1
	for _, pane := range panes {
		if !pane.Active {
			if oldestNonActiveIndex == -1 || pane.Index < oldestNonActiveIndex {
				oldestNonActiveIndex = pane.Index
			}
		}
	}

	// 非アクティブペインが見つかった場合は削除
	if oldestNonActiveIndex >= 0 {
		if err := m.KillPane(sessionName, windowName, oldestNonActiveIndex); err != nil {
			return fmt.Errorf("failed to kill pane %d: %w", oldestNonActiveIndex, err)
		}

		// ペイン削除後のレイアウト調整
		if err := m.ResizePanesEvenlyWithRetry(sessionName, windowName); err != nil {
			// レイアウト調整エラーは記録するが、処理は継続
			// log.Printf("Warning: Failed to adjust layout after pane removal: %v", err)
		}
	}

	return nil
}

// SelectPane 指定されたペインを選択
func (m *DefaultManager) SelectPane(sessionName, windowName string, paneIndex int) error {
	args := []string{"select-pane", "-t", fmt.Sprintf("%s:%s.%d", sessionName, windowName, paneIndex)}
	if _, err := m.executor.Execute("tmux", args...); err != nil {
		return fmt.Errorf("failed to select pane: %w", err)
	}
	return nil
}

// SetPaneTitle ペインのタイトルを設定
func (m *DefaultManager) SetPaneTitle(sessionName, windowName string, paneIndex int, title string) error {
	// ペインのボーダーフォーマットを設定
	target := fmt.Sprintf("%s:%s.%d", sessionName, windowName, paneIndex)
	args := []string{"set-option", "-t", target, "-p", "pane-border-format", fmt.Sprintf(" %s ", title)}
	if _, err := m.executor.Execute("tmux", args...); err != nil {
		return fmt.Errorf("failed to set pane title for %s: %w", target, err)
	}
	return nil
}

// ListPanes ウィンドウ内のペイン一覧を取得
func (m *DefaultManager) ListPanes(sessionName, windowName string) ([]*PaneInfo, error) {
	// list-panesコマンドで情報を取得
	args := []string{"list-panes", "-t", fmt.Sprintf("%s:%s", sessionName, windowName), "-F", "#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}"}
	output, err := m.executor.Execute("tmux", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list panes: %w", err)
	}

	// 出力をパース
	lines := strings.Split(strings.TrimSpace(output), "\n")
	panes := make([]*PaneInfo, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}
		pane, err := parsePaneInfo(line)
		if err != nil {
			return nil, fmt.Errorf("failed to parse pane info: %w", err)
		}
		panes = append(panes, pane)
	}

	return panes, nil
}

// GetPaneByTitle タイトルでペインを検索
func (m *DefaultManager) GetPaneByTitle(sessionName, windowName string, title string) (*PaneInfo, error) {
	panes, err := m.ListPanes(sessionName, windowName)
	if err != nil {
		return nil, err
	}

	for _, pane := range panes {
		if pane.Title == title {
			return pane, nil
		}
	}

	return nil, fmt.Errorf("pane with title '%s' not found", title)
}

// parsePaneInfo ペイン情報の文字列をパース
func parsePaneInfo(line string) (*PaneInfo, error) {
	parts := strings.Split(line, ":")
	if len(parts) != 5 {
		return nil, fmt.Errorf("invalid pane info format: expected 5 fields, got %d", len(parts))
	}

	index, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid pane index: %w", err)
	}

	active, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid pane active state: %w", err)
	}

	width, err := strconv.Atoi(parts[3])
	if err != nil {
		return nil, fmt.Errorf("invalid pane width: %w", err)
	}

	height, err := strconv.Atoi(parts[4])
	if err != nil {
		return nil, fmt.Errorf("invalid pane height: %w", err)
	}

	return &PaneInfo{
		Index:  index,
		Title:  parts[1],
		Active: active == 1,
		Width:  width,
		Height: height,
	}, nil
}

// ResizePanesEvenly ペインを均等にリサイズ
// 下位互換性のため、リトライ機能付きメソッドを呼び出すラッパー
func (m *DefaultManager) ResizePanesEvenly(sessionName, windowName string) error {
	return m.ResizePanesEvenlyWithRetry(sessionName, windowName)
}

// GetWindowSize ウィンドウのサイズ（幅、高さ）を取得
func (m *DefaultManager) GetWindowSize(sessionName, windowName string) (width, height int, err error) {
	target := fmt.Sprintf("%s:%s", sessionName, windowName)
	args := []string{"display-message", "-p", "-t", target, "#{window_width} #{window_height}"}

	output, err := m.executor.Execute("tmux", args...)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get window size for %s: %w", target, err)
	}

	// "width height" 形式の出力をパース
	parts := strings.Fields(strings.TrimSpace(output))
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid window size format: expected 2 fields, got %d", len(parts))
	}

	width, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid window width: %w", err)
	}

	height, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid window height: %w", err)
	}

	return width, height, nil
}

// ResizePanesEvenlyWithRetry ペインを均等にリサイズ（リトライ機能付き）
func (m *DefaultManager) ResizePanesEvenlyWithRetry(sessionName, windowName string) error {
	// ペイン数を確認（1個以下の場合はスキップ）
	panes, err := m.ListPanes(sessionName, windowName)
	if err != nil {
		return fmt.Errorf("failed to list panes: %w", err)
	}

	if len(panes) <= 1 {
		// ペインが1個以下の場合はリサイズ不要
		return nil
	}

	// ウィンドウサイズチェック
	width, height, err := m.GetWindowSize(sessionName, windowName)
	if err != nil {
		return fmt.Errorf("failed to get window size: %w", err)
	}

	// 最小サイズ要件をチェック
	if width < MinWindowWidth || height < MinWindowHeight {
		// サイズ不足の場合はログ出力してスキップ（エラーにはしない）
		// ログ機能は既存の実装に依存するため、コメントアウトしている
		// fmt.Printf("Window size (%dx%d) is too small for resizing, minimum required: %dx%d\n",
		//           width, height, MinWindowWidth, MinWindowHeight)
		return nil
	}

	// リトライロジック実行
	target := fmt.Sprintf("%s:%s", sessionName, windowName)
	args := []string{"select-layout", "-t", target, "even-horizontal"}

	var lastErr error
	for attempt := 1; attempt <= MaxRetries; attempt++ {
		// tmux select-layout even-horizontal を実行
		if _, err := m.executor.Execute("tmux", args...); err != nil {
			lastErr = err

			// 最大リトライに達していない場合は待機してリトライ
			if attempt < MaxRetries {
				// exponential backoff: 100ms, 200ms, 400ms
				delay := BaseRetryInterval * time.Duration(1<<(attempt-1))
				time.Sleep(delay)
				continue
			}

			// 最大リトライに達した場合はエラーを返す
			return fmt.Errorf("failed to resize panes evenly for %s after %d attempts: %w", target, MaxRetries, lastErr)
		}

		// 成功時は安定化待機時間を設定
		time.Sleep(LayoutStabilizationDelay)
		return nil
	}

	return fmt.Errorf("failed to resize panes evenly for %s: %w", target, lastErr)
}

// KillPane 指定されたペインを削除
func (m *DefaultManager) KillPane(sessionName, windowName string, paneIndex int) error {
	target := fmt.Sprintf("%s:%s.%d", sessionName, windowName, paneIndex)
	args := []string{"kill-pane", "-t", target}
	if _, err := m.executor.Execute("tmux", args...); err != nil {
		return fmt.Errorf("failed to kill pane %s: %w", target, err)
	}
	return nil
}

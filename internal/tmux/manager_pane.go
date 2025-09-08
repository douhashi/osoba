package tmux

import (
	"fmt"
	"strconv"
	"strings"
)

// CreatePane 新しいペインを作成
func (m *DefaultManager) CreatePane(sessionName, windowName string, opts PaneOptions) (*PaneInfo, error) {
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

	return newPane, nil
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
func (m *DefaultManager) ResizePanesEvenly(sessionName, windowName string) error {
	// ペイン数を確認（1個以下の場合はスキップ）
	panes, err := m.ListPanes(sessionName, windowName)
	if err != nil {
		return fmt.Errorf("failed to list panes: %w", err)
	}

	if len(panes) <= 1 {
		// ペインが1個以下の場合はリサイズ不要
		return nil
	}

	// tmux select-layout even-horizontal を実行
	target := fmt.Sprintf("%s:%s", sessionName, windowName)
	args := []string{"select-layout", "-t", target, "even-horizontal"}
	if _, err := m.executor.Execute("tmux", args...); err != nil {
		return fmt.Errorf("failed to resize panes evenly for %s: %w", target, err)
	}

	return nil
}

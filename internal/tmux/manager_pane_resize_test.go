package tmux

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockCommandExecutorは既存のpane_test.goで定義されているため、ここでは削除

func TestDefaultManager_ResizePanesEvenly(t *testing.T) {
	tests := []struct {
		name           string
		sessionName    string
		windowName     string
		panes          []*PaneInfo
		executorResult string
		executorError  error
		expectedError  bool
	}{
		{
			name:        "複数ペインでリサイズ成功",
			sessionName: "test-session",
			windowName:  "test-window",
			panes: []*PaneInfo{
				{Index: 0, Title: "pane1", Active: true, Width: 40, Height: 20},
				{Index: 1, Title: "pane2", Active: false, Width: 40, Height: 20},
			},
			executorResult: "",
			executorError:  nil,
			expectedError:  false,
		},
		{
			name:        "ペイン1個の場合はスキップ",
			sessionName: "test-session",
			windowName:  "test-window",
			panes: []*PaneInfo{
				{Index: 0, Title: "pane1", Active: true, Width: 80, Height: 20},
			},
			executorResult: "",
			executorError:  nil,
			expectedError:  false,
		},
		{
			name:           "ペイン0個の場合はスキップ",
			sessionName:    "test-session",
			windowName:     "test-window",
			panes:          []*PaneInfo{},
			executorResult: "",
			executorError:  nil,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックを作成
			mockExecutor := &MockCommandExecutor{}
			manager := &DefaultManager{executor: mockExecutor}

			// ListPanesのモック設定
			listPanesArgs := []string{"list-panes", "-t", tt.sessionName + ":" + tt.windowName, "-F", "#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}"}

			// ペイン情報をモック用の文字列に変換
			var panesOutput string
			for i, pane := range tt.panes {
				active := "0"
				if pane.Active {
					active = "1"
				}
				panesOutput += fmt.Sprintf("%d:%s:%s:%d:%d", pane.Index, pane.Title, active, pane.Width, pane.Height)
				if i < len(tt.panes)-1 {
					panesOutput += "\n"
				}
			}

			mockExecutor.On("Execute", "tmux", listPanesArgs).Return(panesOutput, nil)

			// ペインが2個以上の場合のみ、select-layoutコマンドのモックを設定
			if len(tt.panes) > 1 {
				selectLayoutArgs := []string{"select-layout", "-t", tt.sessionName + ":" + tt.windowName, "even-horizontal"}
				mockExecutor.On("Execute", "tmux", selectLayoutArgs).Return(tt.executorResult, tt.executorError)
			}

			// テスト実行
			err := manager.ResizePanesEvenly(tt.sessionName, tt.windowName)

			// 結果検証
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockExecutor.AssertExpectations(t)
		})
	}
}

func TestDefaultManager_ResizePanesEvenly_ListPanesError(t *testing.T) {
	mockExecutor := &MockCommandExecutor{}
	manager := &DefaultManager{executor: mockExecutor}

	sessionName := "test-session"
	windowName := "test-window"

	// ListPanesでエラーが発生する場合
	listPanesArgs := []string{"list-panes", "-t", sessionName + ":" + windowName, "-F", "#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}"}
	mockExecutor.On("Execute", "tmux", listPanesArgs).Return("", fmt.Errorf("tmux error"))

	err := manager.ResizePanesEvenly(sessionName, windowName)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list panes")
	mockExecutor.AssertExpectations(t)
}

func TestDefaultManager_ResizePanesEvenly_SelectLayoutError(t *testing.T) {
	mockExecutor := &MockCommandExecutor{}
	manager := &DefaultManager{executor: mockExecutor}

	sessionName := "test-session"
	windowName := "test-window"

	// 2つのペインを設定
	panesOutput := "0:pane1:1:40:20\n1:pane2:0:40:20"
	listPanesArgs := []string{"list-panes", "-t", sessionName + ":" + windowName, "-F", "#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}"}
	mockExecutor.On("Execute", "tmux", listPanesArgs).Return(panesOutput, nil)

	// select-layoutでエラーが発生する場合
	selectLayoutArgs := []string{"select-layout", "-t", sessionName + ":" + windowName, "even-horizontal"}
	mockExecutor.On("Execute", "tmux", selectLayoutArgs).Return("", fmt.Errorf("select-layout error"))

	err := manager.ResizePanesEvenly(sessionName, windowName)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resize panes evenly")
	mockExecutor.AssertExpectations(t)
}

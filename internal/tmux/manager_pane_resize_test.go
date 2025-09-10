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

			// ペインが2個以上の場合のみ、新しいリトライ機能のモックを設定
			if len(tt.panes) > 1 {
				// ウィンドウサイズチェックのモック
				target := fmt.Sprintf("%s:%s", tt.sessionName, tt.windowName)
				windowSizeArgs := []string{"display-message", "-p", "-t", target, "#{window_width} #{window_height}"}
				mockExecutor.On("Execute", "tmux", windowSizeArgs).Return("120 40", nil)

				// select-layoutのモック
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

// TestDefaultManager_GetWindowSize ウィンドウサイズ取得のテスト
func TestDefaultManager_GetWindowSize(t *testing.T) {
	tests := []struct {
		name           string
		sessionName    string
		windowName     string
		mockOutput     string
		mockError      error
		expectedWidth  int
		expectedHeight int
		expectedError  bool
	}{
		{
			name:           "正常なウィンドウサイズ取得",
			sessionName:    "test-session",
			windowName:     "test-window",
			mockOutput:     "120 40",
			mockError:      nil,
			expectedWidth:  120,
			expectedHeight: 40,
			expectedError:  false,
		},
		{
			name:           "最小サイズのウィンドウ",
			sessionName:    "test-session",
			windowName:     "test-window",
			mockOutput:     "80 24",
			mockError:      nil,
			expectedWidth:  80,
			expectedHeight: 24,
			expectedError:  false,
		},
		{
			name:          "tmuxコマンドエラー",
			sessionName:   "test-session",
			windowName:    "test-window",
			mockOutput:    "",
			mockError:     fmt.Errorf("tmux: no server running"),
			expectedError: true,
		},
		{
			name:          "不正な出力フォーマット",
			sessionName:   "test-session",
			windowName:    "test-window",
			mockOutput:    "invalid-format",
			mockError:     nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := &MockCommandExecutor{}
			manager := &DefaultManager{executor: mockExecutor}

			// display-messageコマンドのモック設定
			target := fmt.Sprintf("%s:%s", tt.sessionName, tt.windowName)
			args := []string{"display-message", "-p", "-t", target, "#{window_width} #{window_height}"}
			mockExecutor.On("Execute", "tmux", args).Return(tt.mockOutput, tt.mockError)

			// テスト実行
			width, height, err := manager.GetWindowSize(tt.sessionName, tt.windowName)

			// 結果検証
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedWidth, width)
				assert.Equal(t, tt.expectedHeight, height)
			}

			mockExecutor.AssertExpectations(t)
		})
	}
}

// TestDefaultManager_ResizePanesEvenlyWithRetry リトライ機能付きResizePanesEvenlyのテスト
func TestDefaultManager_ResizePanesEvenlyWithRetry(t *testing.T) {
	tests := []struct {
		name               string
		sessionName        string
		windowName         string
		panesOutput        string
		windowSizeOutput   string
		selectLayoutErrors []error // 各リトライにおけるエラー
		expectedError      bool
	}{
		{
			name:               "初回で成功するケース",
			sessionName:        "test-session",
			windowName:         "test-window",
			panesOutput:        "0:pane1:1:60:30\n1:pane2:0:60:30",
			windowSizeOutput:   "120 40",
			selectLayoutErrors: []error{nil},
			expectedError:      false,
		},
		{
			name:               "2回目で成功するケース",
			sessionName:        "test-session",
			windowName:         "test-window",
			panesOutput:        "0:pane1:1:60:30\n1:pane2:0:60:30",
			windowSizeOutput:   "120 40",
			selectLayoutErrors: []error{fmt.Errorf("temporary error"), nil},
			expectedError:      false,
		},
		{
			name:             "最大リトライで失敗するケース",
			sessionName:      "test-session",
			windowName:       "test-window",
			panesOutput:      "0:pane1:1:60:30\n1:pane2:0:60:30",
			windowSizeOutput: "120 40",
			selectLayoutErrors: []error{
				fmt.Errorf("error1"), fmt.Errorf("error2"), fmt.Errorf("error3"),
			},
			expectedError: true,
		},
		{
			name:               "ウィンドウサイズ不足でスキップ",
			sessionName:        "test-session",
			windowName:         "test-window",
			panesOutput:        "0:pane1:1:35:15\n1:pane2:0:35:15",
			windowSizeOutput:   "70 20",
			selectLayoutErrors: []error{}, // select-layoutは呼ばれない
			expectedError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := &MockCommandExecutor{}
			manager := &DefaultManager{executor: mockExecutor}

			// ListPanesのモック設定
			listPanesArgs := []string{"list-panes", "-t", tt.sessionName + ":" + tt.windowName, "-F", "#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}"}
			mockExecutor.On("Execute", "tmux", listPanesArgs).Return(tt.panesOutput, nil)

			// GetWindowSizeのモック設定
			if tt.windowSizeOutput != "" {
				target := fmt.Sprintf("%s:%s", tt.sessionName, tt.windowName)
				windowSizeArgs := []string{"display-message", "-p", "-t", target, "#{window_width} #{window_height}"}
				mockExecutor.On("Execute", "tmux", windowSizeArgs).Return(tt.windowSizeOutput, nil)
			}

			// select-layoutのモック設定（リトライ回数分）
			selectLayoutArgs := []string{"select-layout", "-t", tt.sessionName + ":" + tt.windowName, "even-horizontal"}
			for _, err := range tt.selectLayoutErrors {
				mockExecutor.On("Execute", "tmux", selectLayoutArgs).Return("", err).Once()
			}

			// テスト実行
			err := manager.ResizePanesEvenlyWithRetry(tt.sessionName, tt.windowName)

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

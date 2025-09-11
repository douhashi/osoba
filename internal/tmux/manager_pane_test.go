package tmux

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreatePane_WithPaneLimit(t *testing.T) {
	tests := []struct {
		name            string
		config          *PaneConfig
		existingPanes   string
		setupMock       func(*MockCommandExecutor)
		expectedError   bool
		expectedPaneIdx int
	}{
		{
			name: "制限有効・上限未満",
			config: &PaneConfig{
				LimitPanesEnabled: true,
				MaxPanesPerWindow: 3,
			},
			existingPanes: "0:Plan:1:80:24\n1:Implementation:0:80:24",
			setupMock: func(m *MockCommandExecutor) {
				// ListPanes (制限チェック用)
				m.On("Execute", "tmux", []string{"list-panes", "-t", "test-session:test-window", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}"}).
					Return("0:Plan:1:80:24\n1:Implementation:0:80:24", nil).Once()

				// CreatePane
				m.On("Execute", "tmux", []string{"split-window", "-h", "-p", "50", "-t", "test-session:test-window"}).
					Return("", nil).Once()

				// ListPanes (作成後)
				m.On("Execute", "tmux", []string{"list-panes", "-t", "test-session:test-window", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}"}).
					Return("0:Plan:0:80:24\n1:Implementation:0:80:24\n2::1:80:24", nil).Once()

				// SetPaneTitle
				m.On("Execute", "tmux", []string{"set-option", "-t", "test-session:test-window.2", "-p",
					"pane-border-format", " Review "}).
					Return("", nil).Once()

				// ResizePanesEvenlyWithRetry - ListPanes
				m.On("Execute", "tmux", []string{"list-panes", "-t", "test-session:test-window", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}"}).
					Return("0:Plan:0:80:24\n1:Implementation:0:80:24\n2:Review:1:80:24", nil).Once()

				// ResizePanesEvenlyWithRetry - GetWindowSize
				m.On("Execute", "tmux", []string{"display-message", "-p", "-t", "test-session:test-window",
					"#{window_width} #{window_height}"}).
					Return("240 24", nil).Once()

				// ResizePanesEvenlyWithRetry - select-layout
				m.On("Execute", "tmux", []string{"select-layout", "-t", "test-session:test-window", "even-horizontal"}).
					Return("", nil).Once()
			},
			expectedError:   false,
			expectedPaneIdx: 2,
		},
		{
			name: "制限有効・上限到達・非アクティブペイン削除",
			config: &PaneConfig{
				LimitPanesEnabled: true,
				MaxPanesPerWindow: 3,
			},
			existingPanes: "0:Plan:0:80:24\n1:Implementation:0:80:24\n2:Review:1:80:24",
			setupMock: func(m *MockCommandExecutor) {
				// ListPanes (制限チェック用)
				m.On("Execute", "tmux", []string{"list-panes", "-t", "test-session:test-window", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}"}).
					Return("0:Plan:0:80:24\n1:Implementation:0:80:24\n2:Review:1:80:24", nil).Once()

				// KillPane (最古の非アクティブペイン削除)
				m.On("Execute", "tmux", []string{"kill-pane", "-t", "test-session:test-window.0"}).
					Return("", nil).Once()

				// ResizePanesEvenlyWithRetry after kill - ListPanes
				m.On("Execute", "tmux", []string{"list-panes", "-t", "test-session:test-window", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}"}).
					Return("0:Implementation:0:120:24\n1:Review:1:120:24", nil).Once()

				// ResizePanesEvenlyWithRetry after kill - GetWindowSize
				m.On("Execute", "tmux", []string{"display-message", "-p", "-t", "test-session:test-window",
					"#{window_width} #{window_height}"}).
					Return("240 24", nil).Once()

				// ResizePanesEvenlyWithRetry after kill - select-layout
				m.On("Execute", "tmux", []string{"select-layout", "-t", "test-session:test-window", "even-horizontal"}).
					Return("", nil).Once()

				// CreatePane
				m.On("Execute", "tmux", []string{"split-window", "-h", "-p", "50", "-t", "test-session:test-window"}).
					Return("", nil).Once()

				// ListPanes (作成後)
				m.On("Execute", "tmux", []string{"list-panes", "-t", "test-session:test-window", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}"}).
					Return("0:Implementation:0:80:24\n1:Review:0:80:24\n2::1:80:24", nil).Once()

				// SetPaneTitle
				m.On("Execute", "tmux", []string{"set-option", "-t", "test-session:test-window.2", "-p",
					"pane-border-format", " Debug "}).
					Return("", nil).Once()

				// ResizePanesEvenlyWithRetry - ListPanes
				m.On("Execute", "tmux", []string{"list-panes", "-t", "test-session:test-window", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}"}).
					Return("0:Implementation:0:80:24\n1:Review:0:80:24\n2:Debug:1:80:24", nil).Once()

				// ResizePanesEvenlyWithRetry - GetWindowSize
				m.On("Execute", "tmux", []string{"display-message", "-p", "-t", "test-session:test-window",
					"#{window_width} #{window_height}"}).
					Return("240 24", nil).Once()

				// ResizePanesEvenlyWithRetry - select-layout
				m.On("Execute", "tmux", []string{"select-layout", "-t", "test-session:test-window", "even-horizontal"}).
					Return("", nil).Once()
			},
			expectedError:   false,
			expectedPaneIdx: 2,
		},
		{
			name:   "制限無効",
			config: nil,
			setupMock: func(m *MockCommandExecutor) {
				// CreatePane
				m.On("Execute", "tmux", []string{"split-window", "-h", "-p", "50", "-t", "test-session:test-window"}).
					Return("", nil).Once()

				// ListPanes (作成後)
				m.On("Execute", "tmux", []string{"list-panes", "-t", "test-session:test-window", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}"}).
					Return("0:Plan:0:80:24\n1::1:80:24", nil).Once()

				// SetPaneTitle
				m.On("Execute", "tmux", []string{"set-option", "-t", "test-session:test-window.1", "-p",
					"pane-border-format", " Implementation "}).
					Return("", nil).Once()

				// ResizePanesEvenlyWithRetry - ListPanes
				m.On("Execute", "tmux", []string{"list-panes", "-t", "test-session:test-window", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}"}).
					Return("0:Plan:0:120:24\n1:Implementation:1:120:24", nil).Once()

				// ResizePanesEvenlyWithRetry - GetWindowSize
				m.On("Execute", "tmux", []string{"display-message", "-p", "-t", "test-session:test-window",
					"#{window_width} #{window_height}"}).
					Return("240 24", nil).Once()

				// ResizePanesEvenlyWithRetry - select-layout
				m.On("Execute", "tmux", []string{"select-layout", "-t", "test-session:test-window", "even-horizontal"}).
					Return("", nil).Once()
			},
			expectedError:   false,
			expectedPaneIdx: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := new(MockCommandExecutor)
			tt.setupMock(mockExec)
			defer mockExec.AssertExpectations(t)

			manager := NewDefaultManagerWithExecutor(mockExec)

			opts := PaneOptions{
				Split:      "-h",
				Percentage: 50,
				Config:     tt.config,
			}

			// タイトルを設定してテスト
			titles := map[string]string{
				"制限有効・上限未満":             "Review",
				"制限有効・上限到達・非アクティブペイン削除": "Debug",
				"制限無効": "Implementation",
			}

			if title, ok := titles[tt.name]; ok {
				opts.Title = title
				pane, err := manager.CreatePane("test-session", "test-window", opts)

				if tt.expectedError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, pane)
					assert.Equal(t, tt.expectedPaneIdx, pane.Index)
					assert.Equal(t, title, pane.Title)
				}
			}
		})
	}
}

func TestEnforcePaneLimit(t *testing.T) {
	tests := []struct {
		name          string
		maxPanes      int
		existingPanes string
		setupMock     func(*MockCommandExecutor)
		expectedError bool
	}{
		{
			name:          "上限未満",
			maxPanes:      3,
			existingPanes: "0:Plan:1:120:24\n1:Implementation:0:120:24",
			setupMock: func(m *MockCommandExecutor) {
				// ListPanes
				m.On("Execute", "tmux", []string{"list-panes", "-t", "test-session:test-window", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}"}).
					Return("0:Plan:1:120:24\n1:Implementation:0:120:24", nil).Once()
			},
			expectedError: false,
		},
		{
			name:          "上限到達・非アクティブペイン削除",
			maxPanes:      2,
			existingPanes: "0:Plan:0:120:24\n1:Implementation:1:120:24",
			setupMock: func(m *MockCommandExecutor) {
				// ListPanes
				m.On("Execute", "tmux", []string{"list-panes", "-t", "test-session:test-window", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}"}).
					Return("0:Plan:0:120:24\n1:Implementation:1:120:24", nil).Once()

				// KillPane
				m.On("Execute", "tmux", []string{"kill-pane", "-t", "test-session:test-window.0"}).
					Return("", nil).Once()

				// ResizePanesEvenlyWithRetry - ListPanes
				m.On("Execute", "tmux", []string{"list-panes", "-t", "test-session:test-window", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}"}).
					Return("0:Implementation:1:240:24", nil).Once()

				// ResizePanesEvenlyWithRetry - スキップ（ペイン1個のため）
			},
			expectedError: false,
		},
		{
			name:          "デフォルト値使用",
			maxPanes:      0,
			existingPanes: "0:P1:0:60:24\n1:P2:0:60:24\n2:P3:1:60:24\n3:P4:0:60:24",
			setupMock: func(m *MockCommandExecutor) {
				// ListPanes
				m.On("Execute", "tmux", []string{"list-panes", "-t", "test-session:test-window", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}"}).
					Return("0:P1:0:60:24\n1:P2:0:60:24\n2:P3:1:60:24\n3:P4:0:60:24", nil).Once()

				// KillPane (デフォルト値3なので、4個→3個に削減)
				m.On("Execute", "tmux", []string{"kill-pane", "-t", "test-session:test-window.0"}).
					Return("", nil).Once()

				// ResizePanesEvenlyWithRetry - ListPanes
				m.On("Execute", "tmux", []string{"list-panes", "-t", "test-session:test-window", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}"}).
					Return("0:P2:0:80:24\n1:P3:1:80:24\n2:P4:0:80:24", nil).Once()

				// ResizePanesEvenlyWithRetry - GetWindowSize
				m.On("Execute", "tmux", []string{"display-message", "-p", "-t", "test-session:test-window",
					"#{window_width} #{window_height}"}).
					Return("240 24", nil).Once()

				// ResizePanesEvenlyWithRetry - select-layout
				m.On("Execute", "tmux", []string{"select-layout", "-t", "test-session:test-window", "even-horizontal"}).
					Return("", nil).Once()
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := new(MockCommandExecutor)
			tt.setupMock(mockExec)
			defer mockExec.AssertExpectations(t)

			manager := NewDefaultManagerWithExecutor(mockExec)
			err := manager.enforcePaneLimit("test-session", "test-window", tt.maxPanes)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreatePane_Error(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*MockCommandExecutor)
		expectedError string
	}{
		{
			name: "split-window失敗",
			setupMock: func(m *MockCommandExecutor) {
				m.On("Execute", "tmux", []string{"split-window", "-h", "-p", "50", "-t", "test-session:test-window"}).
					Return("", fmt.Errorf("window not found")).Once()
			},
			expectedError: "failed to create pane: window not found",
		},
		{
			name: "ListPanes失敗",
			setupMock: func(m *MockCommandExecutor) {
				m.On("Execute", "tmux", []string{"split-window", "-h", "-p", "50", "-t", "test-session:test-window"}).
					Return("", nil).Once()
				m.On("Execute", "tmux", []string{"list-panes", "-t", "test-session:test-window", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}"}).
					Return("", fmt.Errorf("session not found")).Once()
			},
			expectedError: "failed to list panes after creation: failed to list panes: session not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := new(MockCommandExecutor)
			tt.setupMock(mockExec)
			defer mockExec.AssertExpectations(t)

			manager := NewDefaultManagerWithExecutor(mockExec)
			opts := PaneOptions{
				Split:      "-h",
				Percentage: 50,
			}

			pane, err := manager.CreatePane("test-session", "test-window", opts)

			assert.Error(t, err)
			assert.Nil(t, pane)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

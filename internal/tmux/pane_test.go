package tmux

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCommandExecutor is a mock implementation of CommandExecutor interface
type MockCommandExecutor struct {
	mock.Mock
}

// Execute mocks the Execute method
func (m *MockCommandExecutor) Execute(cmd string, args ...string) (string, error) {
	ret := m.Called(cmd, args)
	return ret.String(0), ret.Error(1)
}

func TestDefaultManager_CreatePane(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		windowName  string
		opts        PaneOptions
		setupMock   func(*MockCommandExecutor)
		want        *PaneInfo
		wantErr     bool
		errMessage  string
	}{
		{
			name:        "create horizontal pane successfully with auto layout",
			sessionName: "osoba-test",
			windowName:  "issue-123",
			opts: PaneOptions{
				Split:      "-h",
				Percentage: 50,
				Title:      "Implementation",
			},
			setupMock: func(m *MockCommandExecutor) {
				// split-window command
				m.On("Execute", "tmux", []string{
					"split-window", "-h", "-p", "50", "-t", "osoba-test:issue-123",
				}).Return("", nil).Once()

				// list-panes to get new pane info
				m.On("Execute", "tmux", []string{
					"list-panes", "-t", "osoba-test:issue-123", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}",
				}).Return("0:Plan:0:80:40\n1:Implementation:1:80:40", nil).Once()

				// set-option for pane title
				m.On("Execute", "tmux", []string{
					"set-option", "-t", "osoba-test:issue-123.1", "-p", "pane-border-format", " Implementation ",
				}).Return("", nil).Once()

				// Auto layout adjustment - list panes for ResizePanesEvenlyWithRetry
				m.On("Execute", "tmux", []string{
					"list-panes", "-t", "osoba-test:issue-123", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}",
				}).Return("0:Plan:0:80:40\n1:Implementation:1:80:40", nil).Once()

				// Get window size for ResizePanesEvenlyWithRetry
				m.On("Execute", "tmux", []string{
					"display-message", "-p", "-t", "osoba-test:issue-123", "#{window_width} #{window_height}",
				}).Return("160 40", nil).Once()

				// Execute select-layout for ResizePanesEvenlyWithRetry
				m.On("Execute", "tmux", []string{
					"select-layout", "-t", "osoba-test:issue-123", "even-horizontal",
				}).Return("", nil).Once()
			},
			want: &PaneInfo{
				Index:  1,
				Title:  "Implementation",
				Active: true,
				Width:  80,
				Height: 40,
			},
			wantErr: false,
		},
		{
			name:        "create horizontal pane with custom percentage and auto layout",
			sessionName: "osoba-test",
			windowName:  "issue-456",
			opts: PaneOptions{
				Split:      "-h",
				Percentage: 30,
				Title:      "Review",
			},
			setupMock: func(m *MockCommandExecutor) {
				// split-window command
				m.On("Execute", "tmux", []string{
					"split-window", "-h", "-p", "30", "-t", "osoba-test:issue-456",
				}).Return("", nil).Once()

				// list-panes to get new pane info
				m.On("Execute", "tmux", []string{
					"list-panes", "-t", "osoba-test:issue-456", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}",
				}).Return("0:Plan:0:56:80\n1:Review:1:24:80", nil).Once()

				// set-option for pane title
				m.On("Execute", "tmux", []string{
					"set-option", "-t", "osoba-test:issue-456.1", "-p", "pane-border-format", " Review ",
				}).Return("", nil).Once()

				// Auto layout adjustment - list panes for ResizePanesEvenlyWithRetry
				m.On("Execute", "tmux", []string{
					"list-panes", "-t", "osoba-test:issue-456", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}",
				}).Return("0:Plan:0:56:80\n1:Review:1:24:80", nil).Once()

				// Get window size for ResizePanesEvenlyWithRetry
				m.On("Execute", "tmux", []string{
					"display-message", "-p", "-t", "osoba-test:issue-456", "#{window_width} #{window_height}",
				}).Return("80 80", nil).Once()

				// Execute select-layout for ResizePanesEvenlyWithRetry
				m.On("Execute", "tmux", []string{
					"select-layout", "-t", "osoba-test:issue-456", "even-horizontal",
				}).Return("", nil).Once()
			},
			want: &PaneInfo{
				Index:  1,
				Title:  "Review",
				Active: true,
				Width:  24,
				Height: 80,
			},
			wantErr: false,
		},
		{
			name:        "create pane successfully even if layout adjustment fails",
			sessionName: "osoba-test",
			windowName:  "issue-789",
			opts: PaneOptions{
				Split:      "-v",
				Percentage: 50,
				Title:      "Debug",
			},
			setupMock: func(m *MockCommandExecutor) {
				// split-window command
				m.On("Execute", "tmux", []string{
					"split-window", "-v", "-p", "50", "-t", "osoba-test:issue-789",
				}).Return("", nil).Once()

				// list-panes to get new pane info
				m.On("Execute", "tmux", []string{
					"list-panes", "-t", "osoba-test:issue-789", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}",
				}).Return("0:Plan:0:160:20\n1:Debug:1:160:20", nil).Once()

				// set-option for pane title
				m.On("Execute", "tmux", []string{
					"set-option", "-t", "osoba-test:issue-789.1", "-p", "pane-border-format", " Debug ",
				}).Return("", nil).Once()

				// Auto layout adjustment - list panes for ResizePanesEvenlyWithRetry
				m.On("Execute", "tmux", []string{
					"list-panes", "-t", "osoba-test:issue-789", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}",
				}).Return("0:Plan:0:160:20\n1:Debug:1:160:20", nil).Once()

				// Get window size for ResizePanesEvenlyWithRetry
				m.On("Execute", "tmux", []string{
					"display-message", "-p", "-t", "osoba-test:issue-789", "#{window_width} #{window_height}",
				}).Return("160 40", nil).Once()

				// Execute select-layout for ResizePanesEvenlyWithRetry - fails but doesn't affect pane creation
				m.On("Execute", "tmux", []string{
					"select-layout", "-t", "osoba-test:issue-789", "even-horizontal",
				}).Return("", fmt.Errorf("layout adjustment error")).Times(3) // Retries 3 times
			},
			want: &PaneInfo{
				Index:  1,
				Title:  "Debug",
				Active: true,
				Width:  160,
				Height: 20,
			},
			wantErr: false, // Pane creation should succeed despite layout error
		},
		{
			name:        "skip layout adjustment for single pane",
			sessionName: "osoba-test",
			windowName:  "issue-single",
			opts: PaneOptions{
				Split:      "-h",
				Percentage: 50,
				Title:      "OnlyPane",
			},
			setupMock: func(m *MockCommandExecutor) {
				// split-window command
				m.On("Execute", "tmux", []string{
					"split-window", "-h", "-p", "50", "-t", "osoba-test:issue-single",
				}).Return("", nil).Once()

				// list-panes to get new pane info - returns only single pane (edge case)
				m.On("Execute", "tmux", []string{
					"list-panes", "-t", "osoba-test:issue-single", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}",
				}).Return("0:OnlyPane:1:160:40", nil).Once()

				// set-option for pane title
				m.On("Execute", "tmux", []string{
					"set-option", "-t", "osoba-test:issue-single.0", "-p", "pane-border-format", " OnlyPane ",
				}).Return("", nil).Once()

				// Auto layout adjustment - list panes for ResizePanesEvenlyWithRetry
				m.On("Execute", "tmux", []string{
					"list-panes", "-t", "osoba-test:issue-single", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}",
				}).Return("0:OnlyPane:1:160:40", nil).Once()

				// No window size check or select-layout since only 1 pane
			},
			want: &PaneInfo{
				Index:  0,
				Title:  "OnlyPane",
				Active: true,
				Width:  160,
				Height: 40,
			},
			wantErr: false,
		},
		{
			name:        "fail to create pane - window does not exist",
			sessionName: "osoba-test",
			windowName:  "non-existent",
			opts: PaneOptions{
				Split: "-h",
				Title: "Test",
			},
			setupMock: func(m *MockCommandExecutor) {
				m.On("Execute", "tmux", []string{
					"split-window", "-h", "-p", "50", "-t", "osoba-test:non-existent",
				}).Return("", fmt.Errorf("can't find window: non-existent")).Once()
			},
			want:       nil,
			wantErr:    true,
			errMessage: "can't find window",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := new(MockCommandExecutor)
			tt.setupMock(mockExecutor)

			manager := &DefaultManager{executor: mockExecutor}

			got, err := manager.CreatePane(tt.sessionName, tt.windowName, tt.opts)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			mockExecutor.AssertExpectations(t)
		})
	}
}

func TestDefaultManager_SelectPane(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		windowName  string
		paneIndex   int
		setupMock   func(*MockCommandExecutor)
		wantErr     bool
		errMessage  string
	}{
		{
			name:        "select pane successfully",
			sessionName: "osoba-test",
			windowName:  "issue-123",
			paneIndex:   1,
			setupMock: func(m *MockCommandExecutor) {
				m.On("Execute", "tmux", []string{
					"select-pane", "-t", "osoba-test:issue-123.1",
				}).Return("", nil).Once()
			},
			wantErr: false,
		},
		{
			name:        "fail to select pane - invalid index",
			sessionName: "osoba-test",
			windowName:  "issue-123",
			paneIndex:   99,
			setupMock: func(m *MockCommandExecutor) {
				m.On("Execute", "tmux", []string{
					"select-pane", "-t", "osoba-test:issue-123.99",
				}).Return("", fmt.Errorf("can't find pane: 99")).Once()
			},
			wantErr:    true,
			errMessage: "can't find pane",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := new(MockCommandExecutor)
			tt.setupMock(mockExecutor)

			manager := &DefaultManager{executor: mockExecutor}

			err := manager.SelectPane(tt.sessionName, tt.windowName, tt.paneIndex)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
			} else {
				assert.NoError(t, err)
			}

			mockExecutor.AssertExpectations(t)
		})
	}
}

func TestDefaultManager_SetPaneTitle(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		windowName  string
		paneIndex   int
		title       string
		setupMock   func(*MockCommandExecutor)
		wantErr     bool
		errMessage  string
	}{
		{
			name:        "set pane title successfully",
			sessionName: "osoba-test",
			windowName:  "issue-123",
			paneIndex:   0,
			title:       "Plan",
			setupMock: func(m *MockCommandExecutor) {
				m.On("Execute", "tmux", []string{
					"set-option", "-t", "osoba-test:issue-123.0", "-p", "pane-border-format", " Plan ",
				}).Return("", nil).Once()
			},
			wantErr: false,
		},
		{
			name:        "fail to set pane title - pane does not exist",
			sessionName: "osoba-test",
			windowName:  "issue-123",
			paneIndex:   99,
			title:       "Test",
			setupMock: func(m *MockCommandExecutor) {
				m.On("Execute", "tmux", []string{
					"set-option", "-t", "osoba-test:issue-123.99", "-p", "pane-border-format", " Test ",
				}).Return("", fmt.Errorf("can't find pane")).Once()
			},
			wantErr:    true,
			errMessage: "can't find pane",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := new(MockCommandExecutor)
			tt.setupMock(mockExecutor)

			manager := &DefaultManager{executor: mockExecutor}

			err := manager.SetPaneTitle(tt.sessionName, tt.windowName, tt.paneIndex, tt.title)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
			} else {
				assert.NoError(t, err)
			}

			mockExecutor.AssertExpectations(t)
		})
	}
}

func TestDefaultManager_ListPanes(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		windowName  string
		setupMock   func(*MockCommandExecutor)
		want        []*PaneInfo
		wantErr     bool
		errMessage  string
	}{
		{
			name:        "list panes successfully with multiple panes",
			sessionName: "osoba-test",
			windowName:  "issue-123",
			setupMock: func(m *MockCommandExecutor) {
				m.On("Execute", "tmux", []string{
					"list-panes", "-t", "osoba-test:issue-123", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}",
				}).Return("0:Plan:1:80:40\n1:Implementation:0:80:40\n2:Review:0:80:20", nil).Once()
			},
			want: []*PaneInfo{
				{Index: 0, Title: "Plan", Active: true, Width: 80, Height: 40},
				{Index: 1, Title: "Implementation", Active: false, Width: 80, Height: 40},
				{Index: 2, Title: "Review", Active: false, Width: 80, Height: 20},
			},
			wantErr: false,
		},
		{
			name:        "list panes successfully with single pane",
			sessionName: "osoba-test",
			windowName:  "issue-456",
			setupMock: func(m *MockCommandExecutor) {
				m.On("Execute", "tmux", []string{
					"list-panes", "-t", "osoba-test:issue-456", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}",
				}).Return("0::1:160:80", nil).Once()
			},
			want: []*PaneInfo{
				{Index: 0, Title: "", Active: true, Width: 160, Height: 80},
			},
			wantErr: false,
		},
		{
			name:        "fail to list panes - window does not exist",
			sessionName: "osoba-test",
			windowName:  "non-existent",
			setupMock: func(m *MockCommandExecutor) {
				m.On("Execute", "tmux", []string{
					"list-panes", "-t", "osoba-test:non-existent", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}",
				}).Return("", fmt.Errorf("can't find window")).Once()
			},
			want:       nil,
			wantErr:    true,
			errMessage: "can't find window",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := new(MockCommandExecutor)
			tt.setupMock(mockExecutor)

			manager := &DefaultManager{executor: mockExecutor}

			got, err := manager.ListPanes(tt.sessionName, tt.windowName)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			mockExecutor.AssertExpectations(t)
		})
	}
}

func TestDefaultManager_GetPaneByTitle(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		windowName  string
		title       string
		setupMock   func(*MockCommandExecutor)
		want        *PaneInfo
		wantErr     bool
		errMessage  string
	}{
		{
			name:        "get pane by title successfully",
			sessionName: "osoba-test",
			windowName:  "issue-123",
			title:       "Implementation",
			setupMock: func(m *MockCommandExecutor) {
				m.On("Execute", "tmux", []string{
					"list-panes", "-t", "osoba-test:issue-123", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}",
				}).Return("0:Plan:0:80:40\n1:Implementation:1:80:40\n2:Review:0:80:20", nil).Once()
			},
			want: &PaneInfo{
				Index:  1,
				Title:  "Implementation",
				Active: true,
				Width:  80,
				Height: 40,
			},
			wantErr: false,
		},
		{
			name:        "fail to get pane - title not found",
			sessionName: "osoba-test",
			windowName:  "issue-123",
			title:       "NonExistent",
			setupMock: func(m *MockCommandExecutor) {
				m.On("Execute", "tmux", []string{
					"list-panes", "-t", "osoba-test:issue-123", "-F",
					"#{pane_index}:#{pane_title}:#{pane_active}:#{pane_width}:#{pane_height}",
				}).Return("0:Plan:1:80:40\n1:Implementation:0:80:40", nil).Once()
			},
			want:       nil,
			wantErr:    true,
			errMessage: "pane with title 'NonExistent' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := new(MockCommandExecutor)
			tt.setupMock(mockExecutor)

			manager := &DefaultManager{executor: mockExecutor}

			got, err := manager.GetPaneByTitle(tt.sessionName, tt.windowName, tt.title)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			mockExecutor.AssertExpectations(t)
		})
	}
}

func TestDefaultManager_KillPane(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		windowName  string
		paneIndex   int
		setupMock   func(*MockCommandExecutor)
		wantErr     bool
		errMessage  string
	}{
		{
			name:        "kill pane successfully",
			sessionName: "osoba-test",
			windowName:  "issue-123",
			paneIndex:   1,
			setupMock: func(m *MockCommandExecutor) {
				m.On("Execute", "tmux", []string{
					"kill-pane", "-t", "osoba-test:issue-123.1",
				}).Return("", nil).Once()
			},
			wantErr: false,
		},
		{
			name:        "fail to kill pane - invalid index",
			sessionName: "osoba-test",
			windowName:  "issue-123",
			paneIndex:   99,
			setupMock: func(m *MockCommandExecutor) {
				m.On("Execute", "tmux", []string{
					"kill-pane", "-t", "osoba-test:issue-123.99",
				}).Return("", fmt.Errorf("pane not found")).Once()
			},
			wantErr:    true,
			errMessage: "failed to kill pane",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := new(MockCommandExecutor)
			tt.setupMock(mockExecutor)

			manager := &DefaultManager{executor: mockExecutor}

			err := manager.KillPane(tt.sessionName, tt.windowName, tt.paneIndex)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
			} else {
				assert.NoError(t, err)
			}

			mockExecutor.AssertExpectations(t)
		})
	}
}

// parsePaneInfo のテスト
func TestParsePaneInfo(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    *PaneInfo
		wantErr bool
	}{
		{
			name: "parse valid pane info",
			line: "0:Plan:1:80:40",
			want: &PaneInfo{
				Index:  0,
				Title:  "Plan",
				Active: true,
				Width:  80,
				Height: 40,
			},
			wantErr: false,
		},
		{
			name: "parse pane info with empty title",
			line: "1::0:160:80",
			want: &PaneInfo{
				Index:  1,
				Title:  "",
				Active: false,
				Width:  160,
				Height: 80,
			},
			wantErr: false,
		},
		{
			name:    "invalid format - too few fields",
			line:    "0:Plan:1:80",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid format - non-numeric index",
			line:    "abc:Plan:1:80:40",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid format - non-numeric active",
			line:    "0:Plan:yes:80:40",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid format - non-numeric width",
			line:    "0:Plan:1:wide:40",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid format - non-numeric height",
			line:    "0:Plan:1:80:tall",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePaneInfo(tt.line)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

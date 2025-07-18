package tmux

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPaneManager struct {
	mock.Mock
}

func (m *MockPaneManager) CreatePane(sessionName, windowName string, opts PaneOptions) (*PaneInfo, error) {
	args := m.Called(sessionName, windowName, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PaneInfo), args.Error(1)
}

func (m *MockPaneManager) SelectPane(sessionName, windowName string, paneIndex int) error {
	args := m.Called(sessionName, windowName, paneIndex)
	return args.Error(0)
}

func (m *MockPaneManager) SetPaneTitle(sessionName, windowName string, paneIndex int, title string) error {
	args := m.Called(sessionName, windowName, paneIndex, title)
	return args.Error(0)
}

func (m *MockPaneManager) ListPanes(sessionName, windowName string) ([]*PaneInfo, error) {
	args := m.Called(sessionName, windowName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*PaneInfo), args.Error(1)
}

func (m *MockPaneManager) GetPaneByTitle(sessionName, windowName string, title string) (*PaneInfo, error) {
	args := m.Called(sessionName, windowName, title)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PaneInfo), args.Error(1)
}

// Test cases for PaneManager interface
func TestPaneManager_CreatePane(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		windowName  string
		opts        PaneOptions
		mockSetup   func(*MockPaneManager)
		want        *PaneInfo
		wantErr     bool
		errMessage  string
	}{
		{
			name:        "create vertical pane successfully",
			sessionName: "osoba-test",
			windowName:  "issue-123",
			opts: PaneOptions{
				Split:      "-v",
				Percentage: 50,
				Title:      "Implementation",
			},
			mockSetup: func(m *MockPaneManager) {
				m.On("CreatePane", "osoba-test", "issue-123", PaneOptions{
					Split:      "-v",
					Percentage: 50,
					Title:      "Implementation",
				}).Return(&PaneInfo{
					Index:  1,
					Title:  "Implementation",
					Active: true,
					Width:  80,
					Height: 40,
				}, nil)
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
			name:        "create horizontal pane successfully",
			sessionName: "osoba-test",
			windowName:  "issue-123",
			opts: PaneOptions{
				Split:      "-h",
				Percentage: 30,
				Title:      "Review",
			},
			mockSetup: func(m *MockPaneManager) {
				m.On("CreatePane", "osoba-test", "issue-123", PaneOptions{
					Split:      "-h",
					Percentage: 30,
					Title:      "Review",
				}).Return(&PaneInfo{
					Index:  2,
					Title:  "Review",
					Active: true,
					Width:  40,
					Height: 80,
				}, nil)
			},
			want: &PaneInfo{
				Index:  2,
				Title:  "Review",
				Active: true,
				Width:  40,
				Height: 80,
			},
			wantErr: false,
		},
		{
			name:        "fail to create pane - window does not exist",
			sessionName: "osoba-test",
			windowName:  "non-existent",
			opts: PaneOptions{
				Split: "-v",
				Title: "Test",
			},
			mockSetup: func(m *MockPaneManager) {
				m.On("CreatePane", "osoba-test", "non-existent", PaneOptions{
					Split: "-v",
					Title: "Test",
				}).Return(nil, fmt.Errorf("window not found"))
			},
			want:       nil,
			wantErr:    true,
			errMessage: "window not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockPaneManager)
			tt.mockSetup(mockManager)

			got, err := mockManager.CreatePane(tt.sessionName, tt.windowName, tt.opts)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			mockManager.AssertExpectations(t)
		})
	}
}

func TestPaneManager_SelectPane(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		windowName  string
		paneIndex   int
		mockSetup   func(*MockPaneManager)
		wantErr     bool
		errMessage  string
	}{
		{
			name:        "select pane successfully",
			sessionName: "osoba-test",
			windowName:  "issue-123",
			paneIndex:   1,
			mockSetup: func(m *MockPaneManager) {
				m.On("SelectPane", "osoba-test", "issue-123", 1).Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "fail to select pane - invalid index",
			sessionName: "osoba-test",
			windowName:  "issue-123",
			paneIndex:   99,
			mockSetup: func(m *MockPaneManager) {
				m.On("SelectPane", "osoba-test", "issue-123", 99).Return(fmt.Errorf("pane index out of range"))
			},
			wantErr:    true,
			errMessage: "pane index out of range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockPaneManager)
			tt.mockSetup(mockManager)

			err := mockManager.SelectPane(tt.sessionName, tt.windowName, tt.paneIndex)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
			} else {
				assert.NoError(t, err)
			}

			mockManager.AssertExpectations(t)
		})
	}
}

func TestPaneManager_SetPaneTitle(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		windowName  string
		paneIndex   int
		title       string
		mockSetup   func(*MockPaneManager)
		wantErr     bool
		errMessage  string
	}{
		{
			name:        "set pane title successfully",
			sessionName: "osoba-test",
			windowName:  "issue-123",
			paneIndex:   0,
			title:       "Plan",
			mockSetup: func(m *MockPaneManager) {
				m.On("SetPaneTitle", "osoba-test", "issue-123", 0, "Plan").Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "fail to set pane title - pane does not exist",
			sessionName: "osoba-test",
			windowName:  "issue-123",
			paneIndex:   99,
			title:       "Test",
			mockSetup: func(m *MockPaneManager) {
				m.On("SetPaneTitle", "osoba-test", "issue-123", 99, "Test").Return(fmt.Errorf("pane not found"))
			},
			wantErr:    true,
			errMessage: "pane not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockPaneManager)
			tt.mockSetup(mockManager)

			err := mockManager.SetPaneTitle(tt.sessionName, tt.windowName, tt.paneIndex, tt.title)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
			} else {
				assert.NoError(t, err)
			}

			mockManager.AssertExpectations(t)
		})
	}
}

func TestPaneManager_ListPanes(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		windowName  string
		mockSetup   func(*MockPaneManager)
		want        []*PaneInfo
		wantErr     bool
		errMessage  string
	}{
		{
			name:        "list panes successfully",
			sessionName: "osoba-test",
			windowName:  "issue-123",
			mockSetup: func(m *MockPaneManager) {
				m.On("ListPanes", "osoba-test", "issue-123").Return([]*PaneInfo{
					{
						Index:  0,
						Title:  "Plan",
						Active: true,
						Width:  80,
						Height: 50,
					},
					{
						Index:  1,
						Title:  "Implementation",
						Active: false,
						Width:  80,
						Height: 50,
					},
				}, nil)
			},
			want: []*PaneInfo{
				{
					Index:  0,
					Title:  "Plan",
					Active: true,
					Width:  80,
					Height: 50,
				},
				{
					Index:  1,
					Title:  "Implementation",
					Active: false,
					Width:  80,
					Height: 50,
				},
			},
			wantErr: false,
		},
		{
			name:        "fail to list panes - window does not exist",
			sessionName: "osoba-test",
			windowName:  "non-existent",
			mockSetup: func(m *MockPaneManager) {
				m.On("ListPanes", "osoba-test", "non-existent").Return(nil, fmt.Errorf("window not found"))
			},
			want:       nil,
			wantErr:    true,
			errMessage: "window not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockPaneManager)
			tt.mockSetup(mockManager)

			got, err := mockManager.ListPanes(tt.sessionName, tt.windowName)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			mockManager.AssertExpectations(t)
		})
	}
}

func TestPaneManager_GetPaneByTitle(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		windowName  string
		title       string
		mockSetup   func(*MockPaneManager)
		want        *PaneInfo
		wantErr     bool
		errMessage  string
	}{
		{
			name:        "get pane by title successfully",
			sessionName: "osoba-test",
			windowName:  "issue-123",
			title:       "Implementation",
			mockSetup: func(m *MockPaneManager) {
				m.On("GetPaneByTitle", "osoba-test", "issue-123", "Implementation").Return(&PaneInfo{
					Index:  1,
					Title:  "Implementation",
					Active: false,
					Width:  80,
					Height: 50,
				}, nil)
			},
			want: &PaneInfo{
				Index:  1,
				Title:  "Implementation",
				Active: false,
				Width:  80,
				Height: 50,
			},
			wantErr: false,
		},
		{
			name:        "fail to get pane - title not found",
			sessionName: "osoba-test",
			windowName:  "issue-123",
			title:       "NonExistent",
			mockSetup: func(m *MockPaneManager) {
				m.On("GetPaneByTitle", "osoba-test", "issue-123", "NonExistent").Return(nil, fmt.Errorf("pane with title not found"))
			},
			want:       nil,
			wantErr:    true,
			errMessage: "pane with title not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockPaneManager)
			tt.mockSetup(mockManager)

			got, err := mockManager.GetPaneByTitle(tt.sessionName, tt.windowName, tt.title)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			mockManager.AssertExpectations(t)
		})
	}
}

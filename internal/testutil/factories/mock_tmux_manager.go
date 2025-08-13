package factories

import (
	"fmt"
	"strings"
	"sync"

	"github.com/douhashi/osoba/internal/tmux"
)

// MockTmuxManager is a mock implementation of tmux.Manager for testing.
type MockTmuxManager struct {
	mu            sync.RWMutex
	sessions      map[string]*MockSession
	paneBaseIndex int
	errors        map[string]error // For simulating errors
}

// MockSession represents a mock tmux session.
type MockSession struct {
	Name    string
	Windows map[string]*MockWindow
}

// MockWindow represents a mock tmux window.
type MockWindow struct {
	Name  string
	Panes []MockPane
	Keys  []string // Stored keys sent to window
}

// MockPane represents a mock tmux pane.
type MockPane struct {
	Index   int
	Command string
	Keys    []string
}

// NewMockTmuxManager creates a new mock tmux manager.
func NewMockTmuxManager() *MockTmuxManager {
	return &MockTmuxManager{
		sessions:      make(map[string]*MockSession),
		paneBaseIndex: 0,
		errors:        make(map[string]error),
	}
}

// SetError sets an error to be returned for a specific method.
func (m *MockTmuxManager) SetError(method string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[method] = err
}

// ClearError clears an error for a specific method.
func (m *MockTmuxManager) ClearError(method string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.errors, method)
}

// getError returns any configured error for a method.
func (m *MockTmuxManager) getError(method string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.errors[method]
}

// CheckTmuxInstalled checks if tmux is installed (always returns nil for mock).
func (m *MockTmuxManager) CheckTmuxInstalled() error {
	if err := m.getError("CheckTmuxInstalled"); err != nil {
		return err
	}
	return nil
}

// SessionExists checks if a session exists.
func (m *MockTmuxManager) SessionExists(sessionName string) (bool, error) {
	if err := m.getError("SessionExists"); err != nil {
		return false, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.sessions[sessionName]
	return exists, nil
}

// CreateSession creates a new session.
func (m *MockTmuxManager) CreateSession(sessionName string) error {
	if err := m.getError("CreateSession"); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[sessionName]; exists {
		return fmt.Errorf("session %s already exists", sessionName)
	}

	m.sessions[sessionName] = &MockSession{
		Name:    sessionName,
		Windows: make(map[string]*MockWindow),
	}
	return nil
}

// EnsureSession ensures a session exists.
func (m *MockTmuxManager) EnsureSession(sessionName string) error {
	if err := m.getError("EnsureSession"); err != nil {
		return err
	}

	exists, _ := m.SessionExists(sessionName)
	if !exists {
		return m.CreateSession(sessionName)
	}
	return nil
}

// ListSessions lists sessions with the given prefix.
func (m *MockTmuxManager) ListSessions(prefix string) ([]string, error) {
	if err := m.getError("ListSessions"); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var sessions []string
	for name := range m.sessions {
		if prefix == "" || strings.HasPrefix(name, prefix) {
			sessions = append(sessions, name)
		}
	}
	return sessions, nil
}

// CreateWindow creates a new window.
func (m *MockTmuxManager) CreateWindow(sessionName, windowName string) error {
	if err := m.getError("CreateWindow"); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionName]
	if !exists {
		return fmt.Errorf("session %s does not exist", sessionName)
	}

	if _, exists := session.Windows[windowName]; exists {
		return fmt.Errorf("window %s already exists", windowName)
	}

	session.Windows[windowName] = &MockWindow{
		Name:  windowName,
		Panes: []MockPane{{Index: m.paneBaseIndex}},
		Keys:  []string{},
	}
	return nil
}

// SwitchToWindow switches to a window.
func (m *MockTmuxManager) SwitchToWindow(sessionName, windowName string) error {
	if err := m.getError("SwitchToWindow"); err != nil {
		return err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionName]
	if !exists {
		return fmt.Errorf("session %s does not exist", sessionName)
	}

	if _, exists := session.Windows[windowName]; !exists {
		return fmt.Errorf("window %s does not exist", windowName)
	}

	return nil
}

// WindowExists checks if a window exists.
func (m *MockTmuxManager) WindowExists(sessionName, windowName string) (bool, error) {
	if err := m.getError("WindowExists"); err != nil {
		return false, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionName]
	if !exists {
		return false, nil
	}

	_, exists = session.Windows[windowName]
	return exists, nil
}

// KillWindow kills a window.
func (m *MockTmuxManager) KillWindow(sessionName, windowName string) error {
	if err := m.getError("KillWindow"); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionName]
	if !exists {
		return fmt.Errorf("session %s does not exist", sessionName)
	}

	delete(session.Windows, windowName)
	return nil
}

// CreateOrReplaceWindow creates or replaces a window.
func (m *MockTmuxManager) CreateOrReplaceWindow(sessionName, windowName string) error {
	if err := m.getError("CreateOrReplaceWindow"); err != nil {
		return err
	}

	exists, _ := m.WindowExists(sessionName, windowName)
	if exists {
		if err := m.KillWindow(sessionName, windowName); err != nil {
			return err
		}
	}
	return m.CreateWindow(sessionName, windowName)
}

// ListWindows lists windows in a session.
func (m *MockTmuxManager) ListWindows(sessionName string) ([]string, error) {
	if err := m.getError("ListWindows"); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionName]
	if !exists {
		return nil, fmt.Errorf("session %s does not exist", sessionName)
	}

	var windows []string
	for name := range session.Windows {
		windows = append(windows, name)
	}
	return windows, nil
}

// SendKeys sends keys to a window.
func (m *MockTmuxManager) SendKeys(sessionName, windowName, keys string) error {
	if err := m.getError("SendKeys"); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionName]
	if !exists {
		return fmt.Errorf("session %s does not exist", sessionName)
	}

	window, exists := session.Windows[windowName]
	if !exists {
		return fmt.Errorf("window %s does not exist", windowName)
	}

	window.Keys = append(window.Keys, keys)
	return nil
}

// ClearWindow clears a window.
func (m *MockTmuxManager) ClearWindow(sessionName, windowName string) error {
	if err := m.getError("ClearWindow"); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionName]
	if !exists {
		return fmt.Errorf("session %s does not exist", sessionName)
	}

	window, exists := session.Windows[windowName]
	if !exists {
		return fmt.Errorf("window %s does not exist", windowName)
	}

	window.Keys = []string{}
	return nil
}

// RunInWindow runs a command in a window.
func (m *MockTmuxManager) RunInWindow(sessionName, windowName, command string) error {
	if err := m.getError("RunInWindow"); err != nil {
		return err
	}

	// Simulate running command by sending it as keys
	return m.SendKeys(sessionName, windowName, command+" Enter")
}

// GetIssueWindow returns the window name for an issue.
func (m *MockTmuxManager) GetIssueWindow(issueNumber int) string {
	return fmt.Sprintf("issue-%d", issueNumber)
}

// MatchIssueWindow checks if a window name matches the issue pattern.
func (m *MockTmuxManager) MatchIssueWindow(windowName string) bool {
	return strings.HasPrefix(windowName, "issue-")
}

// FindIssueWindow extracts the issue number from a window name.
func (m *MockTmuxManager) FindIssueWindow(windowName string) (int, bool) {
	if !m.MatchIssueWindow(windowName) {
		return 0, false
	}

	var issueNumber int
	_, err := fmt.Sscanf(windowName, "issue-%d", &issueNumber)
	return issueNumber, err == nil
}

// CreateWindowForIssueWithNewWindowDetection creates a window for an issue.
func (m *MockTmuxManager) CreateWindowForIssueWithNewWindowDetection(sessionName string, issueNumber int) (string, bool, error) {
	if err := m.getError("CreateWindowForIssueWithNewWindowDetection"); err != nil {
		return "", false, err
	}

	windowName := m.GetIssueWindow(issueNumber)
	exists, _ := m.WindowExists(sessionName, windowName)

	if err := m.CreateOrReplaceWindow(sessionName, windowName); err != nil {
		return "", false, err
	}

	return windowName, !exists, nil
}

// CreatePane creates a new pane.
func (m *MockTmuxManager) CreatePane(sessionName, windowName string, opts tmux.PaneOptions) (*tmux.PaneInfo, error) {
	if err := m.getError("CreatePane"); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionName]
	if !exists {
		return nil, fmt.Errorf("session %s does not exist", sessionName)
	}

	window, exists := session.Windows[windowName]
	if !exists {
		return nil, fmt.Errorf("window %s does not exist", windowName)
	}

	newIndex := m.paneBaseIndex + len(window.Panes)
	window.Panes = append(window.Panes, MockPane{Index: newIndex})

	return &tmux.PaneInfo{
		Index:  newIndex,
		Title:  opts.Title,
		Active: true,
		Width:  80, // default mock width
		Height: 24, // default mock height
	}, nil
}

// SplitPane splits a pane.
func (m *MockTmuxManager) SplitPane(sessionName, windowName string, paneIndex int, vertical bool, percentage int) (int, error) {
	if err := m.getError("SplitPane"); err != nil {
		return 0, err
	}

	opts := tmux.PaneOptions{
		Split:      "-v",
		Percentage: percentage,
		Title:      "",
	}
	if !vertical {
		opts.Split = "-h"
	}

	info, err := m.CreatePane(sessionName, windowName, opts)
	if err != nil {
		return 0, err
	}
	return info.Index, nil
}

// SendKeysToPane sends keys to a specific pane.
func (m *MockTmuxManager) SendKeysToPane(sessionName, windowName string, paneIndex int, keys string) error {
	if err := m.getError("SendKeysToPane"); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionName]
	if !exists {
		return fmt.Errorf("session %s does not exist", sessionName)
	}

	window, exists := session.Windows[windowName]
	if !exists {
		return fmt.Errorf("window %s does not exist", windowName)
	}

	for i := range window.Panes {
		if window.Panes[i].Index == paneIndex {
			window.Panes[i].Keys = append(window.Panes[i].Keys, keys)
			return nil
		}
	}

	return fmt.Errorf("pane %d does not exist", paneIndex)
}

// RunInPane runs a command in a specific pane.
func (m *MockTmuxManager) RunInPane(sessionName, windowName string, paneIndex int, command string) error {
	if err := m.getError("RunInPane"); err != nil {
		return err
	}

	return m.SendKeysToPane(sessionName, windowName, paneIndex, command+" Enter")
}

// GetPaneBaseIndex returns the pane base index.
func (m *MockTmuxManager) GetPaneBaseIndex() (int, error) {
	if err := m.getError("GetPaneBaseIndex"); err != nil {
		return 0, err
	}

	return m.paneBaseIndex, nil
}

// GetSessions returns all mock sessions (for testing).
func (m *MockTmuxManager) GetSessions() map[string]*MockSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid race conditions
	sessions := make(map[string]*MockSession)
	for k, v := range m.sessions {
		sessions[k] = v
	}
	return sessions
}

// SelectPane selects a specific pane.
func (m *MockTmuxManager) SelectPane(sessionName, windowName string, paneIndex int) error {
	if err := m.getError("SelectPane"); err != nil {
		return err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionName]
	if !exists {
		return fmt.Errorf("session %s does not exist", sessionName)
	}

	window, exists := session.Windows[windowName]
	if !exists {
		return fmt.Errorf("window %s does not exist", windowName)
	}

	for _, pane := range window.Panes {
		if pane.Index == paneIndex {
			return nil
		}
	}

	return fmt.Errorf("pane %d does not exist", paneIndex)
}

// SetPaneTitle sets the title of a pane.
func (m *MockTmuxManager) SetPaneTitle(sessionName, windowName string, paneIndex int, title string) error {
	if err := m.getError("SetPaneTitle"); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionName]
	if !exists {
		return fmt.Errorf("session %s does not exist", sessionName)
	}

	window, exists := session.Windows[windowName]
	if !exists {
		return fmt.Errorf("window %s does not exist", windowName)
	}

	for i := range window.Panes {
		if window.Panes[i].Index == paneIndex {
			// Note: MockPane doesn't have a Title field, but we can simulate success
			return nil
		}
	}

	return fmt.Errorf("pane %d does not exist", paneIndex)
}

// ListPanes lists all panes in a window.
func (m *MockTmuxManager) ListPanes(sessionName, windowName string) ([]*tmux.PaneInfo, error) {
	if err := m.getError("ListPanes"); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionName]
	if !exists {
		return nil, fmt.Errorf("session %s does not exist", sessionName)
	}

	window, exists := session.Windows[windowName]
	if !exists {
		return nil, fmt.Errorf("window %s does not exist", windowName)
	}

	var panes []*tmux.PaneInfo
	for _, pane := range window.Panes {
		panes = append(panes, &tmux.PaneInfo{
			Index:  pane.Index,
			Title:  "",              // MockPane doesn't have a Title field
			Active: pane.Index == 0, // First pane is active
			Width:  80,
			Height: 24,
		})
	}

	return panes, nil
}

// GetPaneByTitle finds a pane by its title.
func (m *MockTmuxManager) GetPaneByTitle(sessionName, windowName string, title string) (*tmux.PaneInfo, error) {
	if err := m.getError("GetPaneByTitle"); err != nil {
		return nil, err
	}

	// For mock purposes, we'll return a default pane info based on title
	// Since MockPane doesn't have a Title field, we'll simulate based on common titles
	knownTitles := map[string]int{
		"Plan":           0,
		"Implementation": 1,
		"Review":         2,
		"Test":           3,
	}

	if index, exists := knownTitles[title]; exists {
		return &tmux.PaneInfo{
			Index:  index,
			Title:  title,
			Active: index == 0,
			Width:  80,
			Height: 24,
		}, nil
	}

	return nil, fmt.Errorf("pane with title %s not found", title)
}

// Reset clears all mock data.
func (m *MockTmuxManager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions = make(map[string]*MockSession)
	m.errors = make(map[string]error)
}

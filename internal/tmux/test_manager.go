package tmux

import (
	"fmt"
	"os"
	"strings"
)

// TestManager is a tmux manager specifically designed for test environments.
// It provides socket isolation and automatic cleanup for test sessions.
type TestManager struct {
	*DefaultManager
	testSocket    string
	sessionPrefix string
	isTestMode    bool
}

// NewTestManager creates a new test-specific tmux manager.
func NewTestManager() *TestManager {
	executor := &DefaultCommandExecutor{}
	
	// Check for test mode environment variables
	testMode := os.Getenv("OSOBA_TEST_MODE") == "true"
	testSocket := os.Getenv("OSOBA_TEST_SOCKET")
	sessionPrefix := os.Getenv("OSOBA_TEST_SESSION_PREFIX")
	
	// Default session prefix for tests
	if sessionPrefix == "" {
		sessionPrefix = "test-osoba-"
	}
	
	// Create test-specific executor if socket isolation is enabled
	var testExecutor CommandExecutor = executor
	if testSocket != "" {
		testExecutor = &TestCommandExecutor{
			base:       executor,
			testSocket: testSocket,
		}
	}
	
	return &TestManager{
		DefaultManager: NewDefaultManagerWithExecutor(testExecutor),
		testSocket:     testSocket,
		sessionPrefix:  sessionPrefix,
		isTestMode:     testMode,
	}
}

// NewTestManagerWithSocket creates a test manager with explicit socket configuration.
func NewTestManagerWithSocket(socket string, prefix string) *TestManager {
	executor := &TestCommandExecutor{
		base:       &DefaultCommandExecutor{},
		testSocket: socket,
	}
	
	return &TestManager{
		DefaultManager: NewDefaultManagerWithExecutor(executor),
		testSocket:     socket,
		sessionPrefix:  prefix,
		isTestMode:     true,
	}
}

// IsTestMode returns true if running in test mode.
func (m *TestManager) IsTestMode() bool {
	return m.isTestMode
}

// GetTestSocket returns the test-specific tmux socket path.
func (m *TestManager) GetTestSocket() string {
	return m.testSocket
}

// GetSessionPrefix returns the session prefix for test sessions.
func (m *TestManager) GetSessionPrefix() string {
	return m.sessionPrefix
}

// EnsureTestSession ensures a test session exists with proper isolation.
func (m *TestManager) EnsureTestSession(sessionName string) error {
	// Add test prefix if not already present
	if !strings.HasPrefix(sessionName, m.sessionPrefix) {
		sessionName = m.sessionPrefix + sessionName
	}
	
	return m.EnsureSession(sessionName)
}

// CreateTestSession creates a new test session with proper isolation.
func (m *TestManager) CreateTestSession(sessionName string) error {
	// Add test prefix if not already present
	if !strings.HasPrefix(sessionName, m.sessionPrefix) {
		sessionName = m.sessionPrefix + sessionName
	}
	
	return m.CreateSession(sessionName)
}

// CleanupTestSessions removes all test sessions with the configured prefix.
func (m *TestManager) CleanupTestSessions() error {
	sessions, err := m.ListSessions(m.sessionPrefix)
	if err != nil {
		return fmt.Errorf("failed to list test sessions: %w", err)
	}
	
	for _, session := range sessions {
		if err := m.KillSession(session); err != nil {
			// Log error but continue cleanup
			if logger := GetLogger(); logger != nil {
				logger.Warn("Failed to kill test session", "session", session, "error", err)
			}
		}
	}
	
	return nil
}

// KillSession kills a tmux session.
func (m *TestManager) KillSession(sessionName string) error {
	_, err := m.executor.Execute("tmux", "kill-session", "-t", sessionName)
	return err
}

// TestCommandExecutor is a command executor that uses a test-specific tmux socket.
type TestCommandExecutor struct {
	base       CommandExecutor
	testSocket string
}

// Execute runs a command with test-specific tmux socket if applicable.
func (e *TestCommandExecutor) Execute(command string, args ...string) (string, error) {
	// Intercept tmux commands and add socket option
	if command == "tmux" && e.testSocket != "" {
		// Insert -S socket option after tmux command
		newArgs := make([]string, 0, len(args)+2)
		newArgs = append(newArgs, "-S", e.testSocket)
		newArgs = append(newArgs, args...)
		return e.base.Execute(command, newArgs...)
	}
	
	return e.base.Execute(command, args...)
}

// ValidateTestEnvironment checks if the test environment is properly configured.
func ValidateTestEnvironment() error {
	if os.Getenv("OSOBA_TEST_MODE") != "true" {
		return fmt.Errorf("OSOBA_TEST_MODE is not set to true")
	}
	
	// Check if test socket is accessible if specified
	if socket := os.Getenv("OSOBA_TEST_SOCKET"); socket != "" {
		// Try to create a test tmux server with the socket
		executor := &TestCommandExecutor{
			base:       &DefaultCommandExecutor{},
			testSocket: socket,
		}
		
		// Start tmux server if not running
		_, _ = executor.Execute("tmux", "start-server")
		
		// Try to list sessions (this will fail if socket is not accessible)
		if _, err := executor.Execute("tmux", "list-sessions"); err != nil {
			// It's okay if there are no sessions, as long as the command doesn't fail due to socket issues
			errStr := err.Error()
			if !strings.Contains(errStr, "no server running") && !strings.Contains(errStr, "no sessions") {
				return fmt.Errorf("test tmux socket not accessible: %w", err)
			}
		}
	}
	
	return nil
}

// CreateIsolatedTestManager creates a fully isolated test manager with a unique socket.
func CreateIsolatedTestManager(testID string) (*TestManager, func(), error) {
	// Generate unique socket path
	socketPath := fmt.Sprintf("/tmp/osoba-test-%s-%d.sock", testID, os.Getpid())
	
	// Create test manager
	manager := NewTestManagerWithSocket(socketPath, "test-"+testID+"-")
	
	// Start tmux server with the test socket
	if testExec, ok := manager.executor.(*TestCommandExecutor); ok {
		if _, err := testExec.Execute("tmux", "start-server"); err != nil {
			return nil, nil, fmt.Errorf("failed to start test tmux server: %w", err)
		}
	}
	
	// Cleanup function
	cleanup := func() {
		// Kill all test sessions
		_ = manager.CleanupTestSessions()
		
		// Kill the tmux server
		if testExec, ok := manager.executor.(*TestCommandExecutor); ok {
			_, _ = testExec.Execute("tmux", "kill-server")
		}
		
		// Remove socket file
		_ = os.Remove(socketPath)
	}
	
	return manager, cleanup, nil
}
package tmux

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// MockConflictManager for testing conflict detector
type MockConflictManager struct {
	sessions map[string]bool
}

func NewMockConflictManager() *MockConflictManager {
	return &MockConflictManager{
		sessions: make(map[string]bool),
	}
}

func (m *MockConflictManager) CheckTmuxInstalled() error {
	return nil
}

func (m *MockConflictManager) SessionExists(sessionName string) (bool, error) {
	return m.sessions[sessionName], nil
}

func (m *MockConflictManager) CreateSession(sessionName string) error {
	m.sessions[sessionName] = true
	return nil
}

func (m *MockConflictManager) EnsureSession(sessionName string) error {
	m.sessions[sessionName] = true
	return nil
}

func (m *MockConflictManager) ListSessions(prefix string) ([]string, error) {
	var result []string
	for session := range m.sessions {
		if prefix == "" || (len(session) >= len(prefix) && session[:len(prefix)] == prefix) {
			result = append(result, session)
		}
	}
	return result, nil
}

// Implement remaining methods to satisfy the Manager interface
func (m *MockConflictManager) CreateWindow(sessionName, windowName string) error   { return nil }
func (m *MockConflictManager) SwitchToWindow(sessionName, windowName string) error { return nil }
func (m *MockConflictManager) WindowExists(sessionName, windowName string) (bool, error) {
	return false, nil
}
func (m *MockConflictManager) KillWindow(sessionName, windowName string) error            { return nil }
func (m *MockConflictManager) CreateOrReplaceWindow(sessionName, windowName string) error { return nil }
func (m *MockConflictManager) ListWindows(sessionName string) ([]string, error)           { return nil, nil }
func (m *MockConflictManager) SendKeys(sessionName, windowName, keys string) error        { return nil }
func (m *MockConflictManager) ClearWindow(sessionName, windowName string) error           { return nil }
func (m *MockConflictManager) RunInWindow(sessionName, windowName, command string) error  { return nil }
func (m *MockConflictManager) GetIssueWindow(issueNumber int) string                      { return "" }
func (m *MockConflictManager) MatchIssueWindow(windowName string) bool                    { return false }
func (m *MockConflictManager) FindIssueWindow(windowName string) (int, bool)              { return 0, false }
func (m *MockConflictManager) CreateWindowForIssueWithNewWindowDetection(sessionName string, issueNumber int) (string, bool, error) {
	return "", false, nil
}
func (m *MockConflictManager) CreatePane(sessionName, windowName string, options PaneOptions) (*PaneInfo, error) {
	return &PaneInfo{Index: 0}, nil
}
func (m *MockConflictManager) SplitPane(sessionName, windowName string, paneIndex int, vertical bool, percentage int) (int, error) {
	return 0, nil
}
func (m *MockConflictManager) SendKeysToPane(sessionName, windowName string, paneIndex int, keys string) error {
	return nil
}
func (m *MockConflictManager) RunInPane(sessionName, windowName string, paneIndex int, command string) error {
	return nil
}
func (m *MockConflictManager) GetPaneBaseIndex() (int, error) { return 0, nil }
func (m *MockConflictManager) GetPaneByTitle(sessionName, windowName, title string) (*PaneInfo, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockConflictManager) ListPanes(sessionName, windowName string) ([]*PaneInfo, error) {
	return []*PaneInfo{}, nil
}
func (m *MockConflictManager) SelectPane(sessionName, windowName string, paneIndex int) error {
	return nil
}
func (m *MockConflictManager) SetPaneTitle(sessionName, windowName string, paneIndex int, title string) error {
	return nil
}
func (m *MockConflictManager) ResizePanesEvenly(sessionName, windowName string) error {
	return nil
}
func (m *MockConflictManager) ResizePanesEvenlyWithRetry(sessionName, windowName string) error {
	return nil
}
func (m *MockConflictManager) GetWindowSize(sessionName, windowName string) (width, height int, err error) {
	return 120, 40, nil
}
func (m *MockConflictManager) KillPane(sessionName, windowName string, paneIndex int) error {
	return nil
}

// DiagnosticManager methods
func (m *MockConflictManager) DiagnoseSession(sessionName string) (*SessionDiagnostics, error) {
	return &SessionDiagnostics{
		Name:      sessionName,
		Windows:   1,
		Attached:  false,
		Created:   "1641641600",
		Errors:    []string{},
		Metadata:  map[string]string{"exists": "true", "mock": "true"},
		Timestamp: time.Now(),
	}, nil
}

func (m *MockConflictManager) DiagnoseWindow(sessionName, windowName string) (*WindowDiagnostics, error) {
	return &WindowDiagnostics{
		Name:        windowName,
		SessionName: sessionName,
		Index:       0,
		Exists:      true,
		Active:      false,
		Panes:       1,
		IssueNumber: 0,
		Phase:       "",
		Errors:      []string{},
		Metadata:    map[string]string{"exists": "true", "mock": "true"},
		Timestamp:   time.Now(),
	}, nil
}

func (m *MockConflictManager) ListSessionDiagnostics(prefix string) ([]*SessionDiagnostics, error) {
	diagnostics := []*SessionDiagnostics{}
	for sessionName := range m.sessions {
		if prefix == "" || len(sessionName) == 0 || (len(sessionName) >= len(prefix) && sessionName[:len(prefix)] == prefix) {
			diagnostics = append(diagnostics, &SessionDiagnostics{
				Name:      sessionName,
				Windows:   1,
				Attached:  false,
				Created:   "1641641600",
				Errors:    []string{},
				Metadata:  map[string]string{"exists": "true", "mock": "true"},
				Timestamp: time.Now(),
			})
		}
	}
	return diagnostics, nil
}

func (m *MockConflictManager) ListWindowDiagnostics(sessionName string) ([]*WindowDiagnostics, error) {
	return []*WindowDiagnostics{
		{
			Name:        "mock-window",
			SessionName: sessionName,
			Index:       0,
			Exists:      true,
			Active:      false,
			Panes:       1,
			IssueNumber: 0,
			Phase:       "",
			Errors:      []string{},
			Metadata:    map[string]string{"exists": "true", "mock": "true"},
			Timestamp:   time.Now(),
		},
	}, nil
}

func TestConflictDetector_CheckSessionConflict(t *testing.T) {
	tests := []struct {
		name             string
		sessionName      string
		existingSessions map[string]bool
		testMode         bool
		expectError      bool
	}{
		{
			name:             "no conflict for new session",
			sessionName:      "test-osoba-123",
			existingSessions: map[string]bool{},
			testMode:         true,
			expectError:      false,
		},
		{
			name:        "conflict when session exists",
			sessionName: "test-osoba-123",
			existingSessions: map[string]bool{
				"test-osoba-123": true,
			},
			testMode:    true,
			expectError: false, // Same mode, no conflict
		},
		{
			name:        "conflict for production session in test mode",
			sessionName: "osoba-prod",
			existingSessions: map[string]bool{
				"osoba-prod": true,
			},
			testMode:    true,
			expectError: true,
		},
		{
			name:        "conflict for test session in production mode",
			sessionName: "test-osoba-123",
			existingSessions: map[string]bool{
				"test-osoba-123": true,
			},
			testMode:    false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			origTestMode := os.Getenv("OSOBA_TEST_MODE")
			defer func() {
				if origTestMode == "" {
					os.Unsetenv("OSOBA_TEST_MODE")
				} else {
					os.Setenv("OSOBA_TEST_MODE", origTestMode)
				}
			}()

			if tt.testMode {
				os.Setenv("OSOBA_TEST_MODE", "true")
			} else {
				os.Unsetenv("OSOBA_TEST_MODE")
			}

			// Create mock manager and detector
			mockManager := NewMockConflictManager()
			mockManager.sessions = tt.existingSessions
			detector := NewConflictDetector(mockManager)

			// Test
			err := detector.CheckSessionConflict(tt.sessionName)
			if (err != nil) != tt.expectError {
				t.Errorf("CheckSessionConflict() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestConflictDetector_LockUnlock(t *testing.T) {
	mockManager := NewMockConflictManager()
	detector := NewConflictDetector(mockManager)

	sessionName := "test-session"

	// Lock session
	err := detector.LockSession(sessionName)
	if err != nil {
		t.Fatalf("LockSession() error = %v", err)
	}

	// Try to lock again from same process - should succeed
	err = detector.LockSession(sessionName)
	if err != nil {
		t.Fatalf("LockSession() again error = %v", err)
	}

	// Unlock session
	err = detector.UnlockSession(sessionName)
	if err != nil {
		t.Fatalf("UnlockSession() error = %v", err)
	}

	// Unlock again - should be no-op
	err = detector.UnlockSession(sessionName)
	if err != nil {
		t.Fatalf("UnlockSession() again error = %v", err)
	}
}

func TestConflictDetector_PortConflicts(t *testing.T) {
	mockManager := NewMockConflictManager()
	detector := NewConflictDetector(mockManager)

	// Reserve port for production
	err := detector.ReservePort(8080, false)
	if err != nil {
		t.Fatalf("ReservePort(8080, production) error = %v", err)
	}

	// Try to reserve same port for test - should fail
	err = detector.ReservePort(8080, true)
	if err == nil {
		t.Fatal("ReservePort(8080, test) should fail when production has it")
	}

	// Reserve different port for test - should succeed
	err = detector.ReservePort(8081, true)
	if err != nil {
		t.Fatalf("ReservePort(8081, test) error = %v", err)
	}

	// Check port conflicts
	err = detector.CheckPortConflict(8080, true)
	if err == nil {
		t.Fatal("CheckPortConflict(8080, test) should fail")
	}

	err = detector.CheckPortConflict(8081, false)
	if err == nil {
		t.Fatal("CheckPortConflict(8081, production) should fail")
	}

	// Release ports
	detector.ReleasePort(8080, false)
	detector.ReleasePort(8081, true)

	// Now both should be available
	err = detector.CheckPortConflict(8080, true)
	if err != nil {
		t.Fatalf("CheckPortConflict(8080, test) after release error = %v", err)
	}

	err = detector.CheckPortConflict(8081, false)
	if err != nil {
		t.Fatalf("CheckPortConflict(8081, production) after release error = %v", err)
	}
}

func TestConflictDetector_ValidateEnvironmentConsistency(t *testing.T) {
	tests := []struct {
		name        string
		testMode    bool
		testSocket  string
		testPrefix  string
		sessions    map[string]bool
		expectError bool
	}{
		{
			name:       "valid test mode with socket",
			testMode:   true,
			testSocket: "/tmp/test.sock",
			testPrefix: "test-osoba-",
			sessions: map[string]bool{
				"osoba-prod": true, // OK with socket isolation
			},
			expectError: false,
		},
		{
			name:       "test mode without socket but with production sessions",
			testMode:   true,
			testSocket: "",
			testPrefix: "test-osoba-",
			sessions: map[string]bool{
				"osoba-prod": true,
			},
			expectError: true,
		},
		{
			name:        "test mode without prefix",
			testMode:    true,
			testSocket:  "/tmp/test.sock",
			testPrefix:  "",
			sessions:    map[string]bool{},
			expectError: true,
		},
		{
			name:       "production mode with test sessions (warning only)",
			testMode:   false,
			testSocket: "",
			testPrefix: "",
			sessions: map[string]bool{
				"test-osoba-123": true,
			},
			expectError: false, // Just warns, doesn't error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			origTestMode := os.Getenv("OSOBA_TEST_MODE")
			origSocket := os.Getenv("OSOBA_TEST_SOCKET")
			origPrefix := os.Getenv("OSOBA_TEST_SESSION_PREFIX")
			defer func() {
				if origTestMode == "" {
					os.Unsetenv("OSOBA_TEST_MODE")
				} else {
					os.Setenv("OSOBA_TEST_MODE", origTestMode)
				}
				if origSocket == "" {
					os.Unsetenv("OSOBA_TEST_SOCKET")
				} else {
					os.Setenv("OSOBA_TEST_SOCKET", origSocket)
				}
				if origPrefix == "" {
					os.Unsetenv("OSOBA_TEST_SESSION_PREFIX")
				} else {
					os.Setenv("OSOBA_TEST_SESSION_PREFIX", origPrefix)
				}
			}()

			// Set test environment
			if tt.testMode {
				os.Setenv("OSOBA_TEST_MODE", "true")
			} else {
				os.Unsetenv("OSOBA_TEST_MODE")
			}

			if tt.testSocket != "" {
				os.Setenv("OSOBA_TEST_SOCKET", tt.testSocket)
			} else {
				os.Unsetenv("OSOBA_TEST_SOCKET")
			}

			if tt.testPrefix != "" {
				os.Setenv("OSOBA_TEST_SESSION_PREFIX", tt.testPrefix)
			} else {
				os.Unsetenv("OSOBA_TEST_SESSION_PREFIX")
			}

			// Create mock manager and detector
			mockManager := NewMockConflictManager()
			mockManager.sessions = tt.sessions
			detector := NewConflictDetector(mockManager)

			// Test
			err := detector.ValidateEnvironmentConsistency()
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateEnvironmentConsistency() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestIsolationValidator_ValidateIsolation(t *testing.T) {
	tests := []struct {
		name        string
		testMode    bool
		testSocket  string
		testPrefix  string
		expectError bool
	}{
		{
			name:        "valid isolation with socket",
			testMode:    true,
			testSocket:  "/tmp/test.sock",
			testPrefix:  "test-osoba-",
			expectError: false,
		},
		{
			name:        "valid isolation with prefix only",
			testMode:    true,
			testSocket:  "",
			testPrefix:  "test-osoba-",
			expectError: false,
		},
		{
			name:        "invalid - no isolation",
			testMode:    true,
			testSocket:  "",
			testPrefix:  "",
			expectError: true,
		},
		{
			name:        "invalid - bad prefix",
			testMode:    true,
			testSocket:  "",
			testPrefix:  "prod-",
			expectError: true,
		},
		{
			name:        "not in test mode",
			testMode:    false,
			testSocket:  "",
			testPrefix:  "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			origTestMode := os.Getenv("OSOBA_TEST_MODE")
			origSocket := os.Getenv("OSOBA_TEST_SOCKET")
			origPrefix := os.Getenv("OSOBA_TEST_SESSION_PREFIX")
			defer func() {
				if origTestMode == "" {
					os.Unsetenv("OSOBA_TEST_MODE")
				} else {
					os.Setenv("OSOBA_TEST_MODE", origTestMode)
				}
				if origSocket == "" {
					os.Unsetenv("OSOBA_TEST_SOCKET")
				} else {
					os.Setenv("OSOBA_TEST_SOCKET", origSocket)
				}
				if origPrefix == "" {
					os.Unsetenv("OSOBA_TEST_SESSION_PREFIX")
				} else {
					os.Setenv("OSOBA_TEST_SESSION_PREFIX", origPrefix)
				}
			}()

			// Set test environment
			if tt.testMode {
				os.Setenv("OSOBA_TEST_MODE", "true")
			} else {
				os.Unsetenv("OSOBA_TEST_MODE")
			}

			if tt.testSocket != "" {
				os.Setenv("OSOBA_TEST_SOCKET", tt.testSocket)
			} else {
				os.Unsetenv("OSOBA_TEST_SOCKET")
			}

			if tt.testPrefix != "" {
				os.Setenv("OSOBA_TEST_SESSION_PREFIX", tt.testPrefix)
			} else {
				os.Unsetenv("OSOBA_TEST_SESSION_PREFIX")
			}

			// Create validator
			mockManager := NewMockConflictManager()
			validator := NewIsolationValidator(mockManager)

			// Test - Skip actual socket test since tmux may not be available
			if tt.testSocket != "" {
				// Would test socket isolation in integration test
				return
			}

			err := validator.ValidateIsolation()
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateIsolation() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

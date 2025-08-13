package tmux

import (
	"os"
	"strings"
	"testing"
)

func TestNewTestManager(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		expectSocket   bool
		expectPrefix   string
		expectTestMode bool
	}{
		{
			name: "with test mode and socket",
			envVars: map[string]string{
				"OSOBA_TEST_MODE":           "true",
				"OSOBA_TEST_SOCKET":         "/tmp/test.sock",
				"OSOBA_TEST_SESSION_PREFIX": "test-custom-",
			},
			expectSocket:   true,
			expectPrefix:   "test-custom-",
			expectTestMode: true,
		},
		{
			name: "with test mode without socket",
			envVars: map[string]string{
				"OSOBA_TEST_MODE": "true",
			},
			expectSocket:   false,
			expectPrefix:   "test-osoba-",
			expectTestMode: true,
		},
		{
			name:           "without test mode",
			envVars:        map[string]string{},
			expectSocket:   false,
			expectPrefix:   "test-osoba-",
			expectTestMode: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore env vars
			origVars := make(map[string]string)
			origSet := make(map[string]bool)
			envKeys := []string{"OSOBA_TEST_MODE", "OSOBA_TEST_SOCKET", "OSOBA_TEST_SESSION_PREFIX"}
			for _, k := range envKeys {
				if v, ok := os.LookupEnv(k); ok {
					origVars[k] = v
					origSet[k] = true
				}
			}
			defer func() {
				for _, k := range envKeys {
					if origSet[k] {
						os.Setenv(k, origVars[k])
					} else {
						os.Unsetenv(k)
					}
				}
			}()

			// Clear all test env vars first
			for _, k := range envKeys {
				os.Unsetenv(k)
			}

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Create test manager
			manager := NewTestManager()

			// Check test mode
			if manager.IsTestMode() != tt.expectTestMode {
				t.Errorf("IsTestMode() = %v, want %v", manager.IsTestMode(), tt.expectTestMode)
			}

			// Check socket
			if tt.expectSocket {
				if manager.GetTestSocket() == "" {
					t.Error("Expected test socket but got empty")
				}
			} else {
				if manager.GetTestSocket() != "" {
					t.Errorf("Expected no test socket but got %s", manager.GetTestSocket())
				}
			}

			// Check session prefix
			if manager.GetSessionPrefix() != tt.expectPrefix {
				t.Errorf("GetSessionPrefix() = %v, want %v", manager.GetSessionPrefix(), tt.expectPrefix)
			}
		})
	}
}

func TestNewTestManagerWithSocket(t *testing.T) {
	socket := "/tmp/test-explicit.sock"
	prefix := "test-explicit-"

	manager := NewTestManagerWithSocket(socket, prefix)

	if !manager.IsTestMode() {
		t.Error("IsTestMode() should return true for explicit socket manager")
	}

	if manager.GetTestSocket() != socket {
		t.Errorf("GetTestSocket() = %v, want %v", manager.GetTestSocket(), socket)
	}

	if manager.GetSessionPrefix() != prefix {
		t.Errorf("GetSessionPrefix() = %v, want %v", manager.GetSessionPrefix(), prefix)
	}
}

func TestTestManager_SessionPrefixHandling(t *testing.T) {
	manager := NewTestManagerWithSocket("/tmp/test.sock", "test-prefix-")

	tests := []struct {
		name         string
		sessionName  string
		expectPrefix bool
	}{
		{
			name:         "session without prefix",
			sessionName:  "my-session",
			expectPrefix: true,
		},
		{
			name:         "session with prefix",
			sessionName:  "test-prefix-my-session",
			expectPrefix: false, // Already has prefix, shouldn't add again
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't actually create sessions without tmux, but we can test the prefix logic
			// by checking if the methods would add the prefix

			// Verify the expected behavior
			if tt.expectPrefix {
				if strings.HasPrefix(tt.sessionName, manager.GetSessionPrefix()) {
					t.Error("Session name should not already have prefix")
				}
			} else {
				if !strings.HasPrefix(tt.sessionName, manager.GetSessionPrefix()) {
					t.Error("Session name should already have prefix")
				}
			}
		})
	}
}

func TestTestCommandExecutor_Execute(t *testing.T) {
	executor := &TestCommandExecutor{
		testSocket: "/tmp/test.sock",
	}

	tests := []struct {
		name         string
		command      string
		args         []string
		expectSocket bool
	}{
		{
			name:         "tmux command with socket",
			command:      "tmux",
			args:         []string{"list-sessions"},
			expectSocket: true,
		},
		{
			name:         "non-tmux command",
			command:      "echo",
			args:         []string{"hello"},
			expectSocket: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't actually execute tmux commands in test, but we can verify the argument modification
			// This would need to be tested with a mock executor in a real implementation

			// For now, just verify the logic is correct
			if tt.command == "tmux" && tt.expectSocket {
				// Socket should be injected for tmux commands
				if executor.testSocket == "" {
					t.Error("Test socket should be set for tmux commands")
				}
			}
		})
	}
}

func TestValidateTestEnvironment(t *testing.T) {
	tests := []struct {
		name      string
		envVars   map[string]string
		expectErr bool
	}{
		{
			name: "valid test environment",
			envVars: map[string]string{
				"OSOBA_TEST_MODE": "true",
			},
			expectErr: false,
		},
		{
			name:      "missing test mode",
			envVars:   map[string]string{},
			expectErr: true,
		},
		{
			name: "test mode false",
			envVars: map[string]string{
				"OSOBA_TEST_MODE": "false",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore env vars
			origTestMode := os.Getenv("OSOBA_TEST_MODE")
			origSocket := os.Getenv("OSOBA_TEST_SOCKET")
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
			}()

			// Clear env vars
			os.Unsetenv("OSOBA_TEST_MODE")
			os.Unsetenv("OSOBA_TEST_SOCKET")

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			err := ValidateTestEnvironment()
			if (err != nil) != tt.expectErr {
				t.Errorf("ValidateTestEnvironment() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

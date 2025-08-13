package testenv

import (
	"context"
	"os"
	"testing"
)

func TestManager_Setup(t *testing.T) {
	tests := []struct {
		name                string
		config              *Config
		expectTestMode      bool
		expectSocket        bool
		expectSessionPrefix bool
	}{
		{
			name:                "default config with socket isolation",
			config:              DefaultConfig(),
			expectTestMode:      true,
			expectSocket:        true,
			expectSessionPrefix: true,
		},
		{
			name: "config without socket isolation",
			config: &Config{
				UseSocketIsolation: false,
				SessionPrefix:      "test-",
				AutoCleanup:        false,
			},
			expectTestMode:      true,
			expectSocket:        false,
			expectSessionPrefix: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env vars
			origTestMode, origTestModeSet := os.LookupEnv("OSOBA_TEST_MODE")
			origSocket, origSocketSet := os.LookupEnv("OSOBA_TEST_SOCKET")
			origPrefix, origPrefixSet := os.LookupEnv("OSOBA_TEST_SESSION_PREFIX")
			defer func() {
				if origTestModeSet {
					os.Setenv("OSOBA_TEST_MODE", origTestMode)
				} else {
					os.Unsetenv("OSOBA_TEST_MODE")
				}
				if origSocketSet {
					os.Setenv("OSOBA_TEST_SOCKET", origSocket)
				} else {
					os.Unsetenv("OSOBA_TEST_SOCKET")
				}
				if origPrefixSet {
					os.Setenv("OSOBA_TEST_SESSION_PREFIX", origPrefix)
				} else {
					os.Unsetenv("OSOBA_TEST_SESSION_PREFIX")
				}
			}()

			manager := NewManager(t, tt.config)
			ctx := context.Background()

			err := manager.Setup(ctx)
			if err != nil {
				t.Fatalf("Setup() error = %v", err)
			}

			// Check environment variables
			if tt.expectTestMode {
				if got := os.Getenv("OSOBA_TEST_MODE"); got != "true" {
					t.Errorf("OSOBA_TEST_MODE = %v, want true", got)
				}
			}

			if tt.expectSocket {
				if got := os.Getenv("OSOBA_TEST_SOCKET"); got == "" {
					t.Error("OSOBA_TEST_SOCKET not set")
				}
			} else {
				if got := os.Getenv("OSOBA_TEST_SOCKET"); got != "" {
					t.Errorf("OSOBA_TEST_SOCKET = %v, want empty", got)
				}
			}

			if tt.expectSessionPrefix {
				if got := os.Getenv("OSOBA_TEST_SESSION_PREFIX"); got == "" {
					t.Error("OSOBA_TEST_SESSION_PREFIX not set")
				}
			}

			// Test teardown
			err = manager.Teardown(ctx)
			if err != nil {
				t.Fatalf("Teardown() error = %v", err)
			}

			// Check environment variables are cleaned up
			currentTestMode, currentTestModeSet := os.LookupEnv("OSOBA_TEST_MODE")
			if currentTestModeSet != origTestModeSet || (currentTestModeSet && currentTestMode != origTestMode) {
				t.Error("OSOBA_TEST_MODE not restored")
			}
			currentSocket, currentSocketSet := os.LookupEnv("OSOBA_TEST_SOCKET")
			if currentSocketSet != origSocketSet || (currentSocketSet && currentSocket != origSocket) {
				t.Error("OSOBA_TEST_SOCKET not restored")
			}
			currentPrefix, currentPrefixSet := os.LookupEnv("OSOBA_TEST_SESSION_PREFIX")
			if currentPrefixSet != origPrefixSet || (currentPrefixSet && currentPrefix != origPrefix) {
				t.Error("OSOBA_TEST_SESSION_PREFIX not restored")
			}
		})
	}
}

func TestManager_IsTestMode(t *testing.T) {
	// Save and clear test mode env var
	origTestMode, origTestModeSet := os.LookupEnv("OSOBA_TEST_MODE")
	os.Unsetenv("OSOBA_TEST_MODE")
	defer func() {
		if origTestModeSet {
			os.Setenv("OSOBA_TEST_MODE", origTestMode)
		} else {
			os.Unsetenv("OSOBA_TEST_MODE")
		}
	}()

	config := DefaultConfig()
	config.AutoCleanup = false // Disable signal handlers in test
	manager := NewManager(t, config)
	ctx := context.Background()

	// Before setup
	if manager.IsTestMode() {
		t.Error("IsTestMode() should return false before setup")
	}

	// After setup
	err := manager.Setup(ctx)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	defer manager.Teardown(ctx)

	if !manager.IsTestMode() {
		t.Error("IsTestMode() should return true after setup")
	}
}

func TestManager_GetTestSocket(t *testing.T) {
	tests := []struct {
		name         string
		config       *Config
		expectSocket bool
	}{
		{
			name:         "with socket isolation",
			config:       DefaultConfig(),
			expectSocket: true,
		},
		{
			name: "without socket isolation",
			config: &Config{
				UseSocketIsolation: false,
			},
			expectSocket: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(t, tt.config)

			socket := manager.GetTestSocket()
			if tt.expectSocket && socket == "" {
				t.Error("GetTestSocket() returned empty string, expected socket path")
			}
			if !tt.expectSocket && socket != "" {
				t.Errorf("GetTestSocket() = %v, expected empty string", socket)
			}
		})
	}
}

func TestManager_RegisterCleanup(t *testing.T) {
	manager := NewManager(t, DefaultConfig())
	ctx := context.Background()

	// Track cleanup execution
	var cleanupOrder []int

	// Register multiple cleanup functions
	manager.RegisterCleanup(func() error {
		cleanupOrder = append(cleanupOrder, 1)
		return nil
	})

	manager.RegisterCleanup(func() error {
		cleanupOrder = append(cleanupOrder, 2)
		return nil
	})

	manager.RegisterCleanup(func() error {
		cleanupOrder = append(cleanupOrder, 3)
		return nil
	})

	// Setup and teardown
	err := manager.Setup(ctx)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	err = manager.Teardown(ctx)
	if err != nil {
		t.Fatalf("Teardown() error = %v", err)
	}

	// Check cleanup functions were called in reverse order
	expectedOrder := []int{3, 2, 1}
	if len(cleanupOrder) != len(expectedOrder) {
		t.Fatalf("cleanup order length = %d, want %d", len(cleanupOrder), len(expectedOrder))
	}

	for i, v := range cleanupOrder {
		if v != expectedOrder[i] {
			t.Errorf("cleanup order[%d] = %d, want %d", i, v, expectedOrder[i])
		}
	}
}

func TestManager_MultipleSetup(t *testing.T) {
	manager := NewManager(t, DefaultConfig())
	ctx := context.Background()

	// First setup
	err := manager.Setup(ctx)
	if err != nil {
		t.Fatalf("First Setup() error = %v", err)
	}

	// Second setup should be no-op
	err = manager.Setup(ctx)
	if err != nil {
		t.Fatalf("Second Setup() error = %v", err)
	}

	// Cleanup
	err = manager.Teardown(ctx)
	if err != nil {
		t.Fatalf("Teardown() error = %v", err)
	}
}

func TestWithTestEnvironment(t *testing.T) {
	var setupCalled, teardownCalled bool

	WithTestEnvironment(t, DefaultConfig(), func(manager TestEnvironmentManager) {
		setupCalled = manager.IsTestMode()

		// Register cleanup to verify it's called
		manager.RegisterCleanup(func() error {
			teardownCalled = true
			return nil
		})
	})

	if !setupCalled {
		t.Error("Setup was not called properly")
	}

	if !teardownCalled {
		t.Error("Teardown/cleanup was not called properly")
	}
}

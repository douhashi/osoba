package config

import (
	"os"
	"testing"
)

// TestNewConfig_TestMode tests NewConfig behavior with OSOBA_TEST_MODE
func TestNewConfig_TestMode(t *testing.T) {
	tests := []struct {
		name              string
		testModeEnv       string
		wantSessionPrefix string
		wantIsTestMode    bool
	}{
		{
			name:              "test mode enabled",
			testModeEnv:       "true",
			wantSessionPrefix: "test-osoba-",
			wantIsTestMode:    true,
		},
		{
			name:              "test mode disabled",
			testModeEnv:       "",
			wantSessionPrefix: "osoba-",
			wantIsTestMode:    false,
		},
		{
			name:              "test mode explicitly false",
			testModeEnv:       "false",
			wantSessionPrefix: "osoba-",
			wantIsTestMode:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			origTestMode := os.Getenv("OSOBA_TEST_MODE")
			if tt.testModeEnv != "" {
				os.Setenv("OSOBA_TEST_MODE", tt.testModeEnv)
			} else {
				os.Unsetenv("OSOBA_TEST_MODE")
			}
			defer func() {
				if origTestMode != "" {
					os.Setenv("OSOBA_TEST_MODE", origTestMode)
				} else {
					os.Unsetenv("OSOBA_TEST_MODE")
				}
			}()

			// Create config
			cfg := NewConfig()

			// Check session prefix
			if cfg.Tmux.SessionPrefix != tt.wantSessionPrefix {
				t.Errorf("SessionPrefix = %q, want %q", cfg.Tmux.SessionPrefix, tt.wantSessionPrefix)
			}

			// Check IsTestMode flag
			if cfg.IsTestMode != tt.wantIsTestMode {
				t.Errorf("IsTestMode = %v, want %v", cfg.IsTestMode, tt.wantIsTestMode)
			}
		})
	}
}

// TestConfig_Load_TestMode tests that OSOBA_TEST_MODE overrides loaded config
func TestConfig_Load_TestMode(t *testing.T) {
	// Create test config file
	configContent := `
tmux:
  session_prefix: "custom-osoba-"
`
	filename := "test_config_test_mode.yml"
	err := os.WriteFile(filename, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	defer os.Remove(filename)

	tests := []struct {
		name              string
		testModeEnv       string
		wantSessionPrefix string
		wantIsTestMode    bool
	}{
		{
			name:              "test mode overrides config file",
			testModeEnv:       "true",
			wantSessionPrefix: "test-osoba-",
			wantIsTestMode:    true,
		},
		{
			name:              "normal mode uses config file",
			testModeEnv:       "",
			wantSessionPrefix: "custom-osoba-",
			wantIsTestMode:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			origTestMode := os.Getenv("OSOBA_TEST_MODE")
			if tt.testModeEnv != "" {
				os.Setenv("OSOBA_TEST_MODE", tt.testModeEnv)
			} else {
				os.Unsetenv("OSOBA_TEST_MODE")
			}
			defer func() {
				if origTestMode != "" {
					os.Setenv("OSOBA_TEST_MODE", origTestMode)
				} else {
					os.Unsetenv("OSOBA_TEST_MODE")
				}
			}()

			// Create and load config
			cfg := NewConfig()
			if err := cfg.Load(filename); err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			// Check session prefix
			if cfg.Tmux.SessionPrefix != tt.wantSessionPrefix {
				t.Errorf("SessionPrefix = %q, want %q", cfg.Tmux.SessionPrefix, tt.wantSessionPrefix)
			}

			// Check IsTestMode flag
			if cfg.IsTestMode != tt.wantIsTestMode {
				t.Errorf("IsTestMode = %v, want %v", cfg.IsTestMode, tt.wantIsTestMode)
			}
		})
	}
}

// TestConfig_LoadOrDefault_TestMode tests LoadOrDefault with OSOBA_TEST_MODE
func TestConfig_LoadOrDefault_TestMode(t *testing.T) {
	tests := []struct {
		name              string
		testModeEnv       string
		wantSessionPrefix string
		wantIsTestMode    bool
	}{
		{
			name:              "test mode with LoadOrDefault",
			testModeEnv:       "true",
			wantSessionPrefix: "test-osoba-",
			wantIsTestMode:    true,
		},
		{
			name:              "normal mode with LoadOrDefault",
			testModeEnv:       "",
			wantSessionPrefix: "osoba-",
			wantIsTestMode:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			origTestMode := os.Getenv("OSOBA_TEST_MODE")
			if tt.testModeEnv != "" {
				os.Setenv("OSOBA_TEST_MODE", tt.testModeEnv)
			} else {
				os.Unsetenv("OSOBA_TEST_MODE")
			}
			defer func() {
				if origTestMode != "" {
					os.Setenv("OSOBA_TEST_MODE", origTestMode)
				} else {
					os.Unsetenv("OSOBA_TEST_MODE")
				}
			}()

			// Create config and load with non-existent file
			cfg := NewConfig()
			cfg.LoadOrDefault("non-existent-file.yml")

			// Check session prefix
			if cfg.Tmux.SessionPrefix != tt.wantSessionPrefix {
				t.Errorf("SessionPrefix = %q, want %q", cfg.Tmux.SessionPrefix, tt.wantSessionPrefix)
			}

			// Check IsTestMode flag
			if cfg.IsTestMode != tt.wantIsTestMode {
				t.Errorf("IsTestMode = %v, want %v", cfg.IsTestMode, tt.wantIsTestMode)
			}
		})
	}
}

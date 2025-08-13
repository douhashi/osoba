//go:build integration
// +build integration

package tmux

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSafetyCheckForProductionSessions tests the safety check functionality
func TestSafetyCheckForProductionSessions(t *testing.T) {
	tests := []struct {
		name              string
		existingSessions  []string
		testSessionPrefix string
		prodSessionPrefix string
		expectWarning     bool
		expectBlocked     bool
	}{
		{
			name:              "no production sessions",
			existingSessions:  []string{},
			testSessionPrefix: "test-osoba-",
			prodSessionPrefix: "osoba-",
			expectWarning:     false,
			expectBlocked:     false,
		},
		{
			name:              "production sessions exist",
			existingSessions:  []string{"osoba-repo1", "osoba-repo2"},
			testSessionPrefix: "test-osoba-",
			prodSessionPrefix: "osoba-",
			expectWarning:     true,
			expectBlocked:     false, // In CI mode, only warn
		},
		{
			name:              "only test sessions exist",
			existingSessions:  []string{"test-osoba-123", "test-osoba-456"},
			testSessionPrefix: "test-osoba-",
			prodSessionPrefix: "osoba-",
			expectWarning:     false,
			expectBlocked:     false,
		},
		{
			name:              "mixed sessions exist",
			existingSessions:  []string{"osoba-prod", "test-osoba-123"},
			testSessionPrefix: "test-osoba-",
			prodSessionPrefix: "osoba-",
			expectWarning:     true,
			expectBlocked:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the session listing
			sessions := tt.existingSessions

			// Check for production sessions
			hasProductionSessions := false
			for _, session := range sessions {
				if strings.HasPrefix(session, tt.prodSessionPrefix) &&
					!strings.HasPrefix(session, tt.testSessionPrefix) {
					hasProductionSessions = true
					break
				}
			}

			assert.Equal(t, tt.expectWarning, hasProductionSessions,
				"Production session detection mismatch")
		})
	}
}

// TestSessionPrefixConfiguration tests that session prefixes are properly configured
func TestSessionPrefixConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		envVar         string
		expectedPrefix string
	}{
		{
			name:           "test mode enabled",
			envVar:         "true",
			expectedPrefix: "test-osoba-",
		},
		{
			name:           "test mode disabled",
			envVar:         "",
			expectedPrefix: "osoba-",
		},
		{
			name:           "test mode explicitly false",
			envVar:         "false",
			expectedPrefix: "osoba-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envVar != "" {
				os.Setenv("OSOBA_TEST_MODE", tt.envVar)
				defer os.Unsetenv("OSOBA_TEST_MODE")
			}

			// Get the session prefix based on environment
			prefix := GetSessionPrefix()
			assert.Equal(t, tt.expectedPrefix, prefix)
		})
	}
}

// TestCIEnvironmentDetection tests CI environment detection
func TestCIEnvironmentDetection(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expectCI bool
	}{
		{
			name:     "CI environment",
			envVars:  map[string]string{"CI": "true"},
			expectCI: true,
		},
		{
			name:     "GitHub Actions environment",
			envVars:  map[string]string{"GITHUB_ACTIONS": "true"},
			expectCI: true,
		},
		{
			name:     "local environment",
			envVars:  map[string]string{},
			expectCI: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, val := range tt.envVars {
				os.Setenv(key, val)
				defer os.Unsetenv(key)
			}

			// Check CI detection
			isCI := IsCIEnvironment()
			assert.Equal(t, tt.expectCI, isCI)
		})
	}
}

// TestCheckProductionSessions checks if production sessions are properly detected
func TestCheckProductionSessions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if tmux is available
	if err := exec.Command("tmux", "list-sessions").Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// No sessions, which is fine
		} else {
			t.Skip("tmux command not available")
		}
	}

	// Get actual sessions
	prodSessions, err := CheckProductionSessions()
	require.NoError(t, err)

	// Log found sessions for debugging
	if len(prodSessions) > 0 {
		t.Logf("Found %d production sessions: %v", len(prodSessions), prodSessions)
	}

	// In test mode, we should not have production sessions with our test prefix
	testPrefix := GetSessionPrefix()
	if testPrefix == "test-osoba-" {
		for _, session := range prodSessions {
			assert.False(t, strings.HasPrefix(session, "test-osoba-"),
				"Test session should not be detected as production: %s", session)
		}
	}
}

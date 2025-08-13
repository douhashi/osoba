package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GetSessionPrefix returns the appropriate session prefix based on environment
func GetSessionPrefix() string {
	if os.Getenv("OSOBA_TEST_MODE") == "true" {
		return "test-osoba-"
	}
	return "osoba-"
}

// IsCIEnvironment detects if running in CI environment
func IsCIEnvironment() bool {
	// Check common CI environment variables
	ciVars := []string{"CI", "GITHUB_ACTIONS", "JENKINS", "TRAVIS", "CIRCLECI"}
	for _, envVar := range ciVars {
		if os.Getenv(envVar) == "true" || os.Getenv(envVar) == "1" {
			return true
		}
	}
	return false
}

// CheckProductionSessions lists existing production sessions
func CheckProductionSessions() ([]string, error) {
	output, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// No sessions exist
			return []string{}, nil
		}
		return nil, err
	}

	sessions := strings.Split(strings.TrimSpace(string(output)), "\n")
	var prodSessions []string

	for _, session := range sessions {
		if session != "" && strings.HasPrefix(session, "osoba-") &&
			!strings.HasPrefix(session, "test-osoba-") {
			prodSessions = append(prodSessions, session)
		}
	}

	return prodSessions, nil
}

// SafetyCheckBeforeTests performs safety checks before running tests
func SafetyCheckBeforeTests() error {
	// Skip check if not in test mode
	if os.Getenv("OSOBA_TEST_MODE") != "true" {
		return nil
	}

	// Check for production sessions
	prodSessions, err := CheckProductionSessions()
	if err != nil {
		return fmt.Errorf("failed to check production sessions: %w", err)
	}

	if len(prodSessions) > 0 {
		fmt.Fprintf(os.Stderr, "⚠️  WARNING: Found %d production osoba session(s):\n", len(prodSessions))
		for _, session := range prodSessions {
			fmt.Fprintf(os.Stderr, "   - %s\n", session)
		}

		// In CI environment, just warn
		if IsCIEnvironment() {
			fmt.Fprintf(os.Stderr, "   Running in CI environment, continuing with tests...\n")
		} else {
			// In local environment, warn but continue
			fmt.Fprintf(os.Stderr, "   Tests will use 'test-osoba-' prefix to avoid conflicts.\n")
			fmt.Fprintf(os.Stderr, "   Production sessions should be safe.\n")
		}
	}

	return nil
}

// IsTestSession checks if a session name is a test session
func IsTestSession(sessionName string) bool {
	return strings.HasPrefix(sessionName, "test-osoba-")
}

// IsProductionSession checks if a session name is a production session
func IsProductionSession(sessionName string) bool {
	return strings.HasPrefix(sessionName, "osoba-") && !IsTestSession(sessionName)
}

// CleanupTestSessions removes all test sessions
func CleanupTestSessions() error {
	output, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// No sessions exist
			return nil
		}
		return err
	}

	sessions := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, session := range sessions {
		if session != "" && IsTestSession(session) {
			// Kill test session, ignore errors if session doesn't exist
			exec.Command("tmux", "kill-session", "-t", session).Run()
		}
	}

	return nil
}

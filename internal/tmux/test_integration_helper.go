//go:build integration
// +build integration

package tmux

import (
	"context"
	"fmt"
	"os"
	"testing"
	
	"github.com/douhashi/osoba/internal/testutil/testenv"
	"github.com/douhashi/osoba/internal/testutil/factories"
)

// IntegrationTestSetup sets up the test environment for tmux integration tests.
type IntegrationTestSetup struct {
	T           *testing.T
	EnvManager  testenv.TestEnvironmentManager
	TmuxManager Manager
	Cleanup     *testenv.CleanupHandler
}

// SetupIntegrationTest creates a new integration test setup.
func SetupIntegrationTest(t *testing.T) *IntegrationTestSetup {
	t.Helper()
	
	// Create test environment manager
	envManager := testenv.NewManager(t, testenv.DefaultConfig())
	
	// Setup test environment
	ctx := context.Background()
	if err := envManager.Setup(ctx); err != nil {
		t.Fatalf("failed to setup test environment: %v", err)
	}
	
	// Create cleanup handler
	cleanupHandler := testenv.NewCleanupHandler()
	cleanupHandler.EnablePanicRecovery()
	
	// Create tmux manager based on environment
	var tmuxManager Manager
	if os.Getenv("OSOBA_USE_MOCK_TMUX") == "true" {
		// Use mock for faster tests
		tmuxManager = factories.NewMockTmuxManager()
	} else if envManager.GetTestSocket() != "" {
		// Use isolated test manager
		tmuxManager = NewTestManagerWithSocket(
			envManager.GetTestSocket(),
			"test-osoba-",
		)
	} else {
		// Use test manager with prefix isolation
		tmuxManager = NewTestManager()
	}
	
	// Register cleanup for environment teardown
	cleanupHandler.RegisterFunc("teardown-environment", func() error {
		return envManager.Teardown(ctx)
	})
	
	// Register cleanup for test sessions
	cleanupHandler.RegisterFunc("cleanup-test-sessions", func() error {
		if testMgr, ok := tmuxManager.(*TestManager); ok {
			return testMgr.CleanupTestSessions()
		}
		return nil
	})
	
	// Add cleanup to test
	t.Cleanup(func() {
		if err := cleanupHandler.Execute(context.Background()); err != nil {
			t.Errorf("cleanup failed: %v", err)
		}
	})
	
	return &IntegrationTestSetup{
		T:           t,
		EnvManager:  envManager,
		TmuxManager: tmuxManager,
		Cleanup:     cleanupHandler,
	}
}

// WithIsolatedTmux runs a test with an isolated tmux environment.
func WithIsolatedTmux(t *testing.T, testFunc func(manager Manager)) {
	t.Helper()
	
	setup := SetupIntegrationTest(t)
	
	// Validate isolation before running test
	validator := NewIsolationValidator(setup.TmuxManager)
	if err := validator.ValidateIsolation(); err != nil {
		t.Logf("WARNING: Isolation validation failed: %v", err)
	}
	
	// Validate no production access
	if err := validator.ValidateNoProductionAccess(); err != nil {
		t.Fatalf("Test can access production sessions: %v", err)
	}
	
	// Run the test
	testFunc(setup.TmuxManager)
}

// CreateTestSession creates a test session with proper naming and cleanup.
func CreateTestSession(t *testing.T, manager Manager, suffix string) string {
	t.Helper()
	
	sessionName := fmt.Sprintf("test-osoba-%s-%d", suffix, os.Getpid())
	
	if err := manager.CreateSession(sessionName); err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	
	// Register cleanup
	t.Cleanup(func() {
		if testMgr, ok := manager.(*TestManager); ok {
			_ = testMgr.KillSession(sessionName)
		} else if defaultMgr, ok := manager.(*DefaultManager); ok {
			_, _ = defaultMgr.executor.Execute("tmux", "kill-session", "-t", sessionName)
		}
	})
	
	return sessionName
}

// AssertNoProductionSessions checks that no production sessions are visible.
func AssertNoProductionSessions(t *testing.T, manager Manager) {
	t.Helper()
	
	sessions, err := manager.ListSessions("")
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}
	
	var productionSessions []string
	for _, session := range sessions {
		if IsProductionSession(session) {
			productionSessions = append(productionSessions, session)
		}
	}
	
	if len(productionSessions) > 0 {
		t.Errorf("Found production sessions in test environment: %v", productionSessions)
	}
}

// SkipIfNoTmux skips the test if tmux is not available.
func SkipIfNoTmux(t *testing.T) {
	t.Helper()
	
	manager := NewDefaultManager()
	if err := manager.CheckTmuxInstalled(); err != nil {
		t.Skip("tmux not available, skipping test")
	}
}

// RunWithConflictDetection runs a test with conflict detection enabled.
func RunWithConflictDetection(t *testing.T, manager Manager, testFunc func()) {
	t.Helper()
	
	detector := NewConflictDetector(manager)
	
	// Validate environment consistency before test
	if err := detector.ValidateEnvironmentConsistency(); err != nil {
		t.Fatalf("Environment inconsistency detected: %v", err)
	}
	
	// Cleanup stale locks
	if err := detector.CleanupStaleLocks(); err != nil {
		t.Logf("WARNING: Failed to cleanup stale locks: %v", err)
	}
	
	// Run the test
	testFunc()
	
	// Validate environment consistency after test
	if err := detector.ValidateEnvironmentConsistency(); err != nil {
		t.Errorf("Environment inconsistency after test: %v", err)
	}
}
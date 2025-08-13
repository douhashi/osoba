package testenv

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
)

// TestEnvironmentManager manages test environment setup and teardown.
type TestEnvironmentManager interface {
	// Setup initializes the test environment.
	Setup(ctx context.Context) error

	// Teardown cleans up the test environment.
	Teardown(ctx context.Context) error

	// IsTestMode returns true if running in test mode.
	IsTestMode() bool

	// GetTestSocket returns the test-specific tmux socket path.
	GetTestSocket() string

	// RegisterCleanup registers a cleanup function to be called on teardown.
	RegisterCleanup(cleanup func() error)
}

// Config contains configuration for the test environment manager.
type Config struct {
	// UseSocketIsolation enables tmux socket isolation for tests.
	UseSocketIsolation bool

	// TestSocketPath is the custom socket path for test tmux server.
	TestSocketPath string

	// SessionPrefix is the prefix for test sessions.
	SessionPrefix string

	// AutoCleanup enables automatic cleanup on process termination.
	AutoCleanup bool
}

// DefaultConfig returns the default test environment configuration.
func DefaultConfig() *Config {
	return &Config{
		UseSocketIsolation: true,
		TestSocketPath:     fmt.Sprintf("/tmp/osoba-test-%d.sock", os.Getpid()),
		SessionPrefix:      "test-osoba-",
		AutoCleanup:        true,
	}
}

// Manager implements TestEnvironmentManager.
type Manager struct {
	config    *Config
	mu        sync.Mutex
	cleanups  []func() error
	setupDone bool
	t         *testing.T
	origEnv   map[string]*string // Store original env values (nil if unset)
}

// NewManager creates a new test environment manager.
func NewManager(t *testing.T, config *Config) *Manager {
	if config == nil {
		config = DefaultConfig()
	}

	return &Manager{
		config:   config,
		cleanups: []func() error{},
		t:        t,
		origEnv:  make(map[string]*string),
	}
}

// Setup initializes the test environment.
func (m *Manager) Setup(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.setupDone {
		return nil
	}

	// Save and set test mode environment variable
	if val, exists := os.LookupEnv("OSOBA_TEST_MODE"); exists {
		m.origEnv["OSOBA_TEST_MODE"] = &val
	} else {
		m.origEnv["OSOBA_TEST_MODE"] = nil
	}
	if err := os.Setenv("OSOBA_TEST_MODE", "true"); err != nil {
		return fmt.Errorf("failed to set OSOBA_TEST_MODE: %w", err)
	}
	m.registerCleanupLocked(func() error {
		if orig := m.origEnv["OSOBA_TEST_MODE"]; orig != nil {
			return os.Setenv("OSOBA_TEST_MODE", *orig)
		}
		return os.Unsetenv("OSOBA_TEST_MODE")
	})

	// Save and set test socket if using socket isolation
	if m.config.UseSocketIsolation {
		if val, exists := os.LookupEnv("OSOBA_TEST_SOCKET"); exists {
			m.origEnv["OSOBA_TEST_SOCKET"] = &val
		} else {
			m.origEnv["OSOBA_TEST_SOCKET"] = nil
		}
		if err := os.Setenv("OSOBA_TEST_SOCKET", m.config.TestSocketPath); err != nil {
			return fmt.Errorf("failed to set OSOBA_TEST_SOCKET: %w", err)
		}
		m.registerCleanupLocked(func() error {
			if orig := m.origEnv["OSOBA_TEST_SOCKET"]; orig != nil {
				return os.Setenv("OSOBA_TEST_SOCKET", *orig)
			}
			return os.Unsetenv("OSOBA_TEST_SOCKET")
		})
	}

	// Save and set session prefix
	if val, exists := os.LookupEnv("OSOBA_TEST_SESSION_PREFIX"); exists {
		m.origEnv["OSOBA_TEST_SESSION_PREFIX"] = &val
	} else {
		m.origEnv["OSOBA_TEST_SESSION_PREFIX"] = nil
	}
	if err := os.Setenv("OSOBA_TEST_SESSION_PREFIX", m.config.SessionPrefix); err != nil {
		return fmt.Errorf("failed to set OSOBA_TEST_SESSION_PREFIX: %w", err)
	}
	m.registerCleanupLocked(func() error {
		if orig := m.origEnv["OSOBA_TEST_SESSION_PREFIX"]; orig != nil {
			return os.Setenv("OSOBA_TEST_SESSION_PREFIX", *orig)
		}
		return os.Unsetenv("OSOBA_TEST_SESSION_PREFIX")
	})

	// Setup signal handlers for cleanup if auto cleanup is enabled
	// Skip signal handlers in test mode to avoid test conflicts
	if m.config.AutoCleanup && m.t == nil {
		m.setupSignalHandlers(ctx)
	}

	m.setupDone = true
	return nil
}

// Teardown cleans up the test environment.
func (m *Manager) Teardown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.setupDone {
		return nil
	}

	var errs []error

	// Execute cleanup functions in reverse order
	for i := len(m.cleanups) - 1; i >= 0; i-- {
		if err := m.cleanups[i](); err != nil {
			errs = append(errs, err)
		}
	}

	m.cleanups = []func() error{}
	m.setupDone = false

	if len(errs) > 0 {
		return fmt.Errorf("teardown errors: %v", errs)
	}

	return nil
}

// IsTestMode returns true if running in test mode.
func (m *Manager) IsTestMode() bool {
	return os.Getenv("OSOBA_TEST_MODE") == "true"
}

// GetTestSocket returns the test-specific tmux socket path.
func (m *Manager) GetTestSocket() string {
	if m.config.UseSocketIsolation {
		return m.config.TestSocketPath
	}
	return ""
}

// RegisterCleanup registers a cleanup function to be called on teardown.
func (m *Manager) RegisterCleanup(cleanup func() error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registerCleanupLocked(cleanup)
}

// registerCleanupLocked registers a cleanup function without locking (must be called with lock held).
func (m *Manager) registerCleanupLocked(cleanup func() error) {
	m.cleanups = append(m.cleanups, cleanup)
}

// setupSignalHandlers sets up signal handlers for cleanup.
func (m *Manager) setupSignalHandlers(ctx context.Context) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigChan:
			if m.t != nil {
				m.t.Log("Received termination signal, cleaning up test environment...")
			}
			_ = m.Teardown(context.Background())
			os.Exit(1)
		case <-ctx.Done():
			return
		}
	}()
}

// WithTestEnvironment is a helper function to set up and tear down test environment.
func WithTestEnvironment(t *testing.T, config *Config, testFunc func(manager TestEnvironmentManager)) {
	t.Helper()

	manager := NewManager(t, config)
	ctx := context.Background()

	if err := manager.Setup(ctx); err != nil {
		t.Fatalf("failed to setup test environment: %v", err)
	}

	defer func() {
		if err := manager.Teardown(ctx); err != nil {
			t.Errorf("failed to teardown test environment: %v", err)
		}
	}()

	testFunc(manager)
}

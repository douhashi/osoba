package testenv

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// CleanupHandler manages cleanup operations for test environments.
type CleanupHandler struct {
	mu           sync.Mutex
	cleanups     []CleanupFunc
	executed     bool
	timeout      time.Duration
	onPanic      bool
	onSignal     bool
	signalChan   chan os.Signal
	stopChan     chan struct{}
}

// CleanupFunc is a function that performs cleanup operations.
type CleanupFunc struct {
	Name        string
	Fn          func() error
	Critical    bool // If true, failure stops further cleanup
	IgnoreError bool // If true, errors are logged but not returned
}

// NewCleanupHandler creates a new cleanup handler.
func NewCleanupHandler() *CleanupHandler {
	return &CleanupHandler{
		cleanups:   []CleanupFunc{},
		timeout:    30 * time.Second,
		stopChan:   make(chan struct{}),
	}
}

// SetTimeout sets the timeout for cleanup operations.
func (h *CleanupHandler) SetTimeout(timeout time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.timeout = timeout
}

// Register adds a cleanup function to be executed.
func (h *CleanupHandler) Register(cleanup CleanupFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cleanups = append(h.cleanups, cleanup)
}

// RegisterFunc is a convenience method to register a simple cleanup function.
func (h *CleanupHandler) RegisterFunc(name string, fn func() error) {
	h.Register(CleanupFunc{
		Name: name,
		Fn:   fn,
	})
}

// EnablePanicRecovery enables cleanup on panic.
func (h *CleanupHandler) EnablePanicRecovery() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onPanic = true
}

// EnableSignalHandling enables cleanup on system signals.
func (h *CleanupHandler) EnableSignalHandling(signals ...os.Signal) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if h.onSignal {
		return // Already enabled
	}
	
	h.onSignal = true
	h.signalChan = make(chan os.Signal, 1)
	
	if len(signals) == 0 {
		signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}
	
	signal.Notify(h.signalChan, signals...)
	
	go h.handleSignals()
}

// handleSignals watches for signals and triggers cleanup.
func (h *CleanupHandler) handleSignals() {
	select {
	case sig := <-h.signalChan:
		fmt.Fprintf(os.Stderr, "\nReceived signal %v, performing cleanup...\n", sig)
		ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
		defer cancel()
		
		if err := h.Execute(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Cleanup error: %v\n", err)
		}
		
		os.Exit(1)
	case <-h.stopChan:
		return
	}
}

// Execute runs all registered cleanup functions.
func (h *CleanupHandler) Execute(ctx context.Context) error {
	h.mu.Lock()
	if h.executed {
		h.mu.Unlock()
		return nil
	}
	h.executed = true
	cleanups := make([]CleanupFunc, len(h.cleanups))
	copy(cleanups, h.cleanups)
	h.mu.Unlock()
	
	// Create a context with timeout if not already set
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.timeout)
		defer cancel()
	}
	
	var errors []error
	
	// Execute cleanups in reverse order (LIFO)
	for i := len(cleanups) - 1; i >= 0; i-- {
		cleanup := cleanups[i]
		
		// Run cleanup in goroutine to handle timeout
		done := make(chan error, 1)
		go func() {
			done <- cleanup.Fn()
		}()
		
		var cleanupErr error
		select {
		case err := <-done:
			cleanupErr = err
		case <-ctx.Done():
			cleanupErr = fmt.Errorf("timeout")
		}
		
		if cleanupErr != nil {
			if !cleanup.IgnoreError {
				errors = append(errors, fmt.Errorf("%s: %w", cleanup.Name, cleanupErr))
			}
			if cleanup.Critical {
				break // Stop further cleanup
			}
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}
	
	return nil
}

// Stop stops signal handling.
func (h *CleanupHandler) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if h.onSignal && h.signalChan != nil {
		signal.Stop(h.signalChan)
		close(h.stopChan)
		h.onSignal = false
	}
}

// Reset resets the cleanup handler state.
func (h *CleanupHandler) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.cleanups = []CleanupFunc{}
	h.executed = false
}

// WithCleanup runs a function with automatic cleanup on completion or panic.
func WithCleanup(fn func() error, cleanup *CleanupHandler) (err error) {
	if cleanup.onPanic {
		defer func() {
			if r := recover(); r != nil {
				// Execute cleanup on panic
				ctx := context.Background()
				_ = cleanup.Execute(ctx)
				panic(r) // Re-panic after cleanup
			}
		}()
	}
	
	// Execute the main function
	err = fn()
	
	// Always execute cleanup
	ctx := context.Background()
	cleanupErr := cleanup.Execute(ctx)
	
	// Return the first error encountered
	if err != nil {
		return err
	}
	return cleanupErr
}

// DeferredCleanup creates a cleanup function that can be deferred.
type DeferredCleanup struct {
	handler *CleanupHandler
	ctx     context.Context
}

// NewDeferredCleanup creates a new deferred cleanup.
func NewDeferredCleanup(ctx context.Context) *DeferredCleanup {
	if ctx == nil {
		ctx = context.Background()
	}
	return &DeferredCleanup{
		handler: NewCleanupHandler(),
		ctx:     ctx,
	}
}

// Register adds a cleanup function.
func (d *DeferredCleanup) Register(name string, fn func() error) {
	d.handler.RegisterFunc(name, fn)
}

// Cleanup executes all registered cleanup functions.
// This should be called in a defer statement.
func (d *DeferredCleanup) Cleanup() error {
	return d.handler.Execute(d.ctx)
}

// MustCleanup is like Cleanup but panics on error.
func (d *DeferredCleanup) MustCleanup() {
	if err := d.Cleanup(); err != nil {
		panic(fmt.Sprintf("cleanup failed: %v", err))
	}
}

// TestCleanupManager manages cleanup for test cases.
type TestCleanupManager struct {
	t        TestingT
	handler  *CleanupHandler
	autoExec bool
}

// TestingT is a minimal interface for test cleanup.
type TestingT interface {
	Helper()
	Cleanup(func())
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

// NewTestCleanupManager creates a cleanup manager for tests.
func NewTestCleanupManager(t TestingT) *TestCleanupManager {
	manager := &TestCleanupManager{
		t:        t,
		handler:  NewCleanupHandler(),
		autoExec: true,
	}
	
	// Register automatic cleanup with test framework
	t.Cleanup(func() {
		if manager.autoExec {
			ctx := context.Background()
			if err := manager.handler.Execute(ctx); err != nil {
				t.Errorf("cleanup failed: %v", err)
			}
		}
	})
	
	return manager
}

// Register adds a cleanup function.
func (m *TestCleanupManager) Register(name string, fn func() error) {
	m.t.Helper()
	m.handler.RegisterFunc(name, fn)
}

// RegisterCritical adds a critical cleanup function.
func (m *TestCleanupManager) RegisterCritical(name string, fn func() error) {
	m.t.Helper()
	m.handler.Register(CleanupFunc{
		Name:     name,
		Fn:       fn,
		Critical: true,
	})
}

// DisableAutoExec disables automatic cleanup execution.
func (m *TestCleanupManager) DisableAutoExec() {
	m.autoExec = false
}

// Execute manually triggers cleanup execution.
func (m *TestCleanupManager) Execute() error {
	m.t.Helper()
	ctx := context.Background()
	return m.handler.Execute(ctx)
}
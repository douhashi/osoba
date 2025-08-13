package tmux

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// ConflictDetector detects and prevents conflicts between test and production tmux sessions.
type ConflictDetector struct {
	mu              sync.RWMutex
	productionPorts map[int]bool
	testPorts       map[int]bool
	sessionLocks    map[string]*SessionLock
	manager         Manager
}

// SessionLock represents a lock on a tmux session.
type SessionLock struct {
	SessionName string
	LockTime    time.Time
	ProcessID   int
	IsTest      bool
}

// NewConflictDetector creates a new conflict detector.
func NewConflictDetector(manager Manager) *ConflictDetector {
	return &ConflictDetector{
		productionPorts: make(map[int]bool),
		testPorts:       make(map[int]bool),
		sessionLocks:    make(map[string]*SessionLock),
		manager:         manager,
	}
}

// CheckSessionConflict checks if a session name would conflict with existing sessions.
func (d *ConflictDetector) CheckSessionConflict(sessionName string) error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	// Check if session is locked
	if lock, exists := d.sessionLocks[sessionName]; exists {
		return fmt.Errorf("session %s is locked by process %d since %v",
			sessionName, lock.ProcessID, lock.LockTime)
	}
	
	// Check if session exists
	exists, err := d.manager.SessionExists(sessionName)
	if err != nil {
		return fmt.Errorf("failed to check session existence: %w", err)
	}
	
	if exists {
		// Determine if it's a test or production session
		isTest := IsTestSession(sessionName)
		isProduction := IsProductionSession(sessionName)
		
		if isProduction && os.Getenv("OSOBA_TEST_MODE") == "true" {
			return fmt.Errorf("cannot use production session %s in test mode", sessionName)
		}
		
		if isTest && os.Getenv("OSOBA_TEST_MODE") != "true" {
			return fmt.Errorf("cannot use test session %s in production mode", sessionName)
		}
	}
	
	return nil
}

// LockSession acquires a lock on a session.
func (d *ConflictDetector) LockSession(sessionName string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	// Check if already locked
	if lock, exists := d.sessionLocks[sessionName]; exists {
		if lock.ProcessID == os.Getpid() {
			// Already locked by this process
			return nil
		}
		return fmt.Errorf("session %s is locked by process %d", sessionName, lock.ProcessID)
	}
	
	// Create lock
	d.sessionLocks[sessionName] = &SessionLock{
		SessionName: sessionName,
		LockTime:    time.Now(),
		ProcessID:   os.Getpid(),
		IsTest:      IsTestSession(sessionName),
	}
	
	return nil
}

// UnlockSession releases a lock on a session.
func (d *ConflictDetector) UnlockSession(sessionName string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	lock, exists := d.sessionLocks[sessionName]
	if !exists {
		return nil // Not locked
	}
	
	if lock.ProcessID != os.Getpid() {
		return fmt.Errorf("cannot unlock session %s locked by process %d", sessionName, lock.ProcessID)
	}
	
	delete(d.sessionLocks, sessionName)
	return nil
}

// CheckPortConflict checks if a port would conflict between test and production.
func (d *ConflictDetector) CheckPortConflict(port int, isTest bool) error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	if isTest {
		if d.productionPorts[port] {
			return fmt.Errorf("port %d is in use by production", port)
		}
	} else {
		if d.testPorts[port] {
			return fmt.Errorf("port %d is in use by tests", port)
		}
	}
	
	return nil
}

// ReservePort reserves a port for test or production use.
func (d *ConflictDetector) ReservePort(port int, isTest bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	// Check for conflicts
	if isTest {
		if d.productionPorts[port] {
			return fmt.Errorf("port %d is already in use by production", port)
		}
		d.testPorts[port] = true
	} else {
		if d.testPorts[port] {
			return fmt.Errorf("port %d is already in use by tests", port)
		}
		d.productionPorts[port] = true
	}
	
	return nil
}

// ReleasePort releases a reserved port.
func (d *ConflictDetector) ReleasePort(port int, isTest bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if isTest {
		delete(d.testPorts, port)
	} else {
		delete(d.productionPorts, port)
	}
}

// ValidateEnvironmentConsistency checks for environment consistency.
func (d *ConflictDetector) ValidateEnvironmentConsistency() error {
	// Check if test mode is consistent
	testMode := os.Getenv("OSOBA_TEST_MODE") == "true"
	testSocket := os.Getenv("OSOBA_TEST_SOCKET")
	testPrefix := os.Getenv("OSOBA_TEST_SESSION_PREFIX")
	
	if testMode {
		// In test mode, check for production sessions
		sessions, err := d.manager.ListSessions("osoba-")
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}
		
		var productionSessions []string
		for _, session := range sessions {
			if IsProductionSession(session) {
				productionSessions = append(productionSessions, session)
			}
		}
		
		if len(productionSessions) > 0 && testSocket == "" {
			// Production sessions exist but no socket isolation
			return fmt.Errorf("found %d production sessions without socket isolation: %v",
				len(productionSessions), productionSessions)
		}
		
		if testPrefix == "" {
			return fmt.Errorf("OSOBA_TEST_SESSION_PREFIX not set in test mode")
		}
	} else {
		// In production mode, check for test sessions
		sessions, err := d.manager.ListSessions("test-")
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}
		
		var testSessions []string
		for _, session := range sessions {
			if IsTestSession(session) {
				testSessions = append(testSessions, session)
			}
		}
		
		if len(testSessions) > 0 {
			// Test sessions exist in production mode
			fmt.Fprintf(os.Stderr, "WARNING: Found %d test sessions in production mode: %v\n",
				len(testSessions), testSessions)
		}
	}
	
	return nil
}

// CleanupStaleLocks removes locks from processes that no longer exist.
func (d *ConflictDetector) CleanupStaleLocks() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	var staleLocks []string
	
	for sessionName, lock := range d.sessionLocks {
		// Check if process still exists
		if !processExists(lock.ProcessID) {
			staleLocks = append(staleLocks, sessionName)
		}
	}
	
	// Remove stale locks
	for _, sessionName := range staleLocks {
		delete(d.sessionLocks, sessionName)
	}
	
	if len(staleLocks) > 0 {
		fmt.Fprintf(os.Stderr, "Cleaned up %d stale session locks\n", len(staleLocks))
	}
	
	return nil
}

// processExists checks if a process with the given PID exists.
func processExists(pid int) bool {
	// Try to send signal 0 to check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	
	err = process.Signal(os.Signal(nil))
	return err == nil
}

// IsolationValidator validates test isolation configuration.
type IsolationValidator struct {
	manager Manager
}

// NewIsolationValidator creates a new isolation validator.
func NewIsolationValidator(manager Manager) *IsolationValidator {
	return &IsolationValidator{
		manager: manager,
	}
}

// ValidateIsolation checks if test isolation is properly configured.
func (v *IsolationValidator) ValidateIsolation() error {
	testMode := os.Getenv("OSOBA_TEST_MODE") == "true"
	testSocket := os.Getenv("OSOBA_TEST_SOCKET")
	
	if !testMode {
		return nil // Not in test mode, no isolation needed
	}
	
	// Check socket isolation
	if testSocket == "" {
		// No socket isolation, check session prefix
		prefix := os.Getenv("OSOBA_TEST_SESSION_PREFIX")
		if prefix == "" || !strings.HasPrefix(prefix, "test") {
			return fmt.Errorf("test mode requires either socket isolation or test session prefix")
		}
	} else {
		// Socket isolation configured, verify it's working
		testManager := NewTestManagerWithSocket(testSocket, "test-validate-")
		
		// Try to create a test session
		testSessionName := fmt.Sprintf("test-validate-%d", os.Getpid())
		if err := testManager.CreateSession(testSessionName); err != nil {
			// Failed to create session with test socket
			return fmt.Errorf("socket isolation not working: %w", err)
		}
		
		// Clean up test session
		_ = testManager.KillSession(testSessionName)
	}
	
	return nil
}

// ValidateNoProductionAccess ensures tests cannot access production sessions.
func (v *IsolationValidator) ValidateNoProductionAccess() error {
	if os.Getenv("OSOBA_TEST_MODE") != "true" {
		return nil // Not in test mode
	}
	
	// List all sessions
	sessions, err := v.manager.ListSessions("")
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}
	
	// Check for production sessions
	for _, session := range sessions {
		if IsProductionSession(session) {
			// Test can see production session - isolation may be incomplete
			if os.Getenv("OSOBA_TEST_SOCKET") == "" {
				return fmt.Errorf("test can access production session %s without socket isolation", session)
			}
		}
	}
	
	return nil
}
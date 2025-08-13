package factories

import (
	"fmt"
	"os"

	"github.com/douhashi/osoba/internal/tmux"
)

// TmuxManagerType represents the type of tmux manager to create.
type TmuxManagerType string

const (
	// TmuxManagerTypeMock creates a mock manager for unit tests.
	TmuxManagerTypeMock TmuxManagerType = "mock"

	// TmuxManagerTypeTest creates a test manager with isolation.
	TmuxManagerTypeTest TmuxManagerType = "test"

	// TmuxManagerTypeReal creates a real tmux manager.
	TmuxManagerTypeReal TmuxManagerType = "real"

	// TmuxManagerTypeAuto automatically selects based on environment.
	TmuxManagerTypeAuto TmuxManagerType = "auto"
)

// TmuxManagerFactory creates tmux managers based on configuration.
type TmuxManagerFactory struct {
	defaultType TmuxManagerType
	testSocket  string
	testPrefix  string
}

// NewTmuxManagerFactory creates a new tmux manager factory.
func NewTmuxManagerFactory() *TmuxManagerFactory {
	return &TmuxManagerFactory{
		defaultType: TmuxManagerTypeAuto,
		testSocket:  os.Getenv("OSOBA_TEST_SOCKET"),
		testPrefix:  os.Getenv("OSOBA_TEST_SESSION_PREFIX"),
	}
}

// SetDefaultType sets the default manager type.
func (f *TmuxManagerFactory) SetDefaultType(managerType TmuxManagerType) {
	f.defaultType = managerType
}

// SetTestSocket sets the test socket path.
func (f *TmuxManagerFactory) SetTestSocket(socket string) {
	f.testSocket = socket
}

// SetTestPrefix sets the test session prefix.
func (f *TmuxManagerFactory) SetTestPrefix(prefix string) {
	f.testPrefix = prefix
}

// Create creates a tmux manager based on the factory configuration.
func (f *TmuxManagerFactory) Create() (tmux.Manager, error) {
	return f.CreateWithType(f.defaultType)
}

// CreateWithType creates a tmux manager of the specified type.
func (f *TmuxManagerFactory) CreateWithType(managerType TmuxManagerType) (tmux.Manager, error) {
	switch managerType {
	case TmuxManagerTypeMock:
		return f.createMockManager(), nil

	case TmuxManagerTypeTest:
		return f.createTestManager(), nil

	case TmuxManagerTypeReal:
		return f.createRealManager(), nil

	case TmuxManagerTypeAuto:
		return f.createAutoManager()

	default:
		return nil, fmt.Errorf("unknown manager type: %s", managerType)
	}
}

// createMockManager creates a mock tmux manager.
func (f *TmuxManagerFactory) createMockManager() tmux.Manager {
	return NewMockTmuxManager()
}

// createTestManager creates a test tmux manager with isolation.
func (f *TmuxManagerFactory) createTestManager() tmux.Manager {
	if f.testSocket != "" {
		return tmux.NewTestManagerWithSocket(f.testSocket, f.testPrefix)
	}
	return tmux.NewTestManager()
}

// createRealManager creates a real tmux manager.
func (f *TmuxManagerFactory) createRealManager() tmux.Manager {
	return tmux.NewDefaultManager()
}

// createAutoManager automatically selects the appropriate manager type.
func (f *TmuxManagerFactory) createAutoManager() (tmux.Manager, error) {
	// Check environment to determine the appropriate type
	if os.Getenv("OSOBA_USE_MOCK_TMUX") == "true" {
		return f.createMockManager(), nil
	}

	if os.Getenv("OSOBA_TEST_MODE") == "true" {
		return f.createTestManager(), nil
	}

	// Check if we're in a test binary
	if isTestBinary() {
		// In test binary, use test manager by default
		return f.createTestManager(), nil
	}

	// Default to real manager
	return f.createRealManager(), nil
}

// isTestBinary checks if the current process is a test binary.
func isTestBinary() bool {
	// Check if running under go test
	for _, arg := range os.Args {
		if arg == "-test.v" || arg == "-test.run" {
			return true
		}
	}

	// Check for test environment variable
	if os.Getenv("GO_TEST") == "1" {
		return true
	}

	return false
}

// GetManager returns a tmux manager based on the current environment.
// This is a convenience function for quick manager creation.
func GetManager() tmux.Manager {
	factory := NewTmuxManagerFactory()
	manager, err := factory.Create()
	if err != nil {
		// Fallback to real manager on error
		return tmux.NewDefaultManager()
	}
	return manager
}

// GetTestManager returns a tmux manager suitable for testing.
// This always returns either a mock or test manager, never a real one.
func GetTestManager() tmux.Manager {
	if os.Getenv("OSOBA_USE_MOCK_TMUX") == "true" {
		return NewMockTmuxManager()
	}
	return tmux.NewTestManager()
}

// ManagerBuilder provides a fluent interface for building tmux managers.
type ManagerBuilder struct {
	factory *TmuxManagerFactory
}

// NewManagerBuilder creates a new manager builder.
func NewManagerBuilder() *ManagerBuilder {
	return &ManagerBuilder{
		factory: NewTmuxManagerFactory(),
	}
}

// WithType sets the manager type.
func (b *ManagerBuilder) WithType(managerType TmuxManagerType) *ManagerBuilder {
	b.factory.SetDefaultType(managerType)
	return b
}

// WithTestSocket sets the test socket.
func (b *ManagerBuilder) WithTestSocket(socket string) *ManagerBuilder {
	b.factory.SetTestSocket(socket)
	return b
}

// WithTestPrefix sets the test session prefix.
func (b *ManagerBuilder) WithTestPrefix(prefix string) *ManagerBuilder {
	b.factory.SetTestPrefix(prefix)
	return b
}

// Build creates the tmux manager.
func (b *ManagerBuilder) Build() (tmux.Manager, error) {
	return b.factory.Create()
}

// MustBuild creates the tmux manager or panics on error.
func (b *ManagerBuilder) MustBuild() tmux.Manager {
	manager, err := b.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build tmux manager: %v", err))
	}
	return manager
}

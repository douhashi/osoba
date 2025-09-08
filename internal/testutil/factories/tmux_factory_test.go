package factories

import (
	"os"
	"testing"

	"github.com/douhashi/osoba/internal/testutil/mocks"
)

func TestTmuxManagerFactory_CreateWithType(t *testing.T) {
	factory := NewTmuxManagerFactory()

	tests := []struct {
		name        string
		managerType TmuxManagerType
		expectError bool
	}{
		{
			name:        "create mock manager",
			managerType: TmuxManagerTypeMock,
			expectError: false,
		},
		{
			name:        "create test manager",
			managerType: TmuxManagerTypeTest,
			expectError: false,
		},
		{
			name:        "create real manager",
			managerType: TmuxManagerTypeReal,
			expectError: false,
		},
		{
			name:        "create auto manager",
			managerType: TmuxManagerTypeAuto,
			expectError: false,
		},
		{
			name:        "invalid manager type",
			managerType: TmuxManagerType("invalid"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := factory.CreateWithType(tt.managerType)
			if (err != nil) != tt.expectError {
				t.Errorf("CreateWithType() error = %v, expectError %v", err, tt.expectError)
			}
			if !tt.expectError && manager == nil {
				t.Error("CreateWithType() returned nil manager")
			}
		})
	}
}

func TestTmuxManagerFactory_AutoSelection(t *testing.T) {
	tests := []struct {
		name       string
		envVars    map[string]string
		expectMock bool
		expectTest bool
	}{
		{
			name: "select mock when OSOBA_USE_MOCK_TMUX is true",
			envVars: map[string]string{
				"OSOBA_USE_MOCK_TMUX": "true",
			},
			expectMock: true,
			expectTest: false,
		},
		{
			name: "select test when OSOBA_TEST_MODE is true",
			envVars: map[string]string{
				"OSOBA_TEST_MODE": "true",
			},
			expectMock: false,
			expectTest: true,
		},
		{
			name: "prefer mock over test mode",
			envVars: map[string]string{
				"OSOBA_USE_MOCK_TMUX": "true",
				"OSOBA_TEST_MODE":     "true",
			},
			expectMock: true,
			expectTest: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore env vars
			origVars := make(map[string]string)
			for k := range tt.envVars {
				origVars[k] = os.Getenv(k)
			}
			defer func() {
				for k, v := range origVars {
					if v == "" {
						os.Unsetenv(k)
					} else {
						os.Setenv(k, v)
					}
				}
			}()

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			factory := NewTmuxManagerFactory()
			manager, err := factory.CreateWithType(TmuxManagerTypeAuto)
			if err != nil {
				t.Fatalf("CreateWithType(auto) error = %v", err)
			}

			// Check type based on interface assertion
			switch m := manager.(type) {
			case *mocks.MockTmuxManager:
				if !tt.expectMock {
					t.Errorf("Expected non-mock manager, got %T", m)
				}
			default:
				if tt.expectMock {
					t.Errorf("Expected mock manager, got %T", m)
				}
			}
		})
	}
}

func TestManagerBuilder(t *testing.T) {
	t.Run("build with type", func(t *testing.T) {
		manager, err := NewManagerBuilder().
			WithType(TmuxManagerTypeMock).
			Build()

		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		if _, ok := manager.(*mocks.MockTmuxManager); !ok {
			t.Errorf("Expected MockTmuxManager, got %T", manager)
		}
	})

	t.Run("build with test socket", func(t *testing.T) {
		manager, err := NewManagerBuilder().
			WithType(TmuxManagerTypeTest).
			WithTestSocket("/tmp/test.sock").
			WithTestPrefix("test-prefix-").
			Build()

		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		if manager == nil {
			t.Error("Build() returned nil manager")
		}
	})

	t.Run("must build panics on error", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustBuild() should panic on invalid type")
			}
		}()

		_ = NewManagerBuilder().
			WithType(TmuxManagerType("invalid")).
			MustBuild()
	})
}

func TestMockTmuxManager(t *testing.T) {
	t.Run("basic mock functionality", func(t *testing.T) {
		// Test that the mock can be created and basic methods work
		mock := mocks.NewMockTmuxManager()
		mock.WithDefaultBehavior()

		// Test basic operations don't panic
		_ = mock.CreateSession("test-session")
		_, _ = mock.SessionExists("test-session")
		_, _ = mock.ListSessions("")
		_ = mock.CreateWindow("test-session", "test-window")
		_, _ = mock.WindowExists("test-session", "test-window")
		_ = mock.SendKeys("test-session", "test-window", "echo hello")
	})
}

func TestGetManager(t *testing.T) {
	// Save and restore env vars
	origTestMode := os.Getenv("OSOBA_TEST_MODE")
	origUseMock := os.Getenv("OSOBA_USE_MOCK_TMUX")
	defer func() {
		if origTestMode == "" {
			os.Unsetenv("OSOBA_TEST_MODE")
		} else {
			os.Setenv("OSOBA_TEST_MODE", origTestMode)
		}
		if origUseMock == "" {
			os.Unsetenv("OSOBA_USE_MOCK_TMUX")
		} else {
			os.Setenv("OSOBA_USE_MOCK_TMUX", origUseMock)
		}
	}()

	t.Run("GetManager returns manager", func(t *testing.T) {
		manager := GetManager()
		if manager == nil {
			t.Error("GetManager() returned nil")
		}
	})

	t.Run("GetTestManager with mock", func(t *testing.T) {
		os.Setenv("OSOBA_USE_MOCK_TMUX", "true")

		manager := GetTestManager()
		if _, ok := manager.(*mocks.MockTmuxManager); !ok {
			t.Errorf("GetTestManager() with OSOBA_USE_MOCK_TMUX should return MockTmuxManager, got %T", manager)
		}
	})

	t.Run("GetTestManager without mock", func(t *testing.T) {
		os.Unsetenv("OSOBA_USE_MOCK_TMUX")

		manager := GetTestManager()
		if manager == nil {
			t.Error("GetTestManager() returned nil")
		}
	})
}

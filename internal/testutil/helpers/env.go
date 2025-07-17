package helpers

import (
	"os"
	"testing"
)

// EnvGuard manages environment variables during tests.
// It saves the original values and restores them after the test.
type EnvGuard struct {
	t        *testing.T
	original map[string]string
}

// NewEnvGuard creates a new EnvGuard for managing environment variables in tests.
func NewEnvGuard(t *testing.T) *EnvGuard {
	t.Helper()
	return &EnvGuard{
		t:        t,
		original: make(map[string]string),
	}
}

// Set sets an environment variable and saves its original value.
// The original value will be restored when Restore is called.
func (g *EnvGuard) Set(key, value string) {
	g.t.Helper()
	if _, saved := g.original[key]; !saved {
		g.original[key] = os.Getenv(key)
	}
	if err := os.Setenv(key, value); err != nil {
		g.t.Fatalf("failed to set env var %s: %v", key, err)
	}
}

// Unset removes an environment variable and saves its original value.
// The original value will be restored when Restore is called.
func (g *EnvGuard) Unset(key string) {
	g.t.Helper()
	if _, saved := g.original[key]; !saved {
		g.original[key] = os.Getenv(key)
	}
	if err := os.Unsetenv(key); err != nil {
		g.t.Fatalf("failed to unset env var %s: %v", key, err)
	}
}

// Restore restores all environment variables to their original values.
// This should be called in a defer statement.
func (g *EnvGuard) Restore() {
	g.t.Helper()
	for key, value := range g.original {
		if value == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, value)
		}
	}
}

// SetEnv is a convenience function for simple environment variable management.
// It returns a cleanup function that should be called to restore the original value.
func SetEnv(t *testing.T, key, value string) func() {
	t.Helper()
	original := os.Getenv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("failed to set env var %s: %v", key, err)
	}
	return func() {
		if original == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, original)
		}
	}
}

// UnsetEnv is a convenience function for removing environment variables.
// It returns a cleanup function that should be called to restore the original value.
func UnsetEnv(t *testing.T, key string) func() {
	t.Helper()
	original := os.Getenv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("failed to unset env var %s: %v", key, err)
	}
	return func() {
		if original != "" {
			os.Setenv(key, original)
		}
	}
}

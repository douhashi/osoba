package helpers

import (
	"os"
	"testing"
)

func TestEnvGuard(t *testing.T) {
	const testKey = "TEST_ENV_GUARD_VAR"
	const originalValue = "original"
	const newValue = "new"

	// Set initial value
	os.Setenv(testKey, originalValue)
	defer os.Unsetenv(testKey)

	t.Run("Set and Restore", func(t *testing.T) {
		guard := NewEnvGuard(t)
		defer guard.Restore()

		// Set new value
		guard.Set(testKey, newValue)
		if got := os.Getenv(testKey); got != newValue {
			t.Errorf("after Set, got %q, want %q", got, newValue)
		}

		// Restore should bring back original
		guard.Restore()
		if got := os.Getenv(testKey); got != originalValue {
			t.Errorf("after Restore, got %q, want %q", got, originalValue)
		}
	})

	t.Run("Unset and Restore", func(t *testing.T) {
		guard := NewEnvGuard(t)
		defer guard.Restore()

		// Unset variable
		guard.Unset(testKey)
		if got := os.Getenv(testKey); got != "" {
			t.Errorf("after Unset, got %q, want empty", got)
		}

		// Restore should bring back original
		guard.Restore()
		if got := os.Getenv(testKey); got != originalValue {
			t.Errorf("after Restore, got %q, want %q", got, originalValue)
		}
	})

	t.Run("Multiple variables", func(t *testing.T) {
		const testKey2 = "TEST_ENV_GUARD_VAR2"
		os.Setenv(testKey2, "value2")
		defer os.Unsetenv(testKey2)

		guard := NewEnvGuard(t)
		defer guard.Restore()

		guard.Set(testKey, "changed1")
		guard.Set(testKey2, "changed2")

		if got := os.Getenv(testKey); got != "changed1" {
			t.Errorf("key1: got %q, want %q", got, "changed1")
		}
		if got := os.Getenv(testKey2); got != "changed2" {
			t.Errorf("key2: got %q, want %q", got, "changed2")
		}

		guard.Restore()

		if got := os.Getenv(testKey); got != originalValue {
			t.Errorf("key1 after restore: got %q, want %q", got, originalValue)
		}
		if got := os.Getenv(testKey2); got != "value2" {
			t.Errorf("key2 after restore: got %q, want %q", got, "value2")
		}
	})
}

func TestSetEnv(t *testing.T) {
	const testKey = "TEST_SET_ENV_VAR"
	const originalValue = "original"
	const newValue = "new"

	os.Setenv(testKey, originalValue)
	defer os.Unsetenv(testKey)

	cleanup := SetEnv(t, testKey, newValue)
	defer cleanup()

	if got := os.Getenv(testKey); got != newValue {
		t.Errorf("after SetEnv, got %q, want %q", got, newValue)
	}

	cleanup()

	if got := os.Getenv(testKey); got != originalValue {
		t.Errorf("after cleanup, got %q, want %q", got, originalValue)
	}
}

func TestUnsetEnv(t *testing.T) {
	const testKey = "TEST_UNSET_ENV_VAR"
	const originalValue = "original"

	os.Setenv(testKey, originalValue)
	defer os.Unsetenv(testKey)

	cleanup := UnsetEnv(t, testKey)
	defer cleanup()

	if got := os.Getenv(testKey); got != "" {
		t.Errorf("after UnsetEnv, got %q, want empty", got)
	}

	cleanup()

	if got := os.Getenv(testKey); got != originalValue {
		t.Errorf("after cleanup, got %q, want %q", got, originalValue)
	}
}

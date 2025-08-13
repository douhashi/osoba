package testenv

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCleanupHandler_Execute(t *testing.T) {
	t.Run("executes cleanups in reverse order", func(t *testing.T) {
		handler := NewCleanupHandler()
		var order []int
		
		handler.RegisterFunc("cleanup1", func() error {
			order = append(order, 1)
			return nil
		})
		
		handler.RegisterFunc("cleanup2", func() error {
			order = append(order, 2)
			return nil
		})
		
		handler.RegisterFunc("cleanup3", func() error {
			order = append(order, 3)
			return nil
		})
		
		ctx := context.Background()
		err := handler.Execute(ctx)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		
		// Check reverse order execution
		expectedOrder := []int{3, 2, 1}
		if len(order) != len(expectedOrder) {
			t.Fatalf("order length = %d, want %d", len(order), len(expectedOrder))
		}
		
		for i, v := range order {
			if v != expectedOrder[i] {
				t.Errorf("order[%d] = %d, want %d", i, v, expectedOrder[i])
			}
		}
	})
	
	t.Run("handles errors", func(t *testing.T) {
		handler := NewCleanupHandler()
		
		handler.RegisterFunc("cleanup1", func() error {
			return nil
		})
		
		handler.RegisterFunc("cleanup2", func() error {
			return errors.New("cleanup2 failed")
		})
		
		handler.RegisterFunc("cleanup3", func() error {
			return nil
		})
		
		ctx := context.Background()
		err := handler.Execute(ctx)
		if err == nil {
			t.Fatal("Execute() should return error")
		}
	})
	
	t.Run("stops on critical error", func(t *testing.T) {
		handler := NewCleanupHandler()
		var executed []string
		
		handler.RegisterFunc("cleanup1", func() error {
			executed = append(executed, "cleanup1")
			return nil
		})
		
		handler.Register(CleanupFunc{
			Name: "critical",
			Fn: func() error {
				executed = append(executed, "critical")
				return errors.New("critical error")
			},
			Critical: true,
		})
		
		handler.RegisterFunc("cleanup3", func() error {
			executed = append(executed, "cleanup3")
			return nil
		})
		
		ctx := context.Background()
		err := handler.Execute(ctx)
		if err == nil {
			t.Fatal("Execute() should return error")
		}
		
		// cleanup3 should execute, critical should execute and fail, cleanup1 should NOT execute
		expectedExecuted := []string{"cleanup3", "critical"}
		if len(executed) != len(expectedExecuted) {
			t.Fatalf("executed = %v, want %v", executed, expectedExecuted)
		}
		
		for i, v := range executed {
			if v != expectedExecuted[i] {
				t.Errorf("executed[%d] = %s, want %s", i, v, expectedExecuted[i])
			}
		}
	})
	
	t.Run("ignores errors when configured", func(t *testing.T) {
		handler := NewCleanupHandler()
		
		handler.Register(CleanupFunc{
			Name: "ignored",
			Fn: func() error {
				return errors.New("ignored error")
			},
			IgnoreError: true,
		})
		
		ctx := context.Background()
		err := handler.Execute(ctx)
		if err != nil {
			t.Fatalf("Execute() error = %v, expected nil", err)
		}
	})
	
	t.Run("executes only once", func(t *testing.T) {
		handler := NewCleanupHandler()
		var count int
		
		handler.RegisterFunc("cleanup", func() error {
			count++
			return nil
		})
		
		ctx := context.Background()
		
		// First execution
		err := handler.Execute(ctx)
		if err != nil {
			t.Fatalf("First Execute() error = %v", err)
		}
		
		// Second execution should be no-op
		err = handler.Execute(ctx)
		if err != nil {
			t.Fatalf("Second Execute() error = %v", err)
		}
		
		if count != 1 {
			t.Errorf("cleanup executed %d times, want 1", count)
		}
	})
	
	t.Run("handles timeout", func(t *testing.T) {
		handler := NewCleanupHandler()
		handler.SetTimeout(100 * time.Millisecond)
		
		handler.RegisterFunc("slow", func() error {
			time.Sleep(200 * time.Millisecond)
			return nil
		})
		
		ctx := context.Background()
		err := handler.Execute(ctx)
		if err == nil {
			t.Fatal("Execute() should return timeout error")
		}
	})
}

func TestCleanupHandler_Reset(t *testing.T) {
	handler := NewCleanupHandler()
	var count int
	
	handler.RegisterFunc("cleanup", func() error {
		count++
		return nil
	})
	
	ctx := context.Background()
	
	// First execution
	err := handler.Execute(ctx)
	if err != nil {
		t.Fatalf("First Execute() error = %v", err)
	}
	
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
	
	// Reset
	handler.Reset()
	
	// Register new cleanup
	handler.RegisterFunc("cleanup2", func() error {
		count++
		return nil
	})
	
	// Execute after reset
	err = handler.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute after reset error = %v", err)
	}
	
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestWithCleanup(t *testing.T) {
	t.Run("executes cleanup on success", func(t *testing.T) {
		handler := NewCleanupHandler()
		var cleanupExecuted bool
		
		handler.RegisterFunc("cleanup", func() error {
			cleanupExecuted = true
			return nil
		})
		
		err := WithCleanup(func() error {
			return nil
		}, handler)
		
		if err != nil {
			t.Fatalf("WithCleanup() error = %v", err)
		}
		
		if !cleanupExecuted {
			t.Error("cleanup not executed")
		}
	})
	
	t.Run("executes cleanup on error", func(t *testing.T) {
		handler := NewCleanupHandler()
		var cleanupExecuted bool
		
		handler.RegisterFunc("cleanup", func() error {
			cleanupExecuted = true
			return nil
		})
		
		err := WithCleanup(func() error {
			return errors.New("function error")
		}, handler)
		
		if err == nil {
			t.Fatal("WithCleanup() should return error")
		}
		
		if !cleanupExecuted {
			t.Error("cleanup not executed after error")
		}
	})
	
	t.Run("executes cleanup on panic when enabled", func(t *testing.T) {
		handler := NewCleanupHandler()
		handler.EnablePanicRecovery()
		
		var cleanupExecuted bool
		handler.RegisterFunc("cleanup", func() error {
			cleanupExecuted = true
			return nil
		})
		
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
			if !cleanupExecuted {
				t.Error("cleanup not executed after panic")
			}
		}()
		
		_ = WithCleanup(func() error {
			panic("test panic")
		}, handler)
	})
}

func TestDeferredCleanup(t *testing.T) {
	t.Run("basic usage", func(t *testing.T) {
		cleanup := NewDeferredCleanup(context.Background())
		var executed []string
		
		cleanup.Register("cleanup1", func() error {
			executed = append(executed, "cleanup1")
			return nil
		})
		
		cleanup.Register("cleanup2", func() error {
			executed = append(executed, "cleanup2")
			return nil
		})
		
		err := cleanup.Cleanup()
		if err != nil {
			t.Fatalf("Cleanup() error = %v", err)
		}
		
		// Check reverse order
		expectedExecuted := []string{"cleanup2", "cleanup1"}
		if len(executed) != len(expectedExecuted) {
			t.Fatalf("executed = %v, want %v", executed, expectedExecuted)
		}
		
		for i, v := range executed {
			if v != expectedExecuted[i] {
				t.Errorf("executed[%d] = %s, want %s", i, v, expectedExecuted[i])
			}
		}
	})
	
	t.Run("MustCleanup panics on error", func(t *testing.T) {
		cleanup := NewDeferredCleanup(context.Background())
		
		cleanup.Register("failing", func() error {
			return errors.New("cleanup failed")
		})
		
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("MustCleanup should panic on error")
			}
		}()
		
		cleanup.MustCleanup()
	})
}

// Mock TestingT for testing TestCleanupManager
type mockTestingT struct {
	cleanupFuncs []func()
	errors       []string
	fatals       []string
}

func (m *mockTestingT) Helper() {}

func (m *mockTestingT) Cleanup(fn func()) {
	m.cleanupFuncs = append(m.cleanupFuncs, fn)
}

func (m *mockTestingT) Errorf(format string, args ...interface{}) {
	m.errors = append(m.errors, format)
}

func (m *mockTestingT) Fatalf(format string, args ...interface{}) {
	m.fatals = append(m.fatals, format)
}

func TestTestCleanupManager(t *testing.T) {
	t.Run("auto execution", func(t *testing.T) {
		mockT := &mockTestingT{}
		manager := NewTestCleanupManager(mockT)
		
		var executed bool
		manager.Register("test-cleanup", func() error {
			executed = true
			return nil
		})
		
		// Simulate test completion
		for _, fn := range mockT.cleanupFuncs {
			fn()
		}
		
		if !executed {
			t.Error("cleanup not executed")
		}
	})
	
	t.Run("critical cleanup", func(t *testing.T) {
		mockT := &mockTestingT{}
		manager := NewTestCleanupManager(mockT)
		
		var executed []string
		
		manager.Register("normal", func() error {
			executed = append(executed, "normal")
			return nil
		})
		
		manager.RegisterCritical("critical", func() error {
			executed = append(executed, "critical")
			return errors.New("critical error")
		})
		
		manager.Register("after-critical", func() error {
			executed = append(executed, "after")
			return nil
		})
		
		// Manually execute to test critical behavior
		manager.DisableAutoExec()
		_ = manager.Execute()
		
		// after should execute, critical should execute and fail, normal should NOT execute
		expectedExecuted := []string{"after", "critical"}
		if len(executed) != len(expectedExecuted) {
			t.Fatalf("executed = %v, want %v", executed, expectedExecuted)
		}
	})
}
package helpers

import (
	"errors"
	"testing"
)

func TestCommandRecorder(t *testing.T) {
	t.Run("Record and retrieve calls", func(t *testing.T) {
		recorder := NewCommandRecorder(t)

		// Record some commands
		recorder.Record("git", []string{"status"}, CommandResult{
			Output: "On branch main",
			Error:  nil,
		})

		recorder.Record("tmux", []string{"new-session"}, CommandResult{
			Output: "",
			Error:  errors.New("tmux not found"),
		})

		// Test GetCalls
		calls := recorder.GetCalls()
		if len(calls) != 2 {
			t.Errorf("expected 2 calls, got %d", len(calls))
		}

		// Test GetCallsForCommand
		gitCalls := recorder.GetCallsForCommand("git")
		if len(gitCalls) != 1 {
			t.Errorf("expected 1 git call, got %d", len(gitCalls))
		}

		if gitCalls[0].Command != "git" {
			t.Errorf("expected command 'git', got %q", gitCalls[0].Command)
		}

		if gitCalls[0].Result.Output != "On branch main" {
			t.Errorf("expected output 'On branch main', got %q", gitCalls[0].Result.Output)
		}
	})

	t.Run("AssertCalled", func(t *testing.T) {
		recorder := NewCommandRecorder(t)
		recorder.Record("git", []string{"status"}, CommandResult{})

		// This should pass
		recorder.AssertCalled("git")

		// Note: We can't test the failure case directly as it would fail the test
	})

	t.Run("AssertNotCalled", func(t *testing.T) {
		recorder := NewCommandRecorder(t)
		recorder.Record("git", []string{"status"}, CommandResult{})

		// This should pass
		recorder.AssertNotCalled("tmux")

		// Note: We can't test the failure case directly as it would fail the test
	})

	t.Run("AssertCallCount", func(t *testing.T) {
		recorder := NewCommandRecorder(t)
		recorder.Record("git", []string{"status"}, CommandResult{})
		recorder.Record("git", []string{"diff"}, CommandResult{})
		recorder.Record("tmux", []string{"ls"}, CommandResult{})

		// This should pass
		recorder.AssertCallCount("git", 2)
		recorder.AssertCallCount("tmux", 1)
		recorder.AssertCallCount("docker", 0)

		// Note: We can't test the failure case directly as it would fail the test
	})

	t.Run("Reset", func(t *testing.T) {
		recorder := NewCommandRecorder(t)
		recorder.Record("git", []string{"status"}, CommandResult{})

		if len(recorder.GetCalls()) != 1 {
			t.Error("expected 1 call before reset")
		}

		recorder.Reset()

		if len(recorder.GetCalls()) != 0 {
			t.Error("expected 0 calls after reset")
		}
	})
}

package helpers

import (
	"errors"
	"fmt"
	"testing"
)

func TestNewTestError(t *testing.T) {
	err := NewTestError("something went wrong")
	want := "test error: something went wrong"
	if err.Error() != want {
		t.Errorf("NewTestError() = %q, want %q", err.Error(), want)
	}
}

func TestNewTestErrorf(t *testing.T) {
	err := NewTestErrorf("failed to %s: %d", "connect", 404)
	want := "test error: failed to connect: 404"
	if err.Error() != want {
		t.Errorf("NewTestErrorf() = %q, want %q", err.Error(), want)
	}
}

func TestWrapTestError(t *testing.T) {
	baseErr := errors.New("base error")
	err := WrapTestError(baseErr, "wrapper")
	want := "test error: wrapper: base error"
	if err.Error() != want {
		t.Errorf("WrapTestError() = %q, want %q", err.Error(), want)
	}

	// Check unwrapping works
	if !errors.Is(err, baseErr) {
		t.Error("wrapped error should match base error with errors.Is")
	}
}

func TestErrorContains(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		substr string
		want   bool
	}{
		{
			name:   "contains substring",
			err:    errors.New("connection refused"),
			substr: "refused",
			want:   true,
		},
		{
			name:   "does not contain substring",
			err:    errors.New("connection refused"),
			substr: "timeout",
			want:   false,
		},
		{
			name:   "nil error",
			err:    nil,
			substr: "anything",
			want:   false,
		},
		{
			name:   "empty substring",
			err:    errors.New("error"),
			substr: "",
			want:   true,
		},
		{
			name:   "exact match",
			err:    errors.New("exact"),
			substr: "exact",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ErrorContains(tt.err, tt.substr)
			if got != tt.want {
				t.Errorf("ErrorContains(%v, %q) = %v, want %v", tt.err, tt.substr, got, tt.want)
			}
		})
	}
}

func TestErrorIs(t *testing.T) {
	err1 := errors.New("error1")
	err2 := fmt.Errorf("wrapped: %w", err1)

	if !ErrorIs(err2, err1) {
		t.Error("ErrorIs should return true for wrapped error")
	}

	if ErrorIs(err1, errors.New("different")) {
		t.Error("ErrorIs should return false for different errors")
	}
}

type customError struct {
	code int
}

func (e *customError) Error() string {
	return fmt.Sprintf("error code: %d", e.code)
}

func TestErrorAs(t *testing.T) {
	err := &customError{code: 404}
	wrapped := fmt.Errorf("wrapped: %w", err)

	var target *customError
	if !ErrorAs(wrapped, &target) {
		t.Error("ErrorAs should return true for matching type")
	}

	if target.code != 404 {
		t.Errorf("ErrorAs target.code = %d, want 404", target.code)
	}
}

func TestCommonErrors(t *testing.T) {
	// Test that common errors are defined
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrTest", ErrTest, "test error"},
		{"ErrNotFound", ErrNotFound, "not found"},
		{"ErrTimeout", ErrTimeout, "timeout"},
		{"ErrConnection", ErrConnection, "connection error"},
		{"ErrPermission", ErrPermission, "permission denied"},
		{"ErrAPILimit", ErrAPILimit, "API rate limit exceeded"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.msg {
				t.Errorf("%s = %q, want %q", tt.name, tt.err.Error(), tt.msg)
			}
		})
	}
}

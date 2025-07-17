package helpers

import (
	"testing"
	"time"
)

// MustParseTime parses a time string using RFC3339 format and panics on error.
// This is useful for test setup where the time string is hardcoded and should be valid.
func MustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	parsedTime, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("failed to parse time %q: %v", s, err)
	}
	return parsedTime
}

// TimePtr returns a pointer to the given time.
// This is useful for creating optional time fields in test data.
func TimePtr(t time.Time) *time.Time {
	return &t
}

// NowPtr returns a pointer to the current time.
// This is useful for creating optional time fields that need the current time.
func NowPtr() *time.Time {
	now := time.Now()
	return &now
}

// MustParseDuration parses a duration string and panics on error.
// This is useful for test setup where the duration string is hardcoded and should be valid.
func MustParseDuration(t *testing.T, s string) time.Duration {
	t.Helper()
	d, err := time.ParseDuration(s)
	if err != nil {
		t.Fatalf("failed to parse duration %q: %v", s, err)
	}
	return d
}

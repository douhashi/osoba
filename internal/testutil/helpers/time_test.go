package helpers

import (
	"testing"
	"time"
)

func TestMustParseTime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantTime time.Time
	}{
		{
			name:     "valid RFC3339 time",
			input:    "2023-01-01T00:00:00Z",
			wantTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "valid RFC3339 time with timezone",
			input:    "2023-01-01T09:00:00+09:00",
			wantTime: time.Date(2023, 1, 1, 9, 0, 0, 0, time.FixedZone("", 9*60*60)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MustParseTime(t, tt.input)
			if !got.Equal(tt.wantTime) {
				t.Errorf("MustParseTime() = %v, want %v", got, tt.wantTime)
			}
		})
	}
}

func TestTimePtr(t *testing.T) {
	now := time.Now()
	ptr := TimePtr(now)
	if ptr == nil {
		t.Fatal("TimePtr() returned nil")
	}
	if !ptr.Equal(now) {
		t.Errorf("TimePtr() = %v, want %v", *ptr, now)
	}
}

func TestNowPtr(t *testing.T) {
	before := time.Now()
	ptr := NowPtr()
	after := time.Now()

	if ptr == nil {
		t.Fatal("NowPtr() returned nil")
	}
	if ptr.Before(before) || ptr.After(after) {
		t.Errorf("NowPtr() = %v, want time between %v and %v", *ptr, before, after)
	}
}

func TestMustParseDuration(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantDuration time.Duration
	}{
		{
			name:         "seconds",
			input:        "10s",
			wantDuration: 10 * time.Second,
		},
		{
			name:         "minutes",
			input:        "5m",
			wantDuration: 5 * time.Minute,
		},
		{
			name:         "hours",
			input:        "2h",
			wantDuration: 2 * time.Hour,
		},
		{
			name:         "complex duration",
			input:        "1h30m45s",
			wantDuration: 1*time.Hour + 30*time.Minute + 45*time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MustParseDuration(t, tt.input)
			if got != tt.wantDuration {
				t.Errorf("MustParseDuration() = %v, want %v", got, tt.wantDuration)
			}
		})
	}
}

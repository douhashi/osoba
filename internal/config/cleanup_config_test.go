package config

import (
	"testing"
	"time"
)

func TestCleanupConfig_Defaults(t *testing.T) {
	tests := []struct {
		name     string
		config   *CleanupConfig
		expected CleanupConfig
	}{
		{
			name:   "empty config should fill interval",
			config: &CleanupConfig{},
			expected: CleanupConfig{
				Enabled:         false, // SetDefaultsはIntervalMinutesのみ設定
				IntervalMinutes: 5,
				IssueWindows: IssueWindowsConfig{
					Enabled: false,
				},
			},
		},
		{
			name: "config with interval set",
			config: &CleanupConfig{
				IntervalMinutes: 10,
			},
			expected: CleanupConfig{
				Enabled:         false,
				IntervalMinutes: 10, // 既に設定されているので変更なし
				IssueWindows: IssueWindowsConfig{
					Enabled: false,
				},
			},
		},
		{
			name: "config with enabled true",
			config: &CleanupConfig{
				Enabled: true,
			},
			expected: CleanupConfig{
				Enabled:         true,
				IntervalMinutes: 5, // デフォルト値が設定される
				IssueWindows: IssueWindowsConfig{
					Enabled: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result CleanupConfig
			if tt.config != nil {
				result = *tt.config
				t.Logf("Before SetDefaults: Enabled=%v, IntervalMinutes=%v, IssueWindows.Enabled=%v",
					result.Enabled, result.IntervalMinutes, result.IssueWindows.Enabled)
			}
			result.SetDefaults()
			t.Logf("After SetDefaults: Enabled=%v, IntervalMinutes=%v, IssueWindows.Enabled=%v",
				result.Enabled, result.IntervalMinutes, result.IssueWindows.Enabled)

			if result.Enabled != tt.expected.Enabled {
				t.Errorf("Enabled = %v, want %v", result.Enabled, tt.expected.Enabled)
			}
			if result.IntervalMinutes != tt.expected.IntervalMinutes {
				t.Errorf("IntervalMinutes = %v, want %v", result.IntervalMinutes, tt.expected.IntervalMinutes)
			}
			if result.IssueWindows.Enabled != tt.expected.IssueWindows.Enabled {
				t.Errorf("IssueWindows.Enabled = %v, want %v", result.IssueWindows.Enabled, tt.expected.IssueWindows.Enabled)
			}
		})
	}
}

func TestCleanupConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  CleanupConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: CleanupConfig{
				Enabled:         true,
				IntervalMinutes: 10,
				IssueWindows: IssueWindowsConfig{
					Enabled: true,
				},
			},
			wantErr: false,
		},
		{
			name: "interval too small",
			config: CleanupConfig{
				Enabled:         true,
				IntervalMinutes: 0,
				IssueWindows: IssueWindowsConfig{
					Enabled: true,
				},
			},
			wantErr: true,
			errMsg:  "cleanup interval must be between 1 and 60 minutes",
		},
		{
			name: "interval too large",
			config: CleanupConfig{
				Enabled:         true,
				IntervalMinutes: 61,
				IssueWindows: IssueWindowsConfig{
					Enabled: true,
				},
			},
			wantErr: true,
			errMsg:  "cleanup interval must be between 1 and 60 minutes",
		},
		{
			name: "disabled cleanup is valid",
			config: CleanupConfig{
				Enabled:         false,
				IntervalMinutes: 0, // Should not be validated when disabled
				IssueWindows: IssueWindowsConfig{
					Enabled: false,
				},
			},
			wantErr: false,
		},
		{
			name: "minimum valid interval",
			config: CleanupConfig{
				Enabled:         true,
				IntervalMinutes: 1,
				IssueWindows: IssueWindowsConfig{
					Enabled: true,
				},
			},
			wantErr: false,
		},
		{
			name: "maximum valid interval",
			config: CleanupConfig{
				Enabled:         true,
				IntervalMinutes: 60,
				IssueWindows: IssueWindowsConfig{
					Enabled: true,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Validate() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestCleanupConfig_GetInterval(t *testing.T) {
	tests := []struct {
		name     string
		config   CleanupConfig
		expected time.Duration
	}{
		{
			name: "5 minutes",
			config: CleanupConfig{
				IntervalMinutes: 5,
			},
			expected: 5 * time.Minute,
		},
		{
			name: "10 minutes",
			config: CleanupConfig{
				IntervalMinutes: 10,
			},
			expected: 10 * time.Minute,
		},
		{
			name: "1 minute",
			config: CleanupConfig{
				IntervalMinutes: 1,
			},
			expected: 1 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetInterval()
			if result != tt.expected {
				t.Errorf("GetInterval() = %v, want %v", result, tt.expected)
			}
		})
	}
}

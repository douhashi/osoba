package watcher

import (
	"testing"
	"time"
)

func TestIssueWatcher_SetPollInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "正常系: 5秒を設定できる",
			interval: 5 * time.Second,
			wantErr:  false,
		},
		{
			name:     "正常系: 1分を設定できる",
			interval: time.Minute,
			wantErr:  false,
		},
		{
			name:     "正常系: 1時間を設定できる",
			interval: time.Hour,
			wantErr:  false,
		},
		{
			name:     "正常系: 最小値の1秒を設定できる",
			interval: time.Second,
			wantErr:  false,
		},
		{
			name:     "異常系: 1秒未満は設定できない",
			interval: 500 * time.Millisecond,
			wantErr:  true,
			errMsg:   "poll interval must be at least 1 second",
		},
		{
			name:     "異常系: 0は設定できない",
			interval: 0,
			wantErr:  true,
			errMsg:   "poll interval must be at least 1 second",
		},
		{
			name:     "異常系: 負の値は設定できない",
			interval: -1 * time.Second,
			wantErr:  true,
			errMsg:   "poll interval must be at least 1 second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockGitHubClient{}
			watcher, err := NewIssueWatcher(mockClient, "douhashi", "osoba", "test-session", []string{"status:needs-plan"})
			if err != nil {
				t.Fatalf("failed to create watcher: %v", err)
			}

			err = watcher.SetPollInterval(tt.interval)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetPollInterval() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("SetPollInterval() error = %v, want %v", err.Error(), tt.errMsg)
			}

			// 設定が反映されているか確認
			if !tt.wantErr && watcher.GetPollInterval() != tt.interval {
				t.Errorf("GetPollInterval() = %v, want %v", watcher.GetPollInterval(), tt.interval)
			}
		})
	}
}

func TestIssueWatcher_GetPollInterval(t *testing.T) {
	t.Run("デフォルトのポーリング間隔が5秒であること", func(t *testing.T) {
		mockClient := &mockGitHubClient{}
		watcher, err := NewIssueWatcher(mockClient, "douhashi", "osoba", "test-session", []string{"status:needs-plan"})
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		defaultInterval := watcher.GetPollInterval()
		if defaultInterval != 5*time.Second {
			t.Errorf("GetPollInterval() = %v, want %v", defaultInterval, 5*time.Second)
		}
	})

	t.Run("設定した値が正しく取得できること", func(t *testing.T) {
		mockClient := &mockGitHubClient{}
		watcher, err := NewIssueWatcher(mockClient, "douhashi", "osoba", "test-session", []string{"status:needs-plan"})
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		testInterval := 30 * time.Second
		err = watcher.SetPollInterval(testInterval)
		if err != nil {
			t.Fatalf("failed to set poll interval: %v", err)
		}

		gotInterval := watcher.GetPollInterval()
		if gotInterval != testInterval {
			t.Errorf("GetPollInterval() = %v, want %v", gotInterval, testInterval)
		}
	})
}

// SetPollIntervalがエラーを返すようにする必要があるため、
// watcher.goでSetPollIntervalメソッドを修正する必要があります。
// 現在のメソッドシグネチャを確認して、エラーを返すように修正します。

func TestPollIntervalValidation(t *testing.T) {
	t.Run("ValidatePollInterval関数の動作確認", func(t *testing.T) {
		tests := []struct {
			name     string
			interval time.Duration
			wantErr  bool
		}{
			{
				name:     "valid: 1 second",
				interval: time.Second,
				wantErr:  false,
			},
			{
				name:     "valid: 5 seconds",
				interval: 5 * time.Second,
				wantErr:  false,
			},
			{
				name:     "invalid: 0",
				interval: 0,
				wantErr:  true,
			},
			{
				name:     "invalid: negative",
				interval: -1 * time.Second,
				wantErr:  true,
			},
			{
				name:     "invalid: less than 1 second",
				interval: 999 * time.Millisecond,
				wantErr:  true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := ValidatePollInterval(tt.interval)
				if (err != nil) != tt.wantErr {
					t.Errorf("ValidatePollInterval() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})
}

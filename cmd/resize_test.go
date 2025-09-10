package cmd

import (
	"bytes"
	"os"
	"strconv"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewResizeCmd(t *testing.T) {
	cmd := newResizeCmd()

	if cmd.Use != "resize [issue-number]" {
		t.Errorf("expected Use to be 'resize [issue-number]', got %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	// フラグの確認
	dryRunFlag := cmd.Flags().Lookup("dry-run")
	if dryRunFlag == nil {
		t.Error("expected --dry-run flag to be defined")
	}

	sessionFlag := cmd.Flags().Lookup("session")
	if sessionFlag == nil {
		t.Error("expected --session flag to be defined")
	}
}

func TestResizeCmd_ArgumentParsing(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()

	// HOME環境変数を設定（CI環境での失敗を防ぐ）
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	tests := []struct {
		name           string
		args           []string
		expectError    bool
		expectedIssue  int
		expectedDryRun bool
	}{
		{
			name:           "引数なし",
			args:           []string{},
			expectError:    false,
			expectedIssue:  0, // 現在のウィンドウを検出
			expectedDryRun: false,
		},
		{
			name:           "Issue番号指定",
			args:           []string{"123"},
			expectError:    false,
			expectedIssue:  123,
			expectedDryRun: false,
		},
		{
			name:           "dry-runフラグ付き",
			args:           []string{"--dry-run"},
			expectError:    false,
			expectedIssue:  0,
			expectedDryRun: true,
		},
		{
			name:           "Issue番号とdry-runフラグ",
			args:           []string{"456", "--dry-run"},
			expectError:    false,
			expectedIssue:  456,
			expectedDryRun: true,
		},
		{
			name:        "無効なIssue番号",
			args:        []string{"invalid"},
			expectError: true,
		},
		{
			name:        "負の数",
			args:        []string{"-1"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newResizeCmd()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// 引数を設定
			cmd.SetArgs(tt.args)

			// コマンドの実行をモック化するため、RunE関数を上書き
			cmd.RunE = func(cmd *cobra.Command, args []string) error {
				// 引数の解析テスト
				var issueNumber int
				var err error

				if len(args) > 0 {
					issueNumber, err = strconv.Atoi(args[0])
					if err != nil || issueNumber <= 0 {
						return err
					}
				}

				dryRun, _ := cmd.Flags().GetBool("dry-run")

				// 期待値と比較
				if issueNumber != tt.expectedIssue {
					t.Errorf("expected issue number %d, got %d", tt.expectedIssue, issueNumber)
				}
				if dryRun != tt.expectedDryRun {
					t.Errorf("expected dry-run %v, got %v", tt.expectedDryRun, dryRun)
				}

				return nil
			}

			err := cmd.Execute()

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestResizeCmd_Help(t *testing.T) {
	cmd := newResizeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("unexpected error executing help: %v", err)
	}

	output := buf.String()
	expectedStrings := []string{
		"resize",
		"ペインを均等にリサイズ",
		"dry-run",
		"session",
	}

	for _, expected := range expectedStrings {
		if !bytes.Contains([]byte(output), []byte(expected)) {
			t.Errorf("help output should contain '%s', got: %s", expected, output)
		}
	}
}

package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestStatusCmd(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		wantErr            bool
		wantOutputContains []string
	}{
		{
			name:    "正常系: statusコマンドヘルプ",
			args:    []string{"status", "--help"},
			wantErr: false,
			wantOutputContains: []string{
				"status",
				"実行中の開発セッション",
			},
		},
		{
			name:    "正常系: statusコマンド実行",
			args:    []string{"status"},
			wantErr: false,
			wantOutputContains: []string{
				"現在のステータス:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)

			rootCmd = newRootCmd()
			rootCmd.AddCommand(newStatusCmd())
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			output := buf.String()
			for _, want := range tt.wantOutputContains {
				if !strings.Contains(output, want) {
					t.Errorf("Execute() output = %v, want to contain %v", output, want)
				}
			}
		})
	}
}

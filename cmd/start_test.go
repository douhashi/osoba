package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestStartCmd(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		wantErr            bool
		wantOutputContains []string
	}{
		{
			name:    "正常系: startコマンドヘルプ",
			args:    []string{"start", "--help"},
			wantErr: false,
			wantOutputContains: []string{
				"start",
				"開発セッションを開始",
			},
		},
		{
			name:    "正常系: startコマンド実行",
			args:    []string{"start"},
			wantErr: false,
			wantOutputContains: []string{
				"開発セッションを開始しました",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)

			rootCmd = newRootCmd()
			rootCmd.AddCommand(newStartCmd())
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

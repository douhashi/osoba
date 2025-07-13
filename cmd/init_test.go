package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestInitCmd(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		wantErr            bool
		wantOutputContains []string
	}{
		{
			name:    "正常系: initコマンドヘルプ",
			args:    []string{"init", "--help"},
			wantErr: false,
			wantOutputContains: []string{
				"init",
				"osobaプロジェクトのための初期設定",
			},
		},
		{
			name:    "正常系: initコマンド実行",
			args:    []string{"init"},
			wantErr: false,
			wantOutputContains: []string{
				"初期化が完了しました",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)

			rootCmd = newRootCmd()
			rootCmd.AddCommand(newInitCmd())
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

package cmd

import (
	"bytes"
	"os"
	"testing"
)

func TestCheckConfigFileExists(t *testing.T) {
	// 元のワーキングディレクトリを保存
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWd)

	// テスト用の一時ディレクトリを作成し、そこに移動
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name            string
		setupFiles      func()
		wantErr         bool
		wantErrContains string
	}{
		{
			name: "カレントディレクトリに.osoba.ymlが存在する場合",
			setupFiles: func() {
				os.WriteFile(".osoba.yml", []byte("test"), 0644)
			},
			wantErr: false,
		},
		{
			name: "カレントディレクトリに.osoba.yamlが存在する場合",
			setupFiles: func() {
				os.WriteFile(".osoba.yaml", []byte("test"), 0644)
			},
			wantErr: false,
		},
		{
			name: "設定ファイルが存在しない場合",
			setupFiles: func() {
				// 何もしない（ファイルを作成しない）
			},
			wantErr:         true,
			wantErrContains: "設定ファイルが見つかりません",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// クリーンアップ
			os.Remove(".osoba.yml")
			os.Remove(".osoba.yaml")

			tt.setupFiles()

			errBuf := new(bytes.Buffer)
			err := checkConfigFileExists(errBuf)

			if (err != nil) != tt.wantErr {
				t.Errorf("checkConfigFileExists() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErrContains != "" {
				errOutput := errBuf.String()
				if !bytes.Contains([]byte(errOutput), []byte(tt.wantErrContains)) {
					t.Errorf("checkConfigFileExists() error output = %v, want to contain %v", errOutput, tt.wantErrContains)
				}
			}
		})
	}
}

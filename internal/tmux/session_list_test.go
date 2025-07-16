package tmux

import (
	"os/exec"
	"testing"
)

func TestListSessions(t *testing.T) {
	tests := []struct {
		name        string
		prefix      string
		mockOutput  string
		mockError   bool
		expectError bool
		expectCount int
	}{
		{
			name:        "正常系: セッションなし",
			prefix:      "osoba-",
			mockOutput:  "",
			mockError:   true, // tmux returns exit code 1 when no sessions
			expectError: false,
			expectCount: 0,
		},
		{
			name:   "正常系: osoba-prefixのセッションのみ",
			prefix: "osoba-",
			mockOutput: "osoba-test:2:1640995200:1\n" +
				"osoba-main:3:1640995100:0\n" +
				"other-session:1:1640995000:1",
			mockError:   false,
			expectError: false,
			expectCount: 2, // only osoba- prefixed sessions
		},
		{
			name:   "正常系: prefixなし（全セッション）",
			prefix: "",
			mockOutput: "osoba-test:2:1640995200:1\n" +
				"osoba-main:3:1640995100:0\n" +
				"other-session:1:1640995000:1",
			mockError:   false,
			expectError: false,
			expectCount: 3, // all sessions
		},
		{
			name:   "正常系: セッション情報の解析",
			prefix: "test-",
			mockOutput: "test-session:5:1640995200:1\n" +
				"test-another:2:1640995100:0",
			mockError:   false,
			expectError: false,
			expectCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックのexecCommandを設定
			originalExecCommand := execCommand
			defer func() { execCommand = originalExecCommand }()

			execCommand = func(name string, args ...string) *exec.Cmd {
				if tt.mockError {
					return exec.Command("false") // コマンドが失敗するようにする
				}
				// echo を使って期待する出力を返す
				return exec.Command("echo", "-n", tt.mockOutput)
			}

			sessions, err := ListSessionsAsSessionInfo(tt.prefix)

			if tt.expectError && err == nil {
				t.Errorf("期待するエラーが発生しませんでした")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("予期しないエラーが発生しました: %v", err)
				return
			}

			if len(sessions) != tt.expectCount {
				t.Errorf("期待するセッション数 = %d, 実際 = %d", tt.expectCount, len(sessions))
				return
			}

			// セッション情報の詳細チェック
			if tt.name == "正常系: セッション情報の解析" && len(sessions) >= 2 {
				// 最初のセッション
				if sessions[0].Name != "test-session" {
					t.Errorf("セッション名 = %s, 期待値 = test-session", sessions[0].Name)
				}
				if sessions[0].Windows != 5 {
					t.Errorf("ウィンドウ数 = %d, 期待値 = 5", sessions[0].Windows)
				}
				if !sessions[0].Attached {
					t.Errorf("アタッチ状態 = %t, 期待値 = true", sessions[0].Attached)
				}

				// 2番目のセッション
				if sessions[1].Name != "test-another" {
					t.Errorf("セッション名 = %s, 期待値 = test-another", sessions[1].Name)
				}
				if sessions[1].Windows != 2 {
					t.Errorf("ウィンドウ数 = %d, 期待値 = 2", sessions[1].Windows)
				}
				if sessions[1].Attached {
					t.Errorf("アタッチ状態 = %t, 期待値 = false", sessions[1].Attached)
				}
			}
		})
	}
}

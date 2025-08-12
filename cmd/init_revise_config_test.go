package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/douhashi/osoba/internal/config"
	"gopkg.in/yaml.v3"
)

// TestInitCmd_ReviseConfigIncluded は、osoba init で作成される設定ファイルに revise の設定が含まれることを確認する
func TestInitCmd_ReviseConfigIncluded(t *testing.T) {
	// テンプレートから設定ファイルの内容を読み込む
	templateContent, err := templateFS.ReadFile("templates/config.yml")
	if err != nil {
		t.Fatalf("Failed to read template: %v", err)
	}

	// YAML パースして revise 設定があることを確認
	var configData map[string]interface{}
	if err := yaml.Unmarshal(templateContent, &configData); err != nil {
		t.Fatalf("Failed to parse template YAML: %v", err)
	}

	// claude.phases.revise が存在することを確認
	claude, ok := configData["claude"].(map[string]interface{})
	if !ok {
		t.Fatal("claude section not found in template")
	}

	phases, ok := claude["phases"].(map[string]interface{})
	if !ok {
		t.Fatal("claude.phases section not found in template")
	}

	revise, ok := phases["revise"].(map[string]interface{})
	if !ok {
		t.Fatal("claude.phases.revise section not found in template")
	}

	// revise フェーズの設定内容を確認
	args, ok := revise["args"].([]interface{})
	if !ok {
		t.Fatal("revise.args not found or not a list")
	}

	prompt, ok := revise["prompt"].(string)
	if !ok {
		t.Fatal("revise.prompt not found or not a string")
	}

	// 期待する値と比較
	expectedArgs := []interface{}{"--dangerously-skip-permissions"}
	expectedPrompt := "/osoba:revise {{issue-number}}"

	if len(args) != len(expectedArgs) {
		t.Errorf("revise.args length = %d, want %d", len(args), len(expectedArgs))
	}

	for i, arg := range args {
		if i < len(expectedArgs) && arg != expectedArgs[i] {
			t.Errorf("revise.args[%d] = %v, want %v", i, arg, expectedArgs[i])
		}
	}

	if prompt != expectedPrompt {
		t.Errorf("revise.prompt = %v, want %v", prompt, expectedPrompt)
	}
}

// TestInitCmd_CreatedConfigContainsRevise は、実際に init コマンドで作成される設定ファイルに revise 設定が含まれることを確認する
func TestInitCmd_CreatedConfigContainsRevise(t *testing.T) {
	// テスト用の一時ディレクトリ
	tempDir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	// モック関数の設定
	origIsGitRepo := isGitRepositoryFunc
	origCheckCommand := checkCommandFunc
	origMkdirAll := mkdirAllFunc
	origWriteFile := writeFileFunc
	origStat := statFunc
	origGetGitHubToken := getGitHubTokenFunc
	defer func() {
		isGitRepositoryFunc = origIsGitRepo
		checkCommandFunc = origCheckCommand
		mkdirAllFunc = origMkdirAll
		writeFileFunc = origWriteFile
		statFunc = origStat
		getGitHubTokenFunc = origGetGitHubToken
	}()

	var createdFileContent []byte
	isGitRepositoryFunc = func(path string) (bool, error) {
		return true, nil
	}
	checkCommandFunc = func(cmd string) error {
		return nil
	}
	mkdirAllFunc = func(path string, perm os.FileMode) error {
		return nil
	}
	statFunc = func(name string) (os.FileInfo, error) {
		return nil, os.ErrNotExist // ファイルが存在しない
	}
	writeFileFunc = func(path string, data []byte, perm os.FileMode) error {
		if strings.HasSuffix(path, ".osoba.yml") {
			createdFileContent = make([]byte, len(data))
			copy(createdFileContent, data)
		}
		return nil
	}
	getGitHubTokenFunc = func(cfg *config.Config) (string, string) {
		return "", "" // トークンなし
	}

	// init コマンドを実行
	buf := new(bytes.Buffer)
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newInitCmd())
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"init"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	// 作成されたファイルの内容を確認
	if len(createdFileContent) == 0 {
		t.Fatal("config file was not created")
	}

	// YAML パースして revise 設定があることを確認
	var configData map[string]interface{}
	if err := yaml.Unmarshal(createdFileContent, &configData); err != nil {
		t.Fatalf("Failed to parse created config YAML: %v", err)
	}

	// claude.phases.revise が存在することを確認
	claude, ok := configData["claude"].(map[string]interface{})
	if !ok {
		t.Fatal("claude section not found in created config")
	}

	phases, ok := claude["phases"].(map[string]interface{})
	if !ok {
		t.Fatal("claude.phases section not found in created config")
	}

	revise, ok := phases["revise"].(map[string]interface{})
	if !ok {
		t.Fatal("claude.phases.revise section not found in created config")
	}

	// revise フェーズの設定内容を確認
	prompt, ok := revise["prompt"].(string)
	if !ok {
		t.Fatal("revise.prompt not found or not a string")
	}

	expectedPrompt := "/osoba:revise {{issue-number}}"
	if prompt != expectedPrompt {
		t.Errorf("revise.prompt = %v, want %v", prompt, expectedPrompt)
	}
}

// TestConfigDefaults_RevisePhaseConfig は、設定ファイルにrevise設定がない場合でもデフォルト値が適用されることを確認する
func TestConfigDefaults_RevisePhaseConfig(t *testing.T) {
	// revise設定を含まない設定ファイルの内容
	configContent := `
github:
  poll_interval: 20s
tmux:
  session_prefix: "osoba-"
claude:
  phases:
    plan:
      args: ["--dangerously-skip-permissions"]
      prompt: "/osoba:plan {{issue-number}}"
    implement:
      args: ["--dangerously-skip-permissions"]
      prompt: "/osoba:implement {{issue-number}}"
    review:
      args: ["--dangerously-skip-permissions"]
      prompt: "/osoba:review {{issue-number}}"
`

	// 一時ファイルに設定を書き込み
	tempDir := t.TempDir()
	configPath := tempDir + "/test-config.yml"
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// 設定を読み込み
	cfg := config.NewConfig()
	if err := cfg.Load(configPath); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// revise フェーズの設定が存在することを確認
	reviseConfig, exists := cfg.Claude.GetPhase("revise")
	if !exists {
		t.Fatal("revise phase config not found after loading config with defaults")
	}

	// デフォルト値が設定されていることを確認
	expectedArgs := []string{"--dangerously-skip-permissions"}
	expectedPrompt := "/osoba:revise {{issue-number}}"

	if len(reviseConfig.Args) != len(expectedArgs) {
		t.Errorf("revise args length = %d, want %d", len(reviseConfig.Args), len(expectedArgs))
	}

	for i, arg := range reviseConfig.Args {
		if i < len(expectedArgs) && arg != expectedArgs[i] {
			t.Errorf("revise args[%d] = %v, want %v", i, arg, expectedArgs[i])
		}
	}

	if reviseConfig.Prompt != expectedPrompt {
		t.Errorf("revise prompt = %v, want %v", reviseConfig.Prompt, expectedPrompt)
	}
}

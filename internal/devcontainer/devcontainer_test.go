package devcontainer

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DevContainer構成のテスト
type DevContainerConfig struct {
	Name           string                 `json:"name"`
	Build          BuildConfig            `json:"build,omitempty"`
	DockerFile     string                 `json:"dockerFile,omitempty"`
	Context        string                 `json:"context,omitempty"`
	RunArgs        []string               `json:"runArgs,omitempty"`
	Mounts         []string               `json:"mounts,omitempty"`
	Features       map[string]interface{} `json:"features,omitempty"`
	Customizations Customizations         `json:"customizations,omitempty"`
	ForwardPorts   []int                  `json:"forwardPorts,omitempty"`
	PostCreateCmd  interface{}            `json:"postCreateCommand,omitempty"`
	PostStartCmd   interface{}            `json:"postStartCommand,omitempty"`
	PostAttachCmd  interface{}            `json:"postAttachCommand,omitempty"`
	RemoteUser     string                 `json:"remoteUser,omitempty"`
	ContainerEnv   map[string]string      `json:"containerEnv,omitempty"`
	RemoteEnv      map[string]string      `json:"remoteEnv,omitempty"`
}

type BuildConfig struct {
	Dockerfile string            `json:"dockerfile"`
	Context    string            `json:"context"`
	Args       map[string]string `json:"args,omitempty"`
}

type Customizations struct {
	VSCode VSCodeConfig `json:"vscode,omitempty"`
}

type VSCodeConfig struct {
	Extensions []string               `json:"extensions,omitempty"`
	Settings   map[string]interface{} `json:"settings,omitempty"`
}

func TestDevContainerConfig(t *testing.T) {
	// .devcontainer/devcontainer.jsonの存在確認
	t.Run("devcontainer.json exists", func(t *testing.T) {
		configPath := filepath.Join("..", "..", ".devcontainer", "devcontainer.json")
		_, err := os.Stat(configPath)
		if os.IsNotExist(err) {
			t.Skip("DevContainer configuration not yet created")
		}
		require.NoError(t, err, "devcontainer.json should exist")
	})

	// devcontainer.jsonの構造検証
	t.Run("devcontainer.json is valid", func(t *testing.T) {
		configPath := filepath.Join("..", "..", ".devcontainer", "devcontainer.json")
		data, err := os.ReadFile(configPath)
		if os.IsNotExist(err) {
			t.Skip("DevContainer configuration not yet created")
		}
		require.NoError(t, err)

		var config DevContainerConfig
		err = json.Unmarshal(data, &config)
		require.NoError(t, err, "devcontainer.json should be valid JSON")

		// 必須フィールドの確認
		assert.NotEmpty(t, config.Name, "DevContainer should have a name")
		assert.Contains(t, strings.ToLower(config.Name), "osoba", "Name should contain 'osoba'")
	})

	// Go開発環境の設定確認
	t.Run("Go development environment", func(t *testing.T) {
		configPath := filepath.Join("..", "..", ".devcontainer", "devcontainer.json")
		data, err := os.ReadFile(configPath)
		if os.IsNotExist(err) {
			t.Skip("DevContainer configuration not yet created")
		}
		require.NoError(t, err)

		var config DevContainerConfig
		err = json.Unmarshal(data, &config)
		require.NoError(t, err)

		// VS Code Go拡張機能の確認
		extensions := config.Customizations.VSCode.Extensions
		assert.Contains(t, extensions, "golang.go", "Should include Go extension")

		// GitHub関連拡張機能の確認
		hasGitHubExtension := false
		for _, ext := range extensions {
			if strings.Contains(ext, "github") || strings.Contains(ext, "GitHub") {
				hasGitHubExtension = true
				break
			}
		}
		assert.True(t, hasGitHubExtension, "Should include GitHub extension")
	})

	// Rails関連設定が削除されていることを確認
	t.Run("Rails configurations removed", func(t *testing.T) {
		configPath := filepath.Join("..", "..", ".devcontainer", "devcontainer.json")
		data, err := os.ReadFile(configPath)
		if os.IsNotExist(err) {
			t.Skip("DevContainer configuration not yet created")
		}
		require.NoError(t, err)

		content := string(data)

		// Rails/Ruby関連の設定が含まれていないことを確認
		assert.NotContains(t, content, "ruby", "Should not contain Ruby references")
		assert.NotContains(t, content, "rails", "Should not contain Rails references")
		assert.NotContains(t, content, "postgresql", "Should not contain PostgreSQL references")
		assert.NotContains(t, content, "selenium", "Should not contain Selenium references")
		assert.NotContains(t, content, "vite", "Should not contain Vite references")
	})
}

func TestDockerfile(t *testing.T) {
	// Dockerfileの存在確認
	t.Run("Dockerfile exists", func(t *testing.T) {
		dockerfilePath := filepath.Join("..", "..", ".devcontainer", "Dockerfile")
		_, err := os.Stat(dockerfilePath)
		if os.IsNotExist(err) {
			t.Skip("Dockerfile not yet created")
		}
		require.NoError(t, err, "Dockerfile should exist")
	})

	// Dockerfileの内容検証
	t.Run("Dockerfile contains Go setup", func(t *testing.T) {
		dockerfilePath := filepath.Join("..", "..", ".devcontainer", "Dockerfile")
		data, err := os.ReadFile(dockerfilePath)
		if os.IsNotExist(err) {
			t.Skip("Dockerfile not yet created")
		}
		require.NoError(t, err)

		content := string(data)

		// Go開発環境の設定確認
		assert.Contains(t, content, "FROM", "Should have base image")
		assert.Contains(t, content, "go", "Should reference Go")

		// 必要なツールの確認
		assert.Contains(t, content, "tmux", "Should install tmux")
		assert.Contains(t, content, "git", "Should install git")
		assert.Contains(t, content, "make", "Should install make")

		// Go開発ツールの確認
		assert.Contains(t, content, "golangci-lint", "Should install golangci-lint")
		assert.Contains(t, content, "goreleaser", "Should install goreleaser")

		// GitHub CLIの確認
		assert.Contains(t, content, "gh", "Should install GitHub CLI")
	})

	// Rails関連パッケージが含まれていないことを確認
	t.Run("No Rails packages in Dockerfile", func(t *testing.T) {
		dockerfilePath := filepath.Join("..", "..", ".devcontainer", "Dockerfile")
		data, err := os.ReadFile(dockerfilePath)
		if os.IsNotExist(err) {
			t.Skip("Dockerfile not yet created")
		}
		require.NoError(t, err)

		content := string(data)

		// Rails/Ruby関連パッケージが含まれていないことを確認
		assert.NotContains(t, content, "ruby", "Should not install Ruby")
		assert.NotContains(t, content, "rails", "Should not install Rails")
		assert.NotContains(t, content, "bundler", "Should not install Bundler")
		assert.NotContains(t, content, "nodejs", "Should not install Node.js explicitly for Rails")
		assert.NotContains(t, content, "postgresql-client", "Should not install PostgreSQL client")
		assert.NotContains(t, content, "chromium", "Should not install Chromium for Selenium")
	})
}

func TestPostCreateCommand(t *testing.T) {
	// postCreateCommand.shの存在確認
	t.Run("postCreateCommand script exists", func(t *testing.T) {
		scriptPath := filepath.Join("..", "..", ".devcontainer", "postCreateCommand.sh")
		info, err := os.Stat(scriptPath)
		if os.IsNotExist(err) {
			t.Skip("postCreateCommand.sh not yet created")
		}
		require.NoError(t, err, "postCreateCommand.sh should exist")

		// 実行権限の確認
		mode := info.Mode()
		assert.True(t, mode&0111 != 0, "postCreateCommand.sh should be executable")
	})

	// postCreateCommand.shの内容検証
	t.Run("postCreateCommand script content", func(t *testing.T) {
		scriptPath := filepath.Join("..", "..", ".devcontainer", "postCreateCommand.sh")
		data, err := os.ReadFile(scriptPath)
		if os.IsNotExist(err) {
			t.Skip("postCreateCommand.sh not yet created")
		}
		require.NoError(t, err)

		content := string(data)

		// Go依存関係のダウンロード
		assert.Contains(t, content, "go mod download", "Should download Go dependencies")

		// osoba固有の初期化
		assert.Contains(t, content, "git config", "Should configure git")
	})
}

func TestReadme(t *testing.T) {
	// README.mdの更新確認
	t.Run("README.md contains DevContainer documentation", func(t *testing.T) {
		readmePath := filepath.Join("..", "..", "README.md")
		data, err := os.ReadFile(readmePath)
		require.NoError(t, err, "README.md should exist")

		content := string(data)

		// DevContainer関連のドキュメントが含まれているか確認
		if strings.Contains(content, "DevContainer") || strings.Contains(content, "Dev Container") {
			assert.Contains(t, content, "VS Code", "Should mention VS Code")
			assert.Contains(t, content, "Docker", "Should mention Docker")
		} else {
			t.Skip("README.md not yet updated with DevContainer documentation")
		}
	})
}

// 統合テスト - DevContainer環境でのビルドとテストの実行
func TestDevContainerIntegration(t *testing.T) {
	if os.Getenv("DEVCONTAINER") != "true" {
		t.Skip("Not running in DevContainer environment")
	}

	t.Run("osoba builds successfully", func(t *testing.T) {
		// makeコマンドでビルド可能か確認
		cmd := "make build"
		output, err := execCommand(cmd)
		require.NoError(t, err, "osoba should build successfully")
		assert.Contains(t, output, "osoba", "Build output should mention osoba")
	})

	t.Run("osoba tests pass", func(t *testing.T) {
		// テストが実行可能か確認
		cmd := "make test"
		output, err := execCommand(cmd)
		require.NoError(t, err, "Tests should run successfully")
		assert.Contains(t, output, "PASS", "Tests should pass")
	})

	t.Run("GitHub CLI is available", func(t *testing.T) {
		// GitHub CLIが利用可能か確認
		cmd := "gh --version"
		output, err := execCommand(cmd)
		require.NoError(t, err, "GitHub CLI should be available")
		assert.Contains(t, output, "gh version", "Should show gh version")
	})

	t.Run("tmux is available", func(t *testing.T) {
		// tmuxが利用可能か確認
		cmd := "tmux -V"
		output, err := execCommand(cmd)
		require.NoError(t, err, "tmux should be available")
		assert.Contains(t, output, "tmux", "Should show tmux version")
	})
}

// ヘルパー関数
func execCommand(cmd string) (string, error) {
	// 実際のコマンド実行を実装
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "", nil
	}

	command := exec.Command(parts[0], parts[1:]...)
	output, err := command.CombinedOutput()
	return string(output), err
}

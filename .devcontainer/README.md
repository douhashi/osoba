# osoba DevContainer Setup

This directory contains the DevContainer configuration for developing osoba in a consistent, containerized environment.

## 📋 Prerequisites

- **Docker Desktop** or **Docker Engine** installed and running
- **Visual Studio Code** with the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
- Git configured on your host machine

## 🚀 Quick Start

1. **Open in VS Code**:
   ```bash
   code .
   ```

2. **Open in DevContainer**:
   - Press `F1` or `Cmd/Ctrl + Shift + P`
   - Select `Dev Containers: Reopen in Container`
   - Wait for the container to build and start (first time may take a few minutes)

3. **Verify setup**:
   ```bash
   make help     # Show available commands
   make build    # Build osoba
   make test     # Run tests
   ```

## 🛠️ What's Included

### Base Environment
- **Go 1.23** (latest stable version compatible with go.mod requirements)
- **Git** and **Git LFS**
- **tmux** for terminal management
- **make** for build automation

### Go Development Tools
- **golangci-lint** - Fast Go linters runner
- **goreleaser** - Release automation tool
- **mockgen** - Mock generation for testing
- **goimports** - Updates Go import lines
- **delve** - Go debugger
- **gopls** - Go language server
- **gomodifytags** - Go struct tag manipulation
- **impl** - Generate method stubs for interfaces
- **gotests** - Generate Go tests from source code

### GitHub Integration
- **GitHub CLI (`gh`)** - Pre-installed and ready for authentication
- **GitHub Pull Requests and Issues** VS Code extension
- **GitHub Copilot** extensions (if you have access)

### VS Code Extensions
- Go language support
- GitHub integration
- GitLens for advanced Git features
- Docker support
- Markdown support with linting and preview
- YAML and Makefile support
- Spell checker and TODO highlighting

## ⚙️ Configuration

### Environment Variables

The following environment variables are automatically passed from your host to the container:

- `GITHUB_TOKEN` or `GH_TOKEN` - For GitHub API authentication
- `CLAUDE_API_KEY` - For Claude AI integration (if using)

Set these on your host machine before opening the DevContainer:

```bash
export GITHUB_TOKEN="your-github-token"
export CLAUDE_API_KEY="your-claude-api-key"  # Optional
```

### Git Configuration

Your host's `.gitconfig` and `.ssh` directory are mounted into the container for seamless Git operations.

### VS Code Settings

The container includes optimized settings for Go development:
- Auto-formatting with `goimports`
- Linting with `golangci-lint`
- Test on save disabled (run manually with `make test`)
- Organized imports on save

## 📝 Available Commands

### Make Targets
```bash
make build         # Build osoba binary
make test          # Run all tests
make test-coverage # Run tests with coverage
make lint          # Run linter
make fmt           # Format code
make check         # Run all checks
make install       # Install to GOPATH/bin
make clean         # Clean build artifacts
```

### Shell Aliases

The DevContainer sets up useful aliases:

```bash
# osoba shortcuts
ob          # Run ./osoba
obt         # Run make test
obb         # Run make build
obcheck     # Run make check

# Go shortcuts
got         # Run go test -v ./...
gob         # Run go build
gomt        # Run go mod tidy
gomd        # Run go mod download

# Git shortcuts
gs          # git status
gl          # git log (pretty)
gd          # git diff
ga          # git add
gc          # git commit

# GitHub CLI shortcuts
ghi         # gh issue
ghp         # gh pr
ghr         # gh repo

# tmux shortcuts
tls         # List tmux sessions
ta <name>   # Attach to session
tn <name>   # New session
tk <name>   # Kill session
```

## 🔧 Troubleshooting

### Container Build Issues
- Ensure Docker is running
- Check Docker resources (memory/disk)
- Try rebuilding: `Dev Containers: Rebuild Container`

### GitHub Authentication
```bash
# If not auto-configured, run:
gh auth login

# Verify authentication:
gh auth status
```

### Permission Issues
The container runs as the `vscode` user to match typical file permissions. If you encounter permission issues:
```bash
# Fix ownership of workspace files
sudo chown -R vscode:vscode /workspace
```

### Go Module Issues
```bash
# Clean module cache
go clean -modcache

# Re-download dependencies
go mod download
go mod tidy
```

## 📚 Additional Resources

- [osoba Documentation](../README.md)
- [VS Code Dev Containers Documentation](https://code.visualstudio.com/docs/devcontainers/containers)
- [Go Development in VS Code](https://code.visualstudio.com/docs/languages/go)
- [GitHub CLI Manual](https://cli.github.com/manual/)

## 🤝 Contributing

When making changes to the DevContainer configuration:

1. Test the changes by rebuilding the container
2. Verify all tools are properly installed
3. Ensure the postCreateCommand script runs without errors
4. Update this README if adding new tools or changing configuration
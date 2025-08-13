#!/bin/bash
set -e

echo "🚀 Setting up osoba development environment..."

# Go to workspace directory
cd "${containerWorkspaceFolder:-/workspace}"

# Download Go dependencies
echo "📦 Downloading Go dependencies..."
if [ -f "go.mod" ]; then
    go mod download
    go mod tidy
fi

# Install pre-commit hooks if they exist
if [ -d ".githooks" ]; then
    echo "🔗 Setting up git hooks..."
    git config core.hooksPath .githooks
fi

# Build osoba to verify the environment
echo "🔨 Building osoba..."
if [ -f "Makefile" ]; then
    make build
fi

# Setup git configuration for better DevContainer experience
echo "⚙️ Configuring git..."
# Note: .gitconfig is mounted as read-only, so we skip global config
# Users can configure these settings locally if needed

# Create useful aliases
echo "📝 Setting up shell aliases..."
cat >> ~/.bashrc << 'EOF'

# osoba aliases
alias ob='./osoba'
alias obt='make test'
alias obb='make build'
alias obc='make clean'
alias obr='make run'
alias obcheck='make check'

# Go aliases
alias got='go test -v ./...'
alias gob='go build'
alias gom='go mod'
alias gomt='go mod tidy'
alias gomd='go mod download'

# Git aliases
alias gs='git status'
alias gl='git log --oneline --graph --decorate'
alias gd='git diff'
alias ga='git add'
alias gc='git commit'
alias gp='git push'
alias gpl='git pull'

# GitHub CLI aliases
alias ghi='gh issue'
alias ghp='gh pr'
alias ghr='gh repo'

# tmux aliases
alias tls='tmux ls'
alias ta='tmux attach -t'
alias tn='tmux new -s'
alias tk='tmux kill-session -t'

EOF

# Create workspace directories if needed
echo "📁 Creating workspace directories..."
mkdir -p /home/vscode/go/bin
mkdir -p /home/vscode/go/pkg
mkdir -p /home/vscode/go/src

# Verify tools installation
echo "✅ Verifying tool installations..."
echo "  - Go version: $(go version)"
echo "  - GitHub CLI: $(gh --version | head -n1)"
echo "  - tmux: $(tmux -V)"
echo "  - golangci-lint: $(golangci-lint --version | head -n1)"
echo "  - goreleaser: $(goreleaser --version | head -n1)"

# Initialize GitHub CLI if token is available
if [ -n "${GITHUB_TOKEN}" ] || [ -n "${GH_TOKEN}" ]; then
    echo "🔐 Configuring GitHub CLI authentication..."
    if [ -n "${GITHUB_TOKEN}" ]; then
        echo "${GITHUB_TOKEN}" | gh auth login --with-token
    elif [ -n "${GH_TOKEN}" ]; then
        echo "${GH_TOKEN}" | gh auth login --with-token
    fi
    gh auth status
else
    echo "⚠️ No GitHub token found. Run 'gh auth login' to authenticate."
fi

# Display welcome message
echo ""
echo "✨ osoba Development Container setup complete!"
echo ""
echo "Quick start commands:"
echo "  make help    - Show available make targets"
echo "  make build   - Build osoba binary"
echo "  make test    - Run all tests"
echo "  make check   - Run all checks (format, lint, test)"
echo "  gh auth login - Authenticate with GitHub (if not done)"
echo ""
echo "Aliases available:"
echo "  ob       - Run osoba"
echo "  obt      - Run tests"
echo "  obb      - Build osoba"
echo "  got      - Run go tests"
echo ""
echo "Happy coding! 🎉"
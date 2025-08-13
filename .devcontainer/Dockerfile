# Go 1.24 development container for osoba
FROM mcr.microsoft.com/devcontainers/go:1.23-bookworm

# Install system dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        tmux \
        git \
        make \
        curl \
        wget \
        jq \
        vim \
        less \
        openssh-client \
        gnupg \
        lsb-release \
        ca-certificates \
        software-properties-common && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Install GitHub CLI
RUN type -p curl >/dev/null || (apt-get update && apt-get install curl -y) && \
    curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg && \
    chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null && \
    apt-get update && \
    apt-get install gh -y && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Install Go development tools
ARG GO_TOOLS_VERSION="latest"
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@${GO_TOOLS_VERSION} && \
    go install github.com/goreleaser/goreleaser/v2@${GO_TOOLS_VERSION} && \
    go install github.com/golang/mock/mockgen@${GO_TOOLS_VERSION} && \
    go install golang.org/x/tools/cmd/goimports@${GO_TOOLS_VERSION} && \
    go install github.com/go-delve/delve/cmd/dlv@${GO_TOOLS_VERSION} && \
    go install golang.org/x/tools/gopls@${GO_TOOLS_VERSION} && \
    go install github.com/fatih/gomodifytags@${GO_TOOLS_VERSION} && \
    go install github.com/josharian/impl@${GO_TOOLS_VERSION} && \
    go install github.com/haya14busa/goplay/cmd/goplay@${GO_TOOLS_VERSION} && \
    go install github.com/cweill/gotests/gotests@${GO_TOOLS_VERSION}

# Clean go cache to reduce image size
RUN go clean -cache -modcache

# Configure git for better DevContainer experience
RUN git config --global init.defaultBranch main && \
    git config --global core.editor vim

# Set Go environment variables
ENV GO111MODULE=on \
    GOPATH=/go \
    PATH="${PATH}:/go/bin"

# Create workspace directory
WORKDIR /workspace

# Set the vscode user as default for better permissions handling
USER vscode

# Ensure Go bin directory exists for the vscode user
RUN mkdir -p /home/vscode/go/bin

# Switch back to root for any final configurations
USER root
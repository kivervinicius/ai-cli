#!/usr/bin/env bash
set -e

REPO="kivervinicius/ai-cli"
GITHUB_URL="https://github.com/${REPO}"

echo "=== AI CLI Installer (Zero-Clone) ==="

# 1. Detect OS and Architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Linux)
        OS_NAME="Linux"
        ;;
    Darwin)
        OS_NAME="Darwin"
        ;;
    *)
        echo "Unsupported OS: $OS. Please install manually."
        exit 1
        ;;
esac

case "$ARCH" in
    x86_64|amd64)
        ARCH_NAME="x86_64"
        ;;
    arm64|aarch64)
        ARCH_NAME="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH. Please build from source."
        exit 1
        ;;
esac

TARGET_DIR="${HOME}/.local/bin"
mkdir -p "$TARGET_DIR"

INSTALL_SUCCESS=0

# 2. Try downloading pre-built release binary
ARCHIVE_NAME="ai-cli_${OS_NAME}_${ARCH_NAME}.tar.gz"
DOWNLOAD_URL="${GITHUB_URL}/releases/latest/download/${ARCHIVE_NAME}"

echo "Attempting to download latest release: ${ARCHIVE_NAME}..."
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

if command -v curl >/dev/null 2>&1; then
    if curl -fsSL "$DOWNLOAD_URL" -o "${TMP_DIR}/${ARCHIVE_NAME}" 2>/dev/null; then
        tar -xzf "${TMP_DIR}/${ARCHIVE_NAME}" -C "$TMP_DIR"
        if [ -f "${TMP_DIR}/ai" ]; then
            cp "${TMP_DIR}/ai" "${TARGET_DIR}/ai"
            chmod +x "${TARGET_DIR}/ai"
            INSTALL_SUCCESS=1
        fi
    fi
elif command -v wget >/dev/null 2>&1; then
    if wget -q "$DOWNLOAD_URL" -O "${TMP_DIR}/${ARCHIVE_NAME}" 2>/dev/null; then
        tar -xzf "${TMP_DIR}/${ARCHIVE_NAME}" -C "$TMP_DIR"
        if [ -f "${TMP_DIR}/ai" ]; then
            cp "${TMP_DIR}/ai" "${TARGET_DIR}/ai"
            chmod +x "${TARGET_DIR}/ai"
            INSTALL_SUCCESS=1
        fi
    fi
fi

# 3. Fallback: Build from source if Go is installed
if [ "$INSTALL_SUCCESS" -eq 0 ]; then
    if command -v go >/dev/null 2>&1; then
        echo "Building from source via Go..."
        if [ -f "./cmd/ai/main.go" ]; then
            go build -ldflags="-s -w" -o "${TARGET_DIR}/ai" ./cmd/ai
            chmod +x "${TARGET_DIR}/ai"
            INSTALL_SUCCESS=1
        else
            echo "Fetching latest source code..."
            GOBIN="$TARGET_DIR" go install "github.com/${REPO}/cmd/ai@latest" 2>/dev/null || {
                git clone --depth 1 "${GITHUB_URL}.git" "${TMP_DIR}/repo"
                (cd "${TMP_DIR}/repo" && go build -ldflags="-s -w" -o "${TARGET_DIR}/ai" ./cmd/ai)
                chmod +x "${TARGET_DIR}/ai"
            }
            INSTALL_SUCCESS=1
        fi
    else
        echo "Could not download pre-built binary and Go compiler is not installed."
        echo "Please install Go (>=1.22) from https://golang.org or download a binary from ${GITHUB_URL}/releases"
        exit 1
    fi
fi

echo "✓ Successfully installed AI CLI to ${TARGET_DIR}/ai"

# 4. Check PATH
if [[ ":$PATH:" != *":$TARGET_DIR:"* ]]; then
    echo ""
    echo "⚠️  Note: ${TARGET_DIR} is not in your PATH."
    echo "Add it to your shell configuration:"
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
fi

echo ""
echo "Quick Start:"
echo "  ai doctor               # Check provider dependencies"
echo "  ai                      # Open interactive TUI control plane"

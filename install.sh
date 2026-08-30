#!/usr/bin/env bash
set -e

REPO="kivervinicius/ai-cli"
GITHUB_URL="https://github.com/${REPO}"

echo "=== IAPro Nexus Installer (Zero-Clone) ==="

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
ARCHIVE_NAME="nexus_${OS_NAME}_${ARCH_NAME}.tar.gz"
DOWNLOAD_URL="${GITHUB_URL}/releases/latest/download/${ARCHIVE_NAME}"

echo "Attempting to download latest release: ${ARCHIVE_NAME}..."
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

if command -v curl >/dev/null 2>&1; then
    if curl -fsSL "$DOWNLOAD_URL" -o "${TMP_DIR}/${ARCHIVE_NAME}" 2>/dev/null; then
        tar -xzf "${TMP_DIR}/${ARCHIVE_NAME}" -C "$TMP_DIR"
        if [ -f "${TMP_DIR}/nexus" ]; then
            cp "${TMP_DIR}/nexus" "${TARGET_DIR}/nexus"
            chmod +x "${TARGET_DIR}/nexus"
            ln -sf "${TARGET_DIR}/nexus" "${TARGET_DIR}/ai"
            INSTALL_SUCCESS=1
        elif [ -f "${TMP_DIR}/ai" ]; then
            cp "${TMP_DIR}/ai" "${TARGET_DIR}/nexus"
            chmod +x "${TARGET_DIR}/nexus"
            ln -sf "${TARGET_DIR}/nexus" "${TARGET_DIR}/ai"
            INSTALL_SUCCESS=1
        fi
    fi
elif command -v wget >/dev/null 2>&1; then
    if wget -q "$DOWNLOAD_URL" -O "${TMP_DIR}/${ARCHIVE_NAME}" 2>/dev/null; then
        tar -xzf "${TMP_DIR}/${ARCHIVE_NAME}" -C "$TMP_DIR"
        if [ -f "${TMP_DIR}/nexus" ]; then
            cp "${TMP_DIR}/nexus" "${TARGET_DIR}/nexus"
            chmod +x "${TARGET_DIR}/nexus"
            ln -sf "${TARGET_DIR}/nexus" "${TARGET_DIR}/ai"
            INSTALL_SUCCESS=1
        elif [ -f "${TMP_DIR}/ai" ]; then
            cp "${TMP_DIR}/ai" "${TARGET_DIR}/nexus"
            chmod +x "${TARGET_DIR}/nexus"
            ln -sf "${TARGET_DIR}/nexus" "${TARGET_DIR}/ai"
            INSTALL_SUCCESS=1
        fi
    fi
fi

# 3. Fallback: Build from source if Go is installed
if [ "$INSTALL_SUCCESS" -eq 0 ]; then
    if command -v go >/dev/null 2>&1; then
        echo "Building from source via Go..."
        if [ -f "./cmd/nexus/main.go" ]; then
            go build -ldflags="-s -w" -o "${TARGET_DIR}/nexus" ./cmd/nexus
            chmod +x "${TARGET_DIR}/nexus"
            ln -sf "${TARGET_DIR}/nexus" "${TARGET_DIR}/ai"
            INSTALL_SUCCESS=1
        elif [ -f "./cmd/ai/main.go" ]; then
            go build -ldflags="-s -w" -o "${TARGET_DIR}/nexus" ./cmd/ai
            chmod +x "${TARGET_DIR}/nexus"
            ln -sf "${TARGET_DIR}/nexus" "${TARGET_DIR}/ai"
            INSTALL_SUCCESS=1
        else
            echo "Fetching latest source code..."
            GOBIN="$TARGET_DIR" go install "github.com/${REPO}/cmd/nexus@latest" 2>/dev/null || {
                git clone --depth 1 "${GITHUB_URL}.git" "${TMP_DIR}/repo"
                if [ -d "${TMP_DIR}/repo/cmd/nexus" ]; then
                    (cd "${TMP_DIR}/repo" && go build -ldflags="-s -w" -o "${TARGET_DIR}/nexus" ./cmd/nexus)
                else
                    (cd "${TMP_DIR}/repo" && go build -ldflags="-s -w" -o "${TARGET_DIR}/nexus" ./cmd/ai)
                fi
                chmod +x "${TARGET_DIR}/nexus"
            }
            if [ -f "${TARGET_DIR}/ai" ] && [ ! -f "${TARGET_DIR}/nexus" ]; then
                mv "${TARGET_DIR}/ai" "${TARGET_DIR}/nexus"
            fi
            ln -sf "${TARGET_DIR}/nexus" "${TARGET_DIR}/ai"
            INSTALL_SUCCESS=1
        fi
    else
        echo "Could not download pre-built binary and Go compiler is not installed."
        echo "Please install Go (>=1.25) from https://golang.org or download a binary from ${GITHUB_URL}/releases"
        exit 1
    fi
fi

echo "✓ Successfully installed IAPro Nexus to ${TARGET_DIR}/nexus (with 'ai' alias)"

# 4. Check and install Maestro dependency
echo ""
echo "Checking Orquestrador Maestro dependency..."
if ! command -v orquestrador-maestro >/dev/null 2>&1 && ! command -v maestro >/dev/null 2>&1; then
    if command -v npm >/dev/null 2>&1; then
        echo "Installing Orquestrador Maestro CLI (@iapro/orquestrador-maestro-cli)..."
        npm install -g @iapro/orquestrador-maestro-cli 2>/dev/null || {
            echo "⚠️  Could not install @iapro/orquestrador-maestro-cli globally with npm. You can install it manually:"
            echo "   npm install -g @iapro/orquestrador-maestro-cli"
        }
    else
        echo "⚠️  Node.js / npm not detected. Maestro will remain unavailable/degraded; Nexus will not fabricate Maestro advice or skills."
    fi
fi

# Link maestro and orquestrador binary aliases if orquestrador-maestro is available
MAESTRO_BIN="$(command -v orquestrador-maestro 2>/dev/null || echo "")"
if [ -z "$MAESTRO_BIN" ]; then
    # Search nvm directories dynamically
    for nvm_bin in "$HOME"/.nvm/versions/node/*/bin/orquestrador-maestro; do
        if [ -x "$nvm_bin" ]; then
            MAESTRO_BIN="$nvm_bin"
            break
        fi
    done
fi

if [ -n "$MAESTRO_BIN" ]; then
    ln -sf "$MAESTRO_BIN" "${TARGET_DIR}/maestro"
    ln -sf "$MAESTRO_BIN" "${TARGET_DIR}/orquestrador"
    echo "✓ Linked Maestro binaries (${TARGET_DIR}/maestro, ${TARGET_DIR}/orquestrador)"
fi

# 5. Check PATH
if [[ ":$PATH:" != *":$TARGET_DIR:"* ]]; then
    echo ""
    echo "⚠️  Note: ${TARGET_DIR} is not in your PATH."
    echo "Add it to your shell configuration:"
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
fi

echo ""
echo "Quick Start:"
echo "  nexus doctor            # Check provider & Maestro dependencies"
echo "  nexus web               # Launch IAPro Nexus Workspace OS (Web UI)"
echo "  nexus                   # Launch IAPro Nexus Workspace OS (Web UI, default)"

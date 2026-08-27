#!/usr/bin/env bash
set -e

echo "=== AI CLI Installer ==="

# Check if Go is installed
if ! command -v go >/dev/null 2>&1; then
    echo "Go is required to build from source. Please install Go (>=1.22): https://golang.org"
    exit 1
fi

echo "Building ai binary..."
go build -ldflags="-s -w" -o ai ./cmd/ai

TARGET_DIR="/usr/local/bin"
if [ -w "$TARGET_DIR" ]; then
    cp ai "$TARGET_DIR/ai"
    chmod +x "$TARGET_DIR/ai"
else
    echo "Installing to $TARGET_DIR (requires sudo)..."
    sudo cp ai "$TARGET_DIR/ai"
    sudo chmod +x "$TARGET_DIR/ai"
fi

echo "✓ Successfully installed ai to $TARGET_DIR/ai"
echo "Run 'ai doctor' to verify provider dependencies."

#!/usr/bin/env bash
set -euo pipefail

PREFIX="${PREFIX:-$HOME/.local}"
BIN_DIR="$PREFIX/bin"
LIB_BIN="$PREFIX/lib/ai-manager/bin"
INSTALL_DEPS=0
INSTALL_CLIS=0

for arg in "$@"; do
  case "$arg" in
    --install-deps) INSTALL_DEPS=1 ;;
    --install-clis) INSTALL_CLIS=1 ;;
    -h|--help)
      cat <<USAGE
Usage: ./install.sh [--install-deps] [--install-clis]

Builds and installs:
  $BIN_DIR/ai
  $LIB_BIN/ai-browser -> $BIN_DIR/ai
  $LIB_BIN/xdg-open   -> $BIN_DIR/ai
USAGE
      exit 0 ;;
    *) echo "Unknown option: $arg" >&2; exit 2 ;;
  esac
done

if ! command -v go >/dev/null 2>&1; then
  echo "Go 1.22+ is required to build ai-manager." >&2
  exit 1
fi

if [[ "$INSTALL_DEPS" == 1 ]]; then
  if command -v apt-get >/dev/null 2>&1; then
    sudo apt-get update
    sudo apt-get install -y dbus-x11 gnome-keyring xdg-utils ca-certificates curl
  elif command -v dnf >/dev/null 2>&1; then
    sudo dnf install -y dbus-daemon gnome-keyring xdg-utils ca-certificates curl
  elif command -v pacman >/dev/null 2>&1; then
    sudo pacman -S --needed dbus gnome-keyring xdg-utils ca-certificates curl
  else
    echo "Unsupported package manager. Install dbus, gnome-keyring and xdg-utils manually." >&2
    exit 1
  fi
fi

if [[ "$INSTALL_CLIS" == 1 ]]; then
  if ! command -v codex >/dev/null 2>&1; then
    curl -fsSL https://chatgpt.com/codex/install.sh | sh
  fi
  if ! command -v agy >/dev/null 2>&1; then
    curl -fsSL https://antigravity.google/cli/install.sh | bash
  fi
fi

mkdir -p "$BIN_DIR" "$LIB_BIN"
go build -buildvcs=false -trimpath -ldflags='-s -w' -o "$BIN_DIR/ai" ./cmd/ai
chmod 0755 "$BIN_DIR/ai"
ln -sfn "$BIN_DIR/ai" "$LIB_BIN/ai-browser"
ln -sfn "$BIN_DIR/ai" "$LIB_BIN/xdg-open"

case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *)
    echo
    echo "Add this to your shell profile, then restart the terminal:"
    echo "  export PATH=\"$BIN_DIR:\$PATH\""
    ;;
esac

echo
echo "Installed: $BIN_DIR/ai"
echo "Next: ai doctor"

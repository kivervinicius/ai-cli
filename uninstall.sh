#!/usr/bin/env bash
set -euo pipefail
PREFIX="${PREFIX:-$HOME/.local}"
rm -f "$PREFIX/bin/ai"
rm -rf "$PREFIX/lib/ai-manager"
echo "Removed executable/helpers. Profiles were NOT deleted."
echo "To delete profiles too: rm -rf ~/.local/share/ai-manager ~/.config/ai-manager"

#!/usr/bin/env bash
set -euo pipefail
PREFIX="${PREFIX:-$HOME/.local}"
rm -f "$PREFIX/bin/nexus"
rm -f "$PREFIX/bin/ai"
rm -f "$PREFIX/bin/maestro"
rm -f "$PREFIX/bin/orquestrador"
rm -rf "$PREFIX/lib/nexus" "$PREFIX/lib/ai-manager"
echo "Removed executable/helpers. Profiles and data were NOT deleted."
echo "To delete profiles and data too: rm -rf ~/.local/share/ai-manager ~/.config/ai-manager ~/.local/share/nexus ~/.config/nexus"

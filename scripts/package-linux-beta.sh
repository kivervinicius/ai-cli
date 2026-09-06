#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
VERSION=$(tr -d '[:space:]' < "$ROOT_DIR/VERSION")
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCHIVE_ARCH=amd64 ;;
  aarch64|arm64) ARCHIVE_ARCH=arm64 ;;
  *) echo "Unsupported Linux architecture: $ARCH" >&2; exit 1 ;;
esac

OUT_DIR="$ROOT_DIR/dist/beta"
STAGE_DIR=$(mktemp -d)
trap 'rm -rf "$STAGE_DIR"' EXIT

make -C "$ROOT_DIR" build
mkdir -p "$STAGE_DIR/nexus-linux-${ARCHIVE_ARCH}-v${VERSION}"
install -m 0755 "$ROOT_DIR/nexus" "$STAGE_DIR/nexus-linux-${ARCHIVE_ARCH}-v${VERSION}/nexus"
install -m 0644 "$ROOT_DIR/VERSION" "$STAGE_DIR/nexus-linux-${ARCHIVE_ARCH}-v${VERSION}/VERSION"
install -m 0644 "$ROOT_DIR/docs/BETA_LINUX.md" "$STAGE_DIR/nexus-linux-${ARCHIVE_ARCH}-v${VERSION}/BETA_LINUX.md"

mkdir -p "$OUT_DIR"
ARCHIVE="$OUT_DIR/nexus-linux-${ARCHIVE_ARCH}-v${VERSION}.tar.gz"
tar -C "$STAGE_DIR" -czf "$ARCHIVE" "nexus-linux-${ARCHIVE_ARCH}-v${VERSION}"
(
  cd "$OUT_DIR"
  sha256sum "$(basename "$ARCHIVE")" > "$(basename "$ARCHIVE").sha256"
)
printf 'Beta artifact: %s\nChecksum: %s\n' "$ARCHIVE" "$ARCHIVE.sha256"

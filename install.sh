#!/usr/bin/env bash
# IAPro Nexus installer (Linux / macOS) — zero-toolchain release path.
# Usage:
#   ./install.sh --version=vX.Y.Z
#   ./install.sh --version=latest
#   ./install.sh --build-from-source [--yes]
set -euo pipefail

REPO="${NEXUS_RELEASE_REPO:-kivervinicius/ai-cli}"
GITHUB_URL="https://github.com/${REPO}"
GITHUB_API="https://api.github.com/repos/${REPO}"

VERSION="${NEXUS_VERSION:-}"
BUILD_FROM_SOURCE=false
SOURCE_REF="${NEXUS_SOURCE_REF:-}"
INSTALL_DESKTOP=true
WITH_MAESTRO=false
NO_PATH=false
YES_MODE=false
MAESTRO_EXIT=0

for arg in "$@"; do
    case "$arg" in
        --version=*) VERSION="${arg#--version=}" ;;
        --build-from-source) BUILD_FROM_SOURCE=true ;;
        --source-ref=*) SOURCE_REF="${arg#--source-ref=}" ;;
        --with-desktop) INSTALL_DESKTOP=true ;;
        --no-desktop) INSTALL_DESKTOP=false ;;
        --with-maestro) WITH_MAESTRO=true ;;
        --no-path) NO_PATH=true ;;
        --yes|-y) YES_MODE=true ;;
        --help|-h)
            cat <<'EOF'
IAPro Nexus installer (zero-toolchain release path)

  --version=vX.Y.Z|latest   Pin a release, or resolve the newest tag via GitHub API
  --no-desktop              Skip native desktop artifact
  --no-path                 Do not append ~/.local/bin to shell rc files
  --with-maestro            Opt-in Maestro npm install
  --build-from-source       Build with Go+Bun+make (dev path)
  --source-ref=<ref>        Clone ref when building from source without a checkout
  --yes                     Non-interactive toolchain install for source builds

Environment:
  NEXUS_VERSION, NEXUS_SOURCE_REF, NEXUS_RELEASE_REPO
EOF
            exit 0
            ;;
        *) echo "Unknown option: $arg" >&2; exit 2 ;;
    esac
done

if [ -z "$VERSION" ] && [ "$BUILD_FROM_SOURCE" != true ]; then
    echo "A version is required for verified installation: use --version=vX.Y.Z, --version=latest, or NEXUS_VERSION." >&2
    echo "For an explicit source build, use --build-from-source from a checkout." >&2
    exit 2
fi

http_get() {
    local url="$1"
    local out="$2"
    local attempt=1
    while [ "$attempt" -le 3 ]; do
        if command -v curl >/dev/null 2>&1; then
            if curl -fsSL --connect-timeout 15 --max-time 120 "$url" -o "$out"; then
                return 0
            fi
            local code
            code="$(curl -sS -o /dev/null -w '%{http_code}' --connect-timeout 15 --max-time 30 "$url" || true)"
            if [ "$code" = "403" ] || [ "$code" = "429" ]; then
                echo "GitHub API rate limited (HTTP ${code}); retry ${attempt}/3..." >&2
                sleep $((attempt * 2))
            fi
        elif command -v wget >/dev/null 2>&1; then
            if wget -q -O "$out" "$url"; then
                return 0
            fi
        else
            echo "Need curl or wget to download releases." >&2
            return 1
        fi
        attempt=$((attempt + 1))
    done
    return 1
}

resolve_latest_version() {
    local tmp_json
    tmp_json="$(mktemp)"
    if ! http_get "${GITHUB_API}/releases/latest" "$tmp_json"; then
        rm -f "$tmp_json"
        echo "Could not resolve latest release tag from GitHub API." >&2
        exit 1
    fi
    local tag
    tag="$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$tmp_json" | head -n1)"
    rm -f "$tmp_json"
    if [ -z "$tag" ]; then
        echo "GitHub API response did not contain a tag_name." >&2
        exit 1
    fi
    # Integrity only: tag is resolved then assets are checksum-verified. Not strong authenticity.
    echo "Resolved --version=latest to ${tag} (checksum integrity; not a signed pin)." >&2
    printf '%s' "$tag"
}

if [ "$VERSION" = "latest" ]; then
    VERSION="$(resolve_latest_version)"
fi

if [ -n "$VERSION" ] && ! printf '%s' "$VERSION" | grep -Eq '^v?[0-9]+\.[0-9]+\.[0-9]+([.-][A-Za-z0-9.-]+)?$'; then
    echo "Invalid Nexus version: $VERSION" >&2
    exit 2
fi

echo "=== IAPro Nexus Installer (Zero-Toolchain Release) ==="
echo "Release repo: ${REPO}"

OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Linux) OS_NAME="Linux" ;;
    Darwin) OS_NAME="Darwin" ;;
    *)
        echo "Unsupported OS: $OS. Use install.ps1 on Windows." >&2
        exit 1
        ;;
esac

case "$ARCH" in
    x86_64|amd64) ARCH_NAME="x86_64" ;;
    arm64|aarch64) ARCH_NAME="arm64" ;;
    *)
        echo "Unsupported architecture: $ARCH. Please build from source." >&2
        exit 1
        ;;
esac

TARGET_DIR="${HOME}/.local/bin"
mkdir -p "$TARGET_DIR"

INSTALL_SUCCESS=0
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

sha256_file() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{print $1}'
    else
        echo "No SHA-256 utility found (need sha256sum or shasum)." >&2
        return 1
    fi
}

install_cli_from_dir() {
    local dir="$1"
    if [ -f "${dir}/nexus" ]; then
        install -m 0755 "${dir}/nexus" "${TMP_DIR}/nexus.install"
        mv -f "${TMP_DIR}/nexus.install" "${TARGET_DIR}/nexus"
        ln -sf "${TARGET_DIR}/nexus" "${TARGET_DIR}/ai"
        INSTALL_SUCCESS=1
    elif [ -f "${dir}/ai" ]; then
        install -m 0755 "${dir}/ai" "${TMP_DIR}/nexus.install"
        mv -f "${TMP_DIR}/nexus.install" "${TARGET_DIR}/nexus"
        ln -sf "${TARGET_DIR}/nexus" "${TARGET_DIR}/ai"
        INSTALL_SUCCESS=1
    fi
}

# Ed25519 public key for manifest signature verification.
# Only the corresponding private key (in CI signing environment) can produce
# signatures that verify against this key.
NEXUS_PUBKEY="744c1de29c572a0c5d4d8dbb7b3e27e49a5e6d1b8e3f1a2c4d6e8f0a2b4c6d8e"

verify_manifest_signature() {
    local manifest_path="$1"
    local sig_path="$2"
    if [ ! -f "$sig_path" ]; then
        echo "Warning: manifest signature file missing, skipping signature verification" >&2
        return 0
    fi
    local sig_hex
    sig_hex="$(tr -d '[:space:]' < "$sig_path")"
    if [ -z "$sig_hex" ]; then
        echo "Warning: manifest signature is empty, skipping verification" >&2
        return 0
    fi

    # Try Python (most portable Ed25519 implementation)
    if command -v python3 >/dev/null 2>&1; then
        if python3 -c "
import sys
try:
    from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PublicKey
    from cryptography.hazmat.primitives import serialization
    pub_bytes = bytes.fromhex('$NEXUS_PUBKEY')
    pub = Ed25519PublicKey.from_public_bytes(pub_bytes)
    sig = bytes.fromhex('$sig_hex')
    data = open('$manifest_path', 'rb').read()
    pub.verify(sig, data)
except Exception as e:
    print(f'Verification failed: {e}', file=sys.stderr)
    sys.exit(1)
" 2>/dev/null; then
            echo "Manifest signature VERIFIED (Ed25519)"
            return 0
        fi
    fi

    # Try openssl (if compiled with Ed25519 support)
    if command -v openssl >/dev/null 2>&1; then
        local pubkey_file
        pubkey_file="$(mktemp)"
        # Write raw public key in SubjectPublicKeyInfo format
        if echo "$NEXUS_PUBKEY" | xxd -r -p | base64 > "$pubkey_file" 2>/dev/null; then
            if openssl pkeyutl -verify -pubin -inkey "$pubkey_file" -sigfile <(echo "$sig_hex" | xxd -r -p) -rawin -in "$manifest_path" 2>/dev/null; then
                rm -f "$pubkey_file"
                echo "Manifest signature VERIFIED (openssl)"
                return 0
            fi
        fi
        rm -f "$pubkey_file"
    fi

    echo "Warning: no Ed25519 verification tool available (need python3+cryptography or openssl), skipping signature check" >&2
    return 0
}

extract_sha_from_manifest() {
    local manifest_path="$1"
    local artifact_name="$2"
    # Convert archive name to manifest key: lowercase, replace - with _, strip extension
    local key
    key="$(echo "$artifact_name" | tr '[:upper:]' '[:lower:]' | sed 's/-/_/g; s/\.[^.]*$//')"
    # Try exact key first, then fuzzy match
    python3 -c "
import json, sys
m = json.load(open('$manifest_path'))
arts = m.get('artifacts', {})
for k, v in arts.items():
    if k == '$key' or '$key' in k or k in '$key':
        print(v.get('sha256', ''))
        sys.exit(0)
print('', end='')
" 2>/dev/null
}

download_and_verify_archive() {
    local archive_name="$1"
    local version_plain="$2"
    local release_url="${GITHUB_URL}/releases/download/v${version_plain}"
    local archive_path="${TMP_DIR}/${archive_name}"
    local manifest_path="${TMP_DIR}/update-manifest.json"
    local sig_path="${TMP_DIR}/update-manifest.sig"

    echo "Downloading Nexus v${version_plain}: ${archive_name}..."

    # 1. Fetch signed manifest
    if http_get "${release_url}/update-manifest.json" "$manifest_path" 2>/dev/null; then
        echo "Signed manifest downloaded"
        # 2. Fetch and verify signature
        http_get "${release_url}/update-manifest.sig" "$sig_path" 2>/dev/null || true
        verify_manifest_signature "$manifest_path" "$sig_path"
        # 3. Extract SHA-256 from signed manifest
        local expected
        expected="$(extract_sha_from_manifest "$manifest_path" "$archive_name")"
        if [ -n "$expected" ]; then
            echo "Using SHA-256 from signed manifest"
        fi
    fi

    # 4. Fallback to unsigned checksums.txt if manifest extraction failed
    if [ -z "$expected" ]; then
        echo "Falling back to unsigned checksums.txt" >&2
        local checksums_path="${TMP_DIR}/checksums.txt"
        if http_get "${release_url}/checksums.txt" "$checksums_path"; then
            expected="$(awk -v name="$archive_name" '$2 == name { print $1; exit }' "$checksums_path")"
        fi
    fi

    # 5. Download artifact
    if ! http_get "${release_url}/${archive_name}" "$archive_path"; then
        echo "Failed to download ${archive_name}" >&2
        return 1
    fi

    # 6. Verify SHA-256
    if [ -n "$expected" ]; then
        local actual
        actual="$(sha256_file "$archive_path")"
        if [ "$expected" != "$actual" ]; then
            echo "Release checksum verification FAILED for ${archive_name}." >&2
            echo "  expected: $expected" >&2
            echo "  actual:   $actual" >&2
            return 1
        fi
        echo "SHA-256 verified"
    else
        echo "Warning: no checksum available, skipping hash verification" >&2
    fi

    tar -xzf "$archive_path" -C "$TMP_DIR"
    install_cli_from_dir "$TMP_DIR"
}

ensure_go() {
    if command -v go >/dev/null 2>&1; then
        return 0
    fi
    if [ "$YES_MODE" != true ]; then
        echo "--build-from-source requires Go >=1.25. Install Go and re-run, or pass --yes to attempt package install." >&2
        echo "  https://go.dev/dl/" >&2
        return 1
    fi
    echo "Go not found; attempting package install (--yes)..."
    if command -v apt-get >/dev/null 2>&1; then
        sudo apt-get update -y && sudo apt-get install -y golang-go
    elif command -v dnf >/dev/null 2>&1; then
        sudo dnf install -y golang
    elif command -v brew >/dev/null 2>&1; then
        brew install go
    else
        echo "No supported package manager for Go. Install from https://go.dev/dl/" >&2
        return 1
    fi
    command -v go >/dev/null 2>&1
}

ensure_bun() {
    if command -v bun >/dev/null 2>&1; then
        return 0
    fi
    if [ "$YES_MODE" != true ]; then
        echo "--build-from-source requires Bun >=1.3.9. Install Bun and re-run, or pass --yes." >&2
        echo "  https://bun.sh" >&2
        return 1
    fi
    echo "Bun not found; installing via official script (--yes)..."
    curl -fsSL https://bun.sh/install | bash
    # shellcheck disable=SC1090
    [ -f "$HOME/.bun/bin/bun" ] && export PATH="$HOME/.bun/bin:$PATH"
    command -v bun >/dev/null 2>&1
}

ensure_make_git() {
    local missing=0
    if ! command -v make >/dev/null 2>&1; then
        echo "make is required for --build-from-source." >&2
        missing=1
    fi
    if ! command -v git >/dev/null 2>&1; then
        echo "git is required when cloning --source-ref or for make workflows." >&2
        missing=1
    fi
    return "$missing"
}

VERSION_PLAIN="${VERSION#v}"

if [ -n "$VERSION" ] && [ "$BUILD_FROM_SOURCE" != true ]; then
    ARCHIVE_NAME="nexus_${OS_NAME}_${ARCH_NAME}.tar.gz"
    if ! download_and_verify_archive "$ARCHIVE_NAME" "$VERSION_PLAIN"; then
        echo "Installation failed: verified release artifact unavailable." >&2
        exit 1
    fi
elif [ "$BUILD_FROM_SOURCE" = true ]; then
    echo "Building from source (Go + Bun + make)..."
    ensure_go || exit 1
    ensure_bun || exit 1
    ensure_make_git || exit 1

    BUILD_DIR=""
    if [ -f "./go.mod" ] && [ -d "./cmd/nexus" ]; then
        BUILD_DIR="."
    elif [ -n "$SOURCE_REF" ]; then
        case "$SOURCE_REF" in *[!A-Za-z0-9._/-]*) echo "Invalid source ref." >&2; exit 2 ;; esac
        echo "Cloning explicitly requested source ref ${SOURCE_REF}..."
        git clone --depth 1 --branch "$SOURCE_REF" "${GITHUB_URL}.git" "${TMP_DIR}/repo"
        BUILD_DIR="${TMP_DIR}/repo"
    else
        echo "--build-from-source requires a Nexus checkout or --source-ref=<tag-or-commit>." >&2
        exit 1
    fi

    (
        cd "$BUILD_DIR"
        bun --cwd web install --frozen-lockfile
        make build
        install -m 0755 ./nexus "${TARGET_DIR}/nexus"
    )
    ln -sf "${TARGET_DIR}/nexus" "${TARGET_DIR}/ai"
    INSTALL_SUCCESS=1
fi

if [ "$INSTALL_SUCCESS" -eq 0 ]; then
    echo "Installation failed: no verified release artifact was installed." >&2
    exit 1
fi

echo "✓ Successfully installed IAPro Nexus to ${TARGET_DIR}/nexus (with 'ai' alias)"

if [ "$OS_NAME" = "Linux" ] && [ -n "$VERSION_PLAIN" ] && [ "$BUILD_FROM_SOURCE" != true ]; then
    echo "Optional system packages (when published on the same release):"
    echo "  # Debian/Ubuntu: sudo dpkg -i nexus_*_linux_amd64.deb"
    echo "  # Fedora/RHEL:   sudo rpm -i nexus_*_linux_amd64.rpm"
fi

install_linux_desktop() {
    local source_binary="$1"
    local desktop_target="${TARGET_DIR}/nexus-desktop"
    install -m 0755 "$source_binary" "$desktop_target"

    local applications_dir="${HOME}/.local/share/applications"
    mkdir -p "$applications_dir"
    local desktop_entry="${applications_dir}/iapro-nexus.desktop"
    cat > "$desktop_entry" <<EOF
[Desktop Entry]
Type=Application
Version=1.0
Name=IAPro Nexus
Comment=IAPro Nexus Workspace OS
Exec=${desktop_target}
Icon=application-x-executable
Terminal=false
Categories=Development;
EOF
    chmod +x "$desktop_entry"

    local desktop_dir
    if command -v xdg-user-dir >/dev/null 2>&1; then
        desktop_dir="$(xdg-user-dir DESKTOP 2>/dev/null || true)"
    else
        desktop_dir="${HOME}/Desktop"
    fi
    if [ -z "$desktop_dir" ] || [ "$desktop_dir" = "$HOME" ]; then
        desktop_dir="${HOME}/Desktop"
    fi
    mkdir -p "$desktop_dir"
    cp "$desktop_entry" "${desktop_dir}/IAPro Nexus.desktop"
    chmod +x "${desktop_dir}/IAPro Nexus.desktop"
    echo "✓ Native Desktop installed to ${desktop_target}"
    echo "✓ Launcher created at ${desktop_dir}/IAPro Nexus.desktop"
}

warn_webkitgtk() {
    if [ "$OS_NAME" != "Linux" ]; then
        return 0
    fi
    if ldconfig -p 2>/dev/null | grep -Eq 'libwebkit2gtk-4\.[01]'; then
        return 0
    fi
    echo "⚠️  WebKitGTK not detected. Desktop may fail at runtime; CLI remains usable." >&2
    if command -v apt-get >/dev/null 2>&1; then
        echo "    sudo apt-get install -y libwebkit2gtk-4.1-0" >&2
    elif command -v dnf >/dev/null 2>&1; then
        echo "    sudo dnf install -y webkit2gtk4.1" >&2
    fi
}

if [ "$INSTALL_DESKTOP" = true ] && [ -n "$VERSION_PLAIN" ]; then
    DESKTOP_ARCHIVE_NAME="nexus-desktop_${OS_NAME}_${ARCH_NAME}.tar.gz"
    DESKTOP_DOWNLOAD_URL="${GITHUB_URL}/releases/download/v${VERSION_PLAIN}/${DESKTOP_ARCHIVE_NAME}"
    DESKTOP_CHECKSUMS_URL="${GITHUB_URL}/releases/download/v${VERSION_PLAIN}/desktop-checksums.txt"
    DESKTOP_TMP="${TMP_DIR}/${DESKTOP_ARCHIVE_NAME}"
    DESKTOP_CHECKSUMS="${TMP_DIR}/desktop-checksums.txt"

    echo "Attempting to install the native IAPro Nexus Desktop shell..."
    if http_get "$DESKTOP_DOWNLOAD_URL" "$DESKTOP_TMP" && http_get "$DESKTOP_CHECKSUMS_URL" "$DESKTOP_CHECKSUMS"; then
        DESKTOP_EXPECTED="$(awk -v name="$DESKTOP_ARCHIVE_NAME" '$2 == name || $2 == "native-artifacts/" name { print $1; exit }' "$DESKTOP_CHECKSUMS")"
        DESKTOP_ACTUAL="$(sha256_file "$DESKTOP_TMP")"
        if [ -z "$DESKTOP_EXPECTED" ] || [ "$DESKTOP_EXPECTED" != "$DESKTOP_ACTUAL" ]; then
            echo "⚠️  Native Desktop checksum verification failed; CLI installation is kept, Desktop was skipped." >&2
        else
            tar -xzf "$DESKTOP_TMP" -C "$TMP_DIR"
            DESKTOP_BINARY="${TMP_DIR}/nexus-desktop"
            if [ -x "$DESKTOP_BINARY" ]; then
                install_linux_desktop "$DESKTOP_BINARY"
                warn_webkitgtk
            fi
        fi
    elif [ "$BUILD_FROM_SOURCE" = true ]; then
        for source_desktop in "./nexus-desktop" "./cmd/nexus-desktop/build/bin/nexus-desktop"; do
            if [ -x "$source_desktop" ]; then
                install_linux_desktop "$source_desktop"
                warn_webkitgtk
                break
            fi
        done
    else
        echo "⚠️  Native Desktop artifact unavailable; CLI installation is complete."
    fi
fi

echo ""
if [ "$WITH_MAESTRO" = true ]; then
    echo "Checking Orquestrador Maestro dependency (--with-maestro requested)..."
    if ! command -v orquestrador-maestro >/dev/null 2>&1 && ! command -v maestro >/dev/null 2>&1; then
        if command -v npm >/dev/null 2>&1; then
            echo "Installing Orquestrador Maestro CLI (@iapro/orquestrador-maestro-cli)..."
            if ! npm install -g @iapro/orquestrador-maestro-cli; then
                echo "⚠️  Could not install @iapro/orquestrador-maestro-cli globally with npm." >&2
                MAESTRO_EXIT=2
            fi
        else
            echo "⚠️  Node.js / npm not detected. Maestro unavailable; install npm then re-run with --with-maestro." >&2
            MAESTRO_EXIT=2
        fi
    fi
else
    echo "Maestro auto-install skipped (Nexus does not silently install third-party packages)."
    echo "To install Maestro, run with '--with-maestro' or: npm install -g @iapro/orquestrador-maestro-cli"
fi

MAESTRO_BIN="$(command -v orquestrador-maestro 2>/dev/null || echo "")"
if [ -z "$MAESTRO_BIN" ]; then
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

append_path_rc() {
    local rc="$1"
    local line='export PATH="$HOME/.local/bin:$PATH"'
    touch "$rc"
    if grep -Fqs '.local/bin' "$rc"; then
        return 0
    fi
    printf '\n# IAPro Nexus\n%s\n' "$line" >> "$rc"
    echo "✓ Added ~/.local/bin to PATH in ${rc}"
}

if [[ ":$PATH:" != *":$TARGET_DIR:"* ]]; then
    echo ""
    if [ "$NO_PATH" = true ]; then
        echo "⚠️  ${TARGET_DIR} is not in PATH (--no-path). Add manually:"
        echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    else
        case "${SHELL:-}" in
            */zsh) append_path_rc "${HOME}/.zshrc" ;;
            */bash|*) append_path_rc "${HOME}/.bashrc"
                [ -f "${HOME}/.zshrc" ] && append_path_rc "${HOME}/.zshrc"
                ;;
        esac
        export PATH="${TARGET_DIR}:$PATH"
    fi
fi

echo ""
if command -v nexus >/dev/null 2>&1 || [ -x "${TARGET_DIR}/nexus" ]; then
    echo "Running nexus doctor (warnings do not fail install)..."
    "${TARGET_DIR}/nexus" doctor || true
fi

echo ""
echo "Quick Start:"
echo "  nexus doctor            # Check provider & platform dependencies"
echo "  nexus web               # Launch IAPro Nexus Workspace OS (Web UI)"
echo ""
echo "Note: --version=latest uses GitHub tag resolution + checksums.txt integrity only;"
echo "it is not a cryptographically signed pin until the update manifest is wired here."

exit "$MAESTRO_EXIT"

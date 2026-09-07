#!/usr/bin/env bash
set -e

REPO="kivervinicius/ai-cli"
GITHUB_URL="https://github.com/${REPO}"

VERSION="${NEXUS_VERSION:-}"
BUILD_FROM_SOURCE=false
SOURCE_REF="${NEXUS_SOURCE_REF:-}"
INSTALL_DESKTOP=true
for arg in "$@"; do
    case "$arg" in
        --version=*) VERSION="${arg#--version=}" ;;
        --build-from-source) BUILD_FROM_SOURCE=true ;;
        --source-ref=*) SOURCE_REF="${arg#--source-ref=}" ;;
        --with-desktop) INSTALL_DESKTOP=true ;;
        --no-desktop) INSTALL_DESKTOP=false ;;
        --with-maestro) ;;
        *) echo "Unknown option: $arg" >&2; exit 2 ;;
    esac
done

if [ -z "$VERSION" ] && [ "$BUILD_FROM_SOURCE" != true ]; then
    echo "A version is required for verified installation: use --version=vX.Y.Z or NEXUS_VERSION." >&2
    echo "For an explicit source build, use --build-from-source from a checkout." >&2
    exit 2
fi

if [ -n "$VERSION" ] && ! printf '%s' "$VERSION" | grep -Eq '^v?[0-9]+\.[0-9]+\.[0-9]+([.-][A-Za-z0-9.-]+)?$'; then
    echo "Invalid Nexus version: $VERSION" >&2
    exit 2
fi

echo "=== IAPro Nexus Installer (Pinned Release) ==="

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

# 2. Download a versioned release binary and verify its published digest.
ARCHIVE_NAME="nexus_${OS_NAME}_${ARCH_NAME}.tar.gz"
VERSION="${VERSION#v}"
DOWNLOAD_URL="${GITHUB_URL}/releases/download/v${VERSION}/${ARCHIVE_NAME}"
CHECKSUMS_URL="${GITHUB_URL}/releases/download/v${VERSION}/checksums.txt"

echo "Attempting to download Nexus v${VERSION}: ${ARCHIVE_NAME}..."
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

if [ -n "$VERSION" ] && command -v curl >/dev/null 2>&1; then
    if curl -fsSL "$DOWNLOAD_URL" -o "${TMP_DIR}/${ARCHIVE_NAME}" && curl -fsSL "$CHECKSUMS_URL" -o "${TMP_DIR}/checksums.txt"; then
        EXPECTED="$(awk -v name="$ARCHIVE_NAME" '$2 == name { print $1; exit }' "${TMP_DIR}/checksums.txt")"
        ACTUAL="$(sha256_file "${TMP_DIR}/${ARCHIVE_NAME}")"
        if [ -z "$EXPECTED" ] || [ "$EXPECTED" != "$ACTUAL" ]; then
            echo "Release checksum verification failed for ${ARCHIVE_NAME}." >&2
            exit 1
        fi
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
elif [ -n "$VERSION" ] && command -v wget >/dev/null 2>&1; then
    if wget -q "$DOWNLOAD_URL" -O "${TMP_DIR}/${ARCHIVE_NAME}" && wget -q "$CHECKSUMS_URL" -O "${TMP_DIR}/checksums.txt"; then
        EXPECTED="$(awk -v name="$ARCHIVE_NAME" '$2 == name { print $1; exit }' "${TMP_DIR}/checksums.txt")"
        ACTUAL="$(sha256_file "${TMP_DIR}/${ARCHIVE_NAME}")"
        if [ -z "$EXPECTED" ] || [ "$EXPECTED" != "$ACTUAL" ]; then
            echo "Release checksum verification failed for ${ARCHIVE_NAME}." >&2
            exit 1
        fi
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

# 3. Explicit source build only; never resolve a mutable latest ref implicitly.
if [ "$INSTALL_SUCCESS" -eq 0 ] && [ "$BUILD_FROM_SOURCE" = true ]; then
    if command -v go >/dev/null 2>&1; then
        echo "Building from source via Go..."
        if [ -f "./go.mod" ] && [ -d "./cmd/nexus" ]; then
            go build -ldflags="-s -w" -o "${TARGET_DIR}/nexus" ./cmd/nexus
            chmod +x "${TARGET_DIR}/nexus"
            ln -sf "${TARGET_DIR}/nexus" "${TARGET_DIR}/ai"
            INSTALL_SUCCESS=1
        elif [ -n "$SOURCE_REF" ] && command -v git >/dev/null 2>&1; then
            case "$SOURCE_REF" in *[!A-Za-z0-9._/-]*) echo "Invalid source ref." >&2; exit 2 ;; esac
            echo "Cloning explicitly requested source ref ${SOURCE_REF}..."
            git clone --depth 1 --branch "$SOURCE_REF" "${GITHUB_URL}.git" "${TMP_DIR}/repo"
            (cd "${TMP_DIR}/repo" && go build -ldflags="-s -w" -o "${TARGET_DIR}/nexus" ./cmd/nexus)
            chmod +x "${TARGET_DIR}/nexus"
            ln -sf "${TARGET_DIR}/nexus" "${TARGET_DIR}/ai"
            INSTALL_SUCCESS=1
        else
            echo "--build-from-source requires a Nexus checkout or --source-ref=<tag-or-commit>." >&2
        fi
    else
        echo "--build-from-source requires Go >=1.25." >&2
    fi
fi

if [ "$INSTALL_SUCCESS" -eq 0 ]; then
    echo "Installation failed: no verified release artifact was installed." >&2
    exit 1
fi

echo "✓ Successfully installed IAPro Nexus to ${TARGET_DIR}/nexus (with 'ai' alias)"

install_linux_desktop() {
    local source_binary="$1"
    local desktop_target="${TARGET_DIR}/nexus-desktop"
    cp "$source_binary" "$desktop_target"
    chmod +x "$desktop_target"

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

# 3b. Install the native desktop shell when the release publishes it. The CLI
# remains usable when a platform has no native desktop artifact or checksum.
if [ "$INSTALL_DESKTOP" = true ]; then
    DESKTOP_ARCHIVE_NAME="nexus-desktop_${OS_NAME}_${ARCH_NAME}.tar.gz"
    DESKTOP_DOWNLOAD_URL="${GITHUB_URL}/releases/download/v${VERSION}/${DESKTOP_ARCHIVE_NAME}"
    DESKTOP_CHECKSUMS_URL="${GITHUB_URL}/releases/download/v${VERSION}/desktop-checksums.txt"
    DESKTOP_TMP="${TMP_DIR}/${DESKTOP_ARCHIVE_NAME}"
    DESKTOP_CHECKSUMS="${TMP_DIR}/desktop-checksums.txt"

    echo "Attempting to install the native IAPro Nexus Desktop shell..."
    if command -v curl >/dev/null 2>&1 && \
        curl -fsSL "$DESKTOP_DOWNLOAD_URL" -o "$DESKTOP_TMP" && \
        curl -fsSL "$DESKTOP_CHECKSUMS_URL" -o "$DESKTOP_CHECKSUMS"; then
        DESKTOP_EXPECTED="$(awk -v name="$DESKTOP_ARCHIVE_NAME" '$2 == name || $2 == "native-artifacts/" name { print $1; exit }' "$DESKTOP_CHECKSUMS")"
        DESKTOP_ACTUAL="$(sha256_file "$DESKTOP_TMP")"
        if [ -z "$DESKTOP_EXPECTED" ] || [ "$DESKTOP_EXPECTED" != "$DESKTOP_ACTUAL" ]; then
            echo "⚠️  Native Desktop checksum verification failed; CLI installation is kept, Desktop was skipped." >&2
        else
            tar -xzf "$DESKTOP_TMP" -C "$TMP_DIR"
            DESKTOP_BINARY="${TMP_DIR}/nexus-desktop"
            if [ -x "$DESKTOP_BINARY" ]; then
                install_linux_desktop "$DESKTOP_BINARY"
            fi
        fi
    elif [ "$BUILD_FROM_SOURCE" = true ]; then
        for source_desktop in "./nexus-desktop" "./cmd/nexus-desktop/build/bin/nexus-desktop"; do
            if [ -x "$source_desktop" ]; then
                install_linux_desktop "$source_desktop"
                break
            fi
        done
    else
        echo "⚠️  Native Desktop artifact unavailable; CLI installation is complete."
    fi
fi

# 4. Check and install Maestro dependency (OPT-IN ONLY)
WITH_MAESTRO=false
for arg in "$@"; do
    case "$arg" in
        --with-maestro)
            WITH_MAESTRO=true
            ;;
    esac
done

echo ""
if [ "$WITH_MAESTRO" = true ]; then
    echo "Checking Orquestrador Maestro dependency (--with-maestro requested)..."
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
else
    echo "Maestro auto-install skipped (Nexus does not silently install third-party packages)."
    echo "To install Maestro orchestration capabilities, run with '--with-maestro' or install manually:"
    echo "  npm install -g @iapro/orquestrador-maestro-cli"
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

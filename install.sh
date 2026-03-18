#!/usr/bin/env bash
set -euo pipefail

# Outpost CLI installer.
# Usage: curl -fsSL https://git.grimes.pro/wesleygrimes/outpost/raw/branch/main/install.sh | bash

REPO="https://git.grimes.pro/wesleygrimes/outpost"
INSTALL_DIR="/usr/local/bin"

# Detect OS and arch.
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

case "$OS" in
    linux|darwin) ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

BINARY="outpost-${OS}-${ARCH}"

# Get latest release tag.
echo "Detecting latest version..."
VERSION=$(curl -sS "${REPO}/releases/latest" -o /dev/null -w '%{redirect_url}' | grep -o '[^/]*$')
if [ -z "$VERSION" ]; then
    echo "Could not detect latest version. Using 'latest'."
    VERSION="latest"
fi

echo "Installing outpost ${VERSION} (${OS}/${ARCH})..."

DOWNLOAD_URL="${REPO}/releases/download/${VERSION}/${BINARY}"

# Download to temp file.
TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT

if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP"; then
    echo "Download failed. URL: $DOWNLOAD_URL"
    exit 1
fi

chmod +x "$TMP"

# Install (may need sudo).
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP" "${INSTALL_DIR}/outpost"
else
    echo "Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo mv "$TMP" "${INSTALL_DIR}/outpost"
fi

echo "Installed outpost to ${INSTALL_DIR}/outpost"
echo ""
echo "Next: outpost server setup <host>"

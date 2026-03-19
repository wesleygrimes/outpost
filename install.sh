#!/usr/bin/env bash
set -euo pipefail

# Outpost CLI installer.
# Usage: curl -fsSL https://git.grimes.pro/wesleygrimes/outpost/raw/branch/main/install.sh | bash

REPO="https://git.grimes.pro/wesleygrimes/outpost"
INSTALL_DIR="${HOME}/.local/bin"

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

mkdir -p "$INSTALL_DIR"
mv "$TMP" "${INSTALL_DIR}/outpost"

echo "Installed outpost to ${INSTALL_DIR}/outpost"

# Install Claude Code slash commands.
COMMANDS_DIR="${HOME}/.claude/commands"
COMMANDS_BASE="${REPO}/raw/branch/main/commands"
mkdir -p "$COMMANDS_DIR"
for cmd in outpost outpost-drop outpost-pickup outpost-status; do
    curl -fsSL "${COMMANDS_BASE}/${cmd}.md" -o "${COMMANDS_DIR}/${cmd}.md"
done
echo "Installed slash commands to ${COMMANDS_DIR}/"
echo ""

# Check if INSTALL_DIR is in PATH.
case ":$PATH:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
        echo "Add ${INSTALL_DIR} to your PATH:"
        echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
        echo ""
        ;;
esac

echo "Next: outpost server setup <host>"

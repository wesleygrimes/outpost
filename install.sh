#!/usr/bin/env bash
set -euo pipefail

# Outpost CLI installer.
# Usage: curl -fsSL https://git.grimes.pro/wesleygrimes/outpost/raw/branch/main/install.sh | bash

REPO="https://git.grimes.pro/wesleygrimes/outpost"
GITEA_REPO="https://git.grimes.pro/wesleygrimes/outpost.git"
INSTALL_DIR="${HOME}/.local/bin"

# --- Detect platform ---

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
    *)       echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    linux|darwin) ;;
    *)            echo "Unsupported OS: $OS"; exit 1 ;;
esac

# --- Detect latest version ---

echo "Detecting latest version..."
VERSION=$(curl -sS "${REPO}/releases/latest" -o /dev/null -w '%{redirect_url}' | grep -o '[^/]*$')
if [ -z "$VERSION" ]; then
    echo "Could not detect latest version. Using 'latest'."
    VERSION="latest"
fi

echo "Installing outpost ${VERSION} (${OS}/${ARCH})..."

# --- Install binary ---

TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT

if ! curl -fsSL "${REPO}/releases/download/${VERSION}/outpost-${OS}-${ARCH}" -o "$TMP"; then
    echo "Download failed."
    exit 1
fi

chmod +x "$TMP"
mkdir -p "$INSTALL_DIR"
mv "$TMP" "${INSTALL_DIR}/outpost"
echo "Installed binary to ${INSTALL_DIR}/outpost"

# --- Install Claude Code plugin ---

if command -v claude >/dev/null 2>&1; then
    claude plugin marketplace add "${GITEA_REPO}" 2>/dev/null || true
    claude plugin marketplace update outpost-marketplace 2>/dev/null || true
    claude plugin install outpost@outpost-marketplace 2>/dev/null \
        || claude plugin update outpost@outpost-marketplace 2>/dev/null || true
    echo "Installed Claude Code plugin."
else
    echo "Claude Code CLI not found, skipping plugin install."
    echo "Install manually: claude plugin marketplace add ${GITEA_REPO} && claude plugin install outpost@outpost-marketplace"
fi

# --- Clean up old-style slash commands ---

for old in outpost outpost-drop outpost-pickup outpost-status; do
    rm -f "${HOME}/.claude/commands/${old}.md"
done

# --- PATH check ---

echo ""
case ":$PATH:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
        echo "Add ${INSTALL_DIR} to your PATH:"
        echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
        echo ""
        ;;
esac

echo "Next: outpost server setup <host>"

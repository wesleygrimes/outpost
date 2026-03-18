#!/usr/bin/env bash
set -euo pipefail

# Outpost VM provisioning script for Debian 12.
# Run as root: sudo ./setup.sh

if [ "$(id -u)" -ne 0 ]; then
    echo "Run as root: sudo $0"
    exit 1
fi

echo "=== Outpost VM Provisioning ==="

# System packages.
echo "Installing system packages..."
apt-get update -qq
apt-get install -y -qq git curl build-essential

# Go.
GO_VERSION="1.26.1"
if command -v go &>/dev/null && go version | grep -q "$GO_VERSION"; then
    echo "Go $GO_VERSION already installed."
else
    echo "Installing Go $GO_VERSION..."
    curl -sL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o /tmp/go.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf /tmp/go.tar.gz
    rm /tmp/go.tar.gz

    # Add to system profile if not already there.
    if ! grep -q '/usr/local/go/bin' /etc/profile.d/go.sh 2>/dev/null; then
        echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
    fi

    echo "Go $GO_VERSION installed."
fi

# Node.js 20 (for Claude Code).
if command -v node &>/dev/null && node --version | grep -q 'v20'; then
    echo "Node.js 20 already installed."
else
    echo "Installing Node.js 20..."
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
    apt-get install -y -qq nodejs
    echo "Node.js $(node --version) installed."
fi

# Zellij.
if command -v zellij &>/dev/null; then
    echo "Zellij already installed."
else
    echo "Installing Zellij..."
    ZELLIJ_URL=$(curl -s https://api.github.com/repos/zellij-org/zellij/releases/latest \
        | grep 'browser_download_url.*x86_64.*linux.*musl' \
        | head -1 \
        | cut -d '"' -f 4)

    if [ -z "$ZELLIJ_URL" ]; then
        echo "Error: Could not determine Zellij download URL."
        echo "Install manually: https://github.com/zellij-org/zellij/releases"
        exit 1
    fi

    curl -sL "$ZELLIJ_URL" -o /tmp/zellij.tar.gz
    tar -C /usr/local/bin -xzf /tmp/zellij.tar.gz zellij
    chmod +x /usr/local/bin/zellij
    rm /tmp/zellij.tar.gz
    echo "Zellij installed."
fi

echo ""
echo "=== Provisioning Complete ==="
echo ""
echo "Next steps:"
echo "  1. Build outpost: go build -o ~/.outpost/bin/outpost ."
echo "  2. Run setup:     ~/.outpost/bin/outpost setup"
echo "  3. Auth Claude:   claude"
echo "  4. Start daemon:  sudo systemctl start outpost"

# Outpost

Remote Claude Code session runner. One command to provision, one command to connect.

## Install

```bash
curl -fsSL https://git.grimes.pro/wesleygrimes/outpost/raw/branch/main/install.sh | bash
```

Or build from source:

```bash
git clone https://git.grimes.pro/wesleygrimes/outpost.git
cd outpost
make build
sudo mv bin/outpost /usr/local/bin/
```

## Quick Start

```bash
# Provision a remote server (builds, uploads, configures via SSH)
outpost server setup myserver

# Or connect to an existing server
outpost login outpost.grimes.pro:7600 <token>

# Hand off work
outpost handoff --plan plan.md --mode headless --branch feat/auth

# Check status
outpost status

# Download results
outpost pickup <run-id>
```

## Commands

### Server

| Command | Description |
|---------|-------------|
| `outpost server setup` | Configure this machine as an outpost server |
| `outpost server setup <ssh-target>` | Provision a remote server via SSH |
| `outpost server doctor` | Check server health via gRPC |
| `outpost serve` | Start the gRPC daemon |

### Client

| Command | Description |
|---------|-------------|
| `outpost login <host> <token>` | Connect to a server |
| `outpost doctor` | Check client health |
| `outpost handoff --plan <path>` | Push work to the server |
| `outpost status` | Dashboard with active runs and history |
| `outpost status <id>` | Single run detail |
| `outpost status <id> --follow` | Tail logs live |
| `outpost pickup <id>` | Download completed patch |
| `outpost drop <id>` | Discard a run |
| `outpost version` | Print version |

## How It Works

1. **Handoff**: Archives your repo, streams it to the server over gRPC
2. **Server**: Extracts, creates a git commit, spawns Claude Code in tmux (interactive) or bash (headless)
3. **Status**: Polls the server for run status, streams logs
4. **Pickup**: Downloads the diff patch, applies it locally

## Architecture

Single Go binary with two roles:

- **Server** (`outpost serve`): gRPC daemon managing Claude Code sessions in tmux/bash. Stores runs under `~/.outpost/runs/`. TLS with self-signed certs, token auth.
- **Client** (all other commands): Connects to the server over gRPC. Config stored at `~/.config/outpost/config.yaml`.

### TLS

Behind a reverse proxy (Traefik, Caddy, nginx) with real certs: client uses system TLS, no extra config needed.

Direct access with self-signed certs: use `--ca-cert` on login, or let `server setup <host>` handle it automatically.

## Config

**Server** (`~/.outpost/config.yaml`):
```yaml
port: 7600
token: <generated>
max_concurrent_runs: 3
tls_cert: ~/.outpost/tls/server.pem
tls_key: ~/.outpost/tls/server-key.pem
tls_ca: ~/.outpost/tls/ca.pem
```

**Client** (`~/.config/outpost/config.yaml`):
```yaml
server: outpost.grimes.pro:7600
token: <from login>
ca_cert: <optional, path to CA>
```

## Development

```bash
make check    # vet + lint + test
make build    # local binary
make proto    # regenerate protobuf
make fmt      # format with gofumpt
```

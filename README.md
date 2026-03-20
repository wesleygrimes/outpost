# Outpost

> Async AI coding agent runner. Hand off tasks, keep working, pick up results.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev)

---

AI coding agents are 10x developers, but they can only do one thing at a time in your terminal. Outpost lets you hand off tasks and keep working. Hand off three refactors before lunch, pick up the patches after. Your agent becomes a team, not a pair.

- **Stop babysitting.** Hand off a task, get a patch back. No terminal to watch.
- **Fire and forget.** Start a refactor before lunch, pick it up after. Start a migration before dinner, pick up the patch in the morning.
- **Parallel, not serial.** Run multiple agent sessions at once. Review results as they land.
- **Local or remote.** Run on your laptop or offload to dedicated hardware. Same workflow.
- **One binary, zero infrastructure.** No Docker, no cloud. `outpost serve` and go.
- **Git in, git out.** Repo goes up, patch comes back. Clean diffs, easy review.

## How It Works

1. **Hand off** — Archive your repo and task, send it to the Outpost server
2. **Run** — Outpost spawns your AI coding agent in the background
3. **Monitor** — Check status, tail logs, keep working on other things
4. **Pick up** — Download the patch, review the diff, apply it locally

## Quick Start

```bash
# Start the server (locally or on a remote box)
outpost serve

# Hand off a task
outpost handoff --session-id <uuid> --mode headless --branch feat/auth

# Check on it
outpost status

# Grab the results
outpost pickup <run-id>
```

## Install

### From Source

```bash
git clone https://github.com/wesleygrimes/outpost.git
cd outpost
make build
sudo mv bin/outpost /usr/local/bin/
```

### Claude Code Plugin

Outpost ships as a Claude Code plugin with slash commands for handoff, status, logs, pickup, drop, and watch.

```bash
claude plugin marketplace add https://github.com/wesleygrimes/outpost.git
claude plugin install outpost@outpost-marketplace
```

## Commands

### Server

| Command | Description |
|---------|-------------|
| `outpost serve` | Start the server daemon |
| `outpost server setup` | Configure the local machine as a server |
| `outpost server setup <ssh-target>` | Provision a remote server via SSH |
| `outpost server doctor` | Check server health |

### Client

| Command | Description |
|---------|-------------|
| `outpost login <host> <token>` | Connect to a server |
| `outpost doctor` | Check client health |
| `outpost handoff` | Hand off a task |
| `outpost status` | Dashboard of runs |
| `outpost status <id>` | Single run detail |
| `outpost status <id> --follow` | Tail logs live |
| `outpost logs <id>` | View log output |
| `outpost pickup <id>` | Download completed patch |
| `outpost drop <id>` | Discard a run |

## Architecture

Single Go binary, two roles:

- **Server** (`outpost serve`) — gRPC daemon that manages agent sessions. TLS + token auth. Runs agents in tmux (interactive) or headless.
- **Client** (everything else) — Connects over gRPC. Config at `~/.config/outpost/config.yaml`.

<details>
<summary>Configuration</summary>

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
server: localhost:7600
token: <from login>
ca_cert: <optional, path to CA>
```

</details>

<details>
<summary>Development</summary>

```bash
make check    # vet + lint + test
make build    # local binary
make proto    # regenerate protobuf
make fmt      # format with gofumpt
```

</details>

## License

[MIT](LICENSE)

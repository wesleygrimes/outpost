<p align="center">
  <img src="logo.svg" alt="Outpost" width="100">
</p>

<h1 align="center">Outpost</h1>

<p align="center">
  <strong>Async AI coding agent runner. Hand off tasks, keep working, pick up results.</strong>
</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white" alt="Go"></a>
</p>

---

AI coding agents are 10x developers, but they can only do one thing at a time in your terminal. Outpost lets you hand off tasks and keep working. Hand off three refactors before lunch, pick up the patches after. Your agent becomes a team, not a pair.

- **Stop babysitting.** Hand off a task, get a patch back. No terminal to watch.
- **Fire and forget.** Start a refactor before lunch, pick it up after. Start a migration before dinner, pick up the patch in the morning.
- **Parallel, not serial.** Run multiple agent sessions at once. Review results as they land.
- **Local or remote.** Run on your laptop or offload to dedicated hardware. Same workflow.
- **One binary, zero infrastructure.** No Docker, no cloud. `outpost serve` and go.
- **Git in, git out.** Repo goes up, patch comes back. Clean diffs, easy review.

---

## See It In Action

### Hand off a task

```
$ outpost handoff --session-id abc123 --mode headless --branch feat/auth

  OUTPOST v0.1.0  Handoff -> myserver.local

  |  ✓ Archiving repo (1,247 files, 18.3 MB)
  |  ✓ Bundling session abc123
  |  ✓ Streamed (18.3 MB in 2.1s)
  |  ✓ Run started
  |
  └  Run op-7f3a | headless | max 50 turns | branch feat/auth

  Watch:  outpost status op-7f3a --follow
  Logs:   outpost logs op-7f3a --tail
```

### Check your runs

```
$ outpost status

  OUTPOST v0.1.0  Runs on myserver.local

  ID        Branch                    Mode      Turns   Status       Age
  op-7f3a   feat/auth                 headless  4/10    ⠸ running    2m
  op-a1c2   feat/onboarding-flow      headless  10/10   ● done       18m
  op-d4e5   refactor/api-cache        interact  6/20    ◉ waiting    1h
  op-f8g9   fix/hydration-error       headless  3/5     ✗ failed     4h

  4 runs total  ·  1 running  ·  1 done  ·  1 waiting  ·  1 failed
```

### Pick up the results

```
$ outpost pickup op-a1c2

  OUTPOST v0.1.0  Pickup op-a1c2

  |  ✓ Downloaded patch (4.2 KB)
  |  ✓ Downloaded session fork
  |
  |  Patch:  .outpost/patches/op-a1c2.patch
  |  Files:  +3 new  ~2 modified  -1 deleted
  |
  |    + app/services/onboarding.go
  |    + app/services/onboarding_test.go
  |    + internal/templates/welcome.tmpl
  |    ~ app/routes.go
  |    ~ config/defaults.yaml
  |    - app/services/legacy_onboarding.go
  |
  └  Apply: git apply .outpost/patches/op-a1c2.patch
```

---

## Use It From Claude Code

Outpost ships as a Claude Code plugin. Hand off work, check on it, and pick up results without leaving your session.

```
You:   /outpost:handoff headless --branch feat/auth --max-turns 20

Agent: Handed off to Outpost. Run op-7f3a started on myserver.local.
       Mode: headless | Max turns: 20 | Branch: feat/auth

       Watch:  /outpost:watch op-7f3a
       Status: /outpost:status op-7f3a
```

Keep working. Outpost watches in the background and tells you when it's done.

```
You:   /outpost:watch op-7f3a

       ... 15 minutes later, while you're working on something else ...

Agent: Run op-7f3a completed. +218 -14 across 6 files.
       Pick up: /outpost:pickup op-7f3a

You:   /outpost:pickup op-7f3a

Agent: Patch applied. Running make check... all checks pass.
       Committed: feat(auth): add JWT middleware and route guards
```

---

## Use Cases

**"I have three features to ship this week"**
Hand off each as a separate run. Check the dashboard between meetings. Pick them up one by one, review the diffs, commit. What used to be three sequential sessions is now parallel.

**"This migration will take 30 minutes"**
Fire and forget. Go to lunch. Come back, run `outpost pickup`, review the diff. Your agent doesn't need you watching.

**"My laptop fans are screaming"**
Set up Outpost on a beefy server or desktop. Same commands, more headroom. Your laptop stays cool.

**"I want to try something risky"**
Hand off an experimental refactor. If the patch looks wrong, `outpost drop` it. Your working tree never changed.

---

## Install

### From Source

```bash
git clone https://github.com/wesleygrimes/outpost.git
cd outpost
make build
sudo mv bin/outpost /usr/local/bin/
```

### Claude Code Plugin

```bash
claude plugin marketplace add https://github.com/wesleygrimes/outpost.git
claude plugin install outpost@outpost-marketplace
```

Adds `/outpost:handoff`, `/outpost:status`, `/outpost:logs`, `/outpost:pickup`, `/outpost:drop`, and `/outpost:watch`.

---

## Quick Start

```bash
# Start the server (locally or on a remote box)
outpost serve

# Or provision a remote server via SSH
outpost server setup myserver

# Hand off a task
outpost handoff --session-id <uuid> --mode headless --branch feat/auth

# Check status
outpost status

# Download results
outpost pickup <run-id>
```

<details>
<summary><strong>Commands</strong></summary>

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

</details>

<details>
<summary><strong>Architecture</strong></summary>

Single Go binary, two roles:

- **Server** (`outpost serve`) -- gRPC daemon that manages agent sessions. TLS + token auth. Runs agents in tmux (interactive) or headless.
- **Client** (everything else) -- Connects over gRPC. Config at `~/.config/outpost/config.yaml`.

</details>

<details>
<summary><strong>Configuration</strong></summary>

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
<summary><strong>Development</strong></summary>

```bash
make check    # vet + lint + test
make build    # local binary
make proto    # regenerate protobuf
make fmt      # format with gofumpt
```

</details>

## License

[MIT](LICENSE)

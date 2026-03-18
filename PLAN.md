# Outpost — Implementation Plan (v2)

## Overview

Outpost is a remote Claude Code session runner. It runs as a daemon on a dedicated Proxmox VM, accepts implementation plans with git bundles from your work laptop, executes them in isolated environments, and produces patch files you pick up and apply locally.

The server is stateless with respect to repos. Each handoff includes a full git bundle — the server doesn't need to know about your projects, remotes, or GitHub. It just receives a bundle, unbundles it, runs Claude Code against it, and produces a patch.

## Architecture

```
Laptop (Claude Code)                        Proxmox VM (Debian 12)
┌──────────────────────────┐               ┌─────────────────────────────┐
│                          │               │  outpost daemon :7600       │
│ /outpost                 │               │                             │
│   git bundle create      │               │  ~/.outpost/                │
│   POST /handoff ─────────│── HTTP ──────▶│    config.yaml              │
│   (bundle + plan)        │               │    runs/                    │
│                          │               │      <run-id>/              │
│ /outpost-status [id]     │── HTTP ──────▶│        repo/    (unbundled) │
│                          │               │        plan.md              │
│ /outpost-pickup <id>     │               │        output.log           │
│   GET /runs/<id>/patch   │◀── .patch ────│        result.patch         │
│   git worktree add       │               │                             │
│   git apply              │               │  zellij sessions            │
│   review → PR            │               │    one per run, isolated    │
│                          │               │                             │
│ ssh outpost -t           │               └─────────────────────────────┘
│   'zellij attach <id>'   │── SSH ───────▶│
└──────────────────────────┘
```

## Config File

Location: `~/.outpost/config.yaml`

```yaml
server:
  port: 7600
  token: "generated-hex-token"
  max_concurrent_runs: 3
```

That's it. No project definitions. The bundle is the project.

`max_concurrent_runs` defaults to 3. `POST /handoff` returns 429 if at capacity. This is important because concurrent headless sessions burn through Claude Max rate limits fast — Max 5 will struggle with more than 1-2 concurrent, Max 20 handles 3 comfortably. Disk is also a factor: each run unbundles a full copy of the repo, so 3 concurrent runs of a 2GB monorepo means ~6GB+ of working copies plus the bundles themselves.

## Single Binary — Subcommands

### `outpost setup`

One-time setup wizard. Run on a fresh VM after installing the binary.

1. Creates `~/.outpost/` and `~/.outpost/runs/`
2. Generates a bearer token (32 bytes hex)
3. Writes `~/.outpost/config.yaml`
4. Installs systemd unit file at `/etc/systemd/system/outpost.service`
5. Enables the service (`systemctl enable outpost`)
6. Checks if Claude Code is installed and authenticated:
   - If `claude` not found: prints install instructions
   - If `~/.claude` doesn't exist: prompts to run `claude` for Max subscription auth
7. Prints summary:
   - The generated token (save on laptop as `~/.outpost-token`)
   - SSH config snippet for laptop
   - How to start: `sudo systemctl start outpost`

### `outpost serve`

Starts the HTTP server in the foreground. This is what systemd calls — not intended for direct use, but works for debugging.

Reads config from `~/.outpost/config.yaml`. Env var overrides: `OUTPOST_PORT`, `OUTPOST_TOKEN`.

### `outpost runs`

Lists runs. Reads from the daemon's API if it's running, falls back to scanning `~/.outpost/runs/` on disk.

```
$ outpost runs
ID                              STATUS    MODE         CREATED
refactor-20260317-143022-a1b2   running   interactive  2 hours ago
fix-nav-20260317-160000-c3d4    complete  headless     45 min ago
```

### `outpost runs <id>`

Detail view for a single run: status, base SHA, final SHA, log tail, patch readiness, attach command.

---

That's the entire CLI. Three commands.

## API Endpoints

All require `Authorization: Bearer <token>` except `/health`.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `POST` | `/handoff` | Submit a run (multipart: bundle + plan + options) |
| `GET` | `/runs` | List all runs |
| `GET` | `/runs/{id}` | Run detail with log tail and patch readiness |
| `GET` | `/runs/{id}/patch` | Download the .patch file |
| `GET` | `/runs/{id}/log` | Download full output log |
| `DELETE` | `/runs/{id}` | Kill a run (captures partial patch) |
| `POST` | `/runs/{id}/cleanup` | Delete the run's directory from disk (bundle, repo copy, logs). Call after successful pickup. Returns 409 if the run is still active. |

### `POST /handoff`

Multipart form data:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `plan` | string | yes | Implementation plan markdown |
| `bundle` | file | yes | Git bundle of the repo |
| `mode` | string | no | `interactive` (default) or `headless` |
| `name` | string | no | Human-friendly run name prefix |
| `branch` | string | no | Branch to create in the unbundled repo |
| `max_turns` | string | no | Max agent turns, headless only (default: 50) |

Returns 429 Too Many Requests if `max_concurrent_runs` running sessions already exist. The response includes the current run count and which runs are active so the caller can decide whether to wait or kill one.

### `GET /runs/{id}` Response

```json
{
  "id": "refactor-20260317-143022-a1b2",
  "name": "refactor",
  "mode": "interactive",
  "status": "complete",
  "base_sha": "abc123...",
  "final_sha": "def456...",
  "created_at": "2026-03-17T14:30:22Z",
  "finished_at": "2026-03-17T16:45:00Z",
  "attach": "ssh outpost -t 'zellij attach refactor-20260317-143022-a1b2'",
  "log_tail": "... last 80 lines ...",
  "patch_ready": true
}
```

## Run Lifecycle

1. **Receive** — POST with bundle + plan
2. **Prepare** — generate run ID, create `~/.outpost/runs/<id>/`, unbundle repo, write plan
3. **Execute** — spawn zellij session with Claude Code
   - Interactive: `claude "Read the plan at /path/plan.md and execute it fully. Do not ask clarifying questions."`
   - Headless: `claude -p --dangerously-skip-permissions --max-turns N "$(cat plan.md)"`
4. **Capture** — on Claude Code exit: `git add -A && git diff --cached <base_sha> > result.patch`
5. **Serve** — patch available via `GET /runs/{id}/patch`
6. **Cleanup** — only triggered by `POST /runs/{id}/cleanup` after a successful pickup on the laptop. Deletes `~/.outpost/runs/<id>/` entirely (repo copy, bundle, logs, plan). Run metadata stays in memory so `/outpost-status` still shows it as "cleaned up". Never auto-cleans — a failed pickup leaves the run intact for retry.

## Laptop-Side Claude Code Slash Commands

Four commands, installed to `~/.claude/commands/`:

### `/outpost [headless|interactive] [--name N] [--branch B] [--max-turns N]`

1. Compile the implementation plan from conversation into a self-contained markdown document. Must include everything — the remote session has zero context from this conversation.
2. Create a git bundle of the current repo: `git bundle create /tmp/outpost-$$.bundle --all`
3. POST to Outpost:
   ```bash
   curl -s -X POST "$OUTPOST_URL/handoff" \
     -H "Authorization: Bearer $(cat ~/.outpost-token)" \
     -F "plan=</tmp/outpost-plan-$$.md" \
     -F "bundle=@/tmp/outpost-$$.bundle" \
     -F "mode=$MODE" \
     -F "name=$NAME" \
     -F "branch=$BRANCH" \
     -F "max_turns=$MAX_TURNS"
   ```
4. Report: run ID, attach command, how to check status, how to pick up
5. Clean up temp files

### `/outpost-status [run_id]`

- No args: `GET /runs` — list all
- With ID: `GET /runs/{id}` — detail with log tail
- If complete + patch ready, suggest `/outpost-pickup <id>`

### `/outpost-pickup <run_id>`

1. Verify run is complete and patch is ready via `GET /runs/{id}`
2. Download patch: `GET /runs/{id}/patch` → `/tmp/outpost-patches/<id>.patch`
3. Create local worktree: `git worktree add -b outpost/<id> ../outpost-<id>`
4. Apply patch: `cd ../outpost-<id> && git apply <patch> && git add -A && git commit -m "outpost: run <id>"`
5. **If step 4 fails** (patch doesn't apply cleanly): report the error, remove the local worktree, but **do NOT clean up the remote run**. The run stays on disk so the user can retry pickup, attach via SSH to inspect, or manually resolve. Exit here.
6. **Only on success**: `POST /runs/{id}/cleanup` — deletes the run directory on Outpost (unbundled repo, bundle file, logs). Frees disk for future runs.
7. Clean up local temp: `rm -f /tmp/outpost-patches/<id>.patch`
8. Report: worktree path, branch name, files changed, next steps

### `/outpost-kill <run_id>`

1. `DELETE /runs/{id}` — kills zellij session, generates patch from partial work
2. Report status, mention partial pickup is available

## Project Structure

```
outpost/
├── main.go                 # Entry point, subcommand dispatch (os.Args)
├── cmd/
│   ├── setup.go            # outpost setup — interactive wizard
│   ├── serve.go            # outpost serve — HTTP daemon
│   └── runs.go             # outpost runs [id]
├── internal/
│   ├── config/
│   │   └── config.go       # YAML config load/save
│   ├── server/
│   │   ├── server.go       # HTTP server, middleware, router
│   │   ├── handoff.go      # POST /handoff
│   │   ├── runs.go         # GET/DELETE /runs handlers
│   │   └── cleanup.go      # POST /runs/{id}/cleanup — disk cleanup
│   ├── runner/
│   │   ├── runner.go       # Zellij session lifecycle
│   │   ├── bundle.go       # Git bundle → unbundle
│   │   └── patch.go        # Patch generation (git diff)
│   └── store/
│       └── store.go        # In-memory run store with RWMutex
├── commands/                # Claude Code slash commands (copy to laptop)
│   ├── outpost.md
│   ├── outpost-status.md
│   ├── outpost-pickup.md
│   └── outpost-kill.md
├── setup.sh                # VM provisioning (system packages, Go, Node, Zellij)
├── go.mod
├── go.sum
└── README.md
```

## Dependencies

Go modules:
- `gopkg.in/yaml.v3` — config file parsing

No HTTP framework (stdlib `net/http` with Go 1.22 routing). No CLI framework (simple `os.Args[1]` switch — three subcommands doesn't warrant cobra).

System (installed by `setup.sh`):
- Debian 12 minimal
- Go 1.22+
- Node.js 20 (for Claude Code)
- Zellij
- Git

## Systemd Unit

Installed by `outpost setup`:

```ini
[Unit]
Description=Outpost - Remote Claude Code Runner
After=network.target

[Service]
Type=simple
User=wes
ExecStart=/home/wes/.outpost/bin/outpost serve
Restart=on-failure
RestartSec=5
WorkingDirectory=/home/wes/.outpost
Environment=PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin
Environment=HOME=/home/wes

[Install]
WantedBy=multi-user.target
```

## State Persistence

v1: in-memory run store. Runs are lost on daemon restart. However:
- Run directories persist on disk at `~/.outpost/runs/<id>/`
- Patches persist on disk
- `outpost runs` CLI falls back to scanning disk when daemon isn't responding
- Good enough — you're not running hundreds of concurrent sessions

v2 (future): persist run metadata to a JSON file or SQLite if needed.

## Security

- Bearer token auth on all API endpoints (constant-time comparison)
- Server never connects to GitHub or any git remote
- Each run gets an isolated repo copy under `~/.outpost/runs/<id>/repo/`
- Headless mode uses `--dangerously-skip-permissions` but blast radius is contained to the run's repo copy
- Config file at `~/.outpost/config.yaml` should be chmod 600
- Token on laptop at `~/.outpost-token` should be chmod 600

## Setup Flow

### On the Proxmox VM (one-time)

```bash
# 1. Create Debian 12 VM in Proxmox (2 cores, 4GB RAM)
# 2. SSH in, copy outpost source or download release binary
# 3. Install system deps
sudo ./setup.sh

# 4. Run outpost setup wizard
outpost setup

# 5. Auth Claude Code (Max subscription — opens browser)
claude

# 6. Start the daemon
sudo systemctl start outpost

# 7. Verify
curl http://localhost:7600/health
```

### On the laptop (one-time)

```bash
# 1. Save the token from setup output
echo '<token>' > ~/.outpost-token && chmod 600 ~/.outpost-token

# 2. Set Outpost URL (or add to shell rc)
export OUTPOST_URL=http://outpost.grimes.pro:7600

# 3. SSH config for attaching to interactive sessions
cat >> ~/.ssh/config << EOF
Host outpost
    HostName <vm-ip>
    User wes
EOF

# 4. Install Claude Code slash commands
cp outpost/commands/* ~/.claude/commands/
```

### Daily workflow

```
# Plan in Claude Code...
# When ready to hand off:
/outpost headless --name big-refactor --branch feature/refactor

# Check later:
/outpost-status big-refactor-20260317-143022-a1b2

# Or attach to watch:
ssh outpost -t 'zellij attach big-refactor-20260317-143022-a1b2'

# When done, pull it back:
/outpost-pickup big-refactor-20260317-143022-a1b2

# Review in the worktree, merge or PR when satisfied
```

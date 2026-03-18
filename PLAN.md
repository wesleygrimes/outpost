# Outpost v2: gRPC Rewrite

Greenfield rewrite from REST/HTTP to gRPC with protobuf and TLS.

## User Stories

### Provision a New Outpost Server

```
laptop$ make build-linux
laptop$ scp bin/outpost-linux server:~/outpost

laptop$ ssh server

server$ ~/outpost setup
  Creating directories...
  Generating config with token...
  Generating TLS certificates...
  Checking Claude Code installation...

  === Outpost Setup Complete ===

  Token:   a1b2c3d4...
  Address: 0.0.0.0:7600
  CA cert: ~/.outpost/tls/ca.pem

  On your laptop:
    scp server:~/.outpost/tls/ca.pem /tmp/outpost-ca.pem
    outpost login <this-host>:7600 a1b2c3d4... --ca-cert /tmp/outpost-ca.pem

server$ sudo systemctl start outpost
```

What `setup` creates on disk:

```
~/.outpost/
  config.yaml              # port, token, max_concurrent_runs, tls paths
  bin/outpost              # copied binary
  tls/
    ca.pem                 # self-signed CA certificate
    ca-key.pem             # CA private key (never leaves server)
    server.pem             # server certificate (signed by CA)
    server-key.pem         # server private key
  runs/                    # run data directory
```

### Connect Local Machine to Outpost

```
laptop$ scp server:~/.outpost/tls/ca.pem /tmp/outpost-ca.pem
laptop$ outpost login myserver:7600 a1b2c3d4... --ca-cert /tmp/outpost-ca.pem

  Verifying connection... healthy
  Credentials saved:
    ~/.outpost-url           # "myserver:7600"
    ~/.outpost-token         # "a1b2c3d4..."
    ~/.outpost-ca.pem        # CA certificate (copied)
```

After login, Claude Code skills work automatically. The `outpost` binary must be in `$PATH`.

### Daily Workflow via Claude Code Skills

**Hand off work:**
```
user> /outpost headless --name fix-auth-bug --branch fix/auth

  [Claude compiles plan from conversation into /tmp/outpost-plan.md]

  $ outpost handoff --plan /tmp/outpost-plan.md --mode headless \
      --name fix-auth-bug --branch fix/auth

  uploading... 3.4 MB / 3.4 MB
  id=fix-auth-bug-20260318-a1b2
  status=running
```

**Check progress:**
```
user> /outpost-status fix-auth-bug-20260318-a1b2

  $ outpost status fix-auth-bug-20260318-a1b2

  id=fix-auth-bug-20260318-a1b2
  status=complete
  patch_ready=true

  Run is complete. Pick up with: /outpost-pickup fix-auth-bug-20260318-a1b2
```

**Pick up results:**
```
user> /outpost-pickup fix-auth-bug-20260318-a1b2

  $ outpost pickup fix-auth-bug-20260318-a1b2

  downloading patch...
  patch=.outpost/patches/fix-auth-bug-20260318-a1b2.patch

   src/auth.go | 15 +++++++++------
   1 file changed, 9 insertions(+), 6 deletions(-)
```

**Drop a stuck run:**
```
user> /outpost-drop fix-auth-bug-20260318-a1b2

  $ outpost drop fix-auth-bug-20260318-a1b2

  id=fix-auth-bug-20260318-a1b2
  status=dropped

  Run dropped.
```

## System Diagram

```
Claude Code (laptop)                         Outpost Server (remote)
========================                     ========================

/outpost skill                               gRPC server (:7600, TLS)
  |                                            |
  | writes plan to file                        | OutpostService
  v                                            |
outpost handoff --plan ...  --- gRPC/TLS ---> Handoff (client stream)
                                               | writes chunks to disk
                                               | runner.Extract()
                                               | runner.Spawn()
                                               v
outpost status <id>         --- gRPC/TLS ---> GetRun / ListRuns (unary)
                                               |
outpost status --follow <id> -- gRPC/TLS ---> TailLogs (server stream)
                                               | polls output.log
                                               v
outpost pickup <id>         --- gRPC/TLS ---> GetRun (unary)
                            --- gRPC/TLS ---> DownloadPatch (server stream)
                            --- gRPC/TLS ---> CleanupRun (unary)
                            saves to .outpost/patches/<id>.patch
                                               |
outpost drop <id>           --- gRPC/TLS ---> DropRun (unary)
                                               | runner.Stop()
                                               | cleanup run dir
                                               v
```

Skills (`~/.claude/commands/*.md`) invoke the `outpost` binary. The binary handles gRPC transport, TLS, and auth. Skills never touch the network directly.

## Proto Definition

`proto/outpost/v1/outpost.proto`:

```protobuf
syntax = "proto3";

package outpost.v1;

option go_package = "github.com/wesgrimes/outpost/gen/outpost/v1;outpostv1";

import "google/protobuf/timestamp.proto";

// ---------- Enums ----------

enum RunStatus {
  RUN_STATUS_UNSPECIFIED = 0;
  RUN_STATUS_PENDING     = 1;
  RUN_STATUS_RUNNING     = 2;
  RUN_STATUS_COMPLETE    = 3;
  RUN_STATUS_FAILED      = 4;
  RUN_STATUS_DROPPED     = 5;
}

enum RunMode {
  RUN_MODE_UNSPECIFIED  = 0;
  RUN_MODE_INTERACTIVE  = 1;
  RUN_MODE_HEADLESS     = 2;
}

// ---------- Core message ----------

message Run {
  string                    id          = 1;
  string                    name        = 2;
  RunMode                   mode        = 3;
  RunStatus                 status      = 4;
  string                    base_sha    = 5;
  string                    final_sha   = 6;
  google.protobuf.Timestamp created_at  = 7;
  google.protobuf.Timestamp finished_at = 8;
  string                    attach      = 9;
  string                    log_tail    = 10;
  bool                      patch_ready = 11;
  string                    branch      = 12;
  int32                     max_turns   = 13;
  string                    subdir      = 14;
}

// ---------- Unary ----------

message GetRunRequest   { string id = 1; }
message ListRunsRequest {}
message ListRunsResponse { repeated Run runs = 1; }
message DropRunRequest  { string id = 1; }
message DropRunResponse { string id = 1; }
message CleanupRunRequest  { string id = 1; }
message CleanupRunResponse { string id = 1; string status = 2; }
message HealthCheckRequest {}
message HealthCheckResponse { string status = 1; }

// ---------- Client streaming: Handoff ----------

message HandoffChunk {
  oneof payload {
    HandoffMetadata metadata = 1;
    bytes           data     = 2;  // 64 KiB archive chunks
  }
}

message HandoffMetadata {
  string  plan      = 1;
  RunMode mode      = 2;
  string  name      = 3;
  string  branch    = 4;
  int32   max_turns = 5;
  string  subdir    = 6;
}

message HandoffResponse {
  string    id     = 1;
  RunStatus status = 2;
  string    attach = 3;
}

// ---------- Server streaming: TailLogs ----------

message TailLogsRequest {
  string id     = 1;
  bool   follow = 2;  // stream until run completes
}

message LogEntry {
  string line = 1;
}

// ---------- Server streaming: DownloadPatch ----------

message PatchRequest { string id = 1; }
message DataChunk    { bytes data = 1; }

// ---------- Bidi streaming: Attach ----------

message SessionInput  { bytes data = 1; }
message SessionOutput { bytes data = 1; }

// ---------- Service ----------

service OutpostService {
  // Unary
  rpc HealthCheck(HealthCheckRequest)  returns (HealthCheckResponse);
  rpc GetRun(GetRunRequest)            returns (Run);
  rpc ListRuns(ListRunsRequest)        returns (ListRunsResponse);
  rpc DropRun(DropRunRequest)           returns (DropRunResponse);
  rpc CleanupRun(CleanupRunRequest)    returns (CleanupRunResponse);

  // Client streaming
  rpc Handoff(stream HandoffChunk)     returns (HandoffResponse);

  // Server streaming
  rpc TailLogs(TailLogsRequest)        returns (stream LogEntry);
  rpc DownloadPatch(PatchRequest)      returns (stream DataChunk);

  // Bidi streaming
  rpc Attach(stream SessionInput)      returns (stream SessionOutput);
}
```

## Design Decisions

- **`HandoffChunk` uses `oneof`**: first message carries metadata, subsequent messages carry 64 KiB archive chunks. Replaces multipart form. Neither side buffers the full archive in memory.
- **Handoff streaming error contract**: first message MUST be metadata (server rejects data-first with `InvalidArgument`). Validation failures (bad mode, empty plan) return error immediately. Client disconnect mid-stream triggers cleanup of the partial run directory via defer.
- **Proto3 zero-value handling**: `max_turns` defaults to `0` if unset. Server treats `0` as "use default (50)." Same pattern for any future int32 fields with meaningful defaults.
- **`TailLogs` with `follow=true`**: replaces polling. Server watches the log file and pushes new lines instantly.
- **`TailLogs` with `follow=false`**: streams the entire log from line 0, then closes. Replaces the old `GET /runs/{id}/log` endpoint.
- **TailLogs sends raw lines** (no ANSI stripping). `GetRun.log_tail` continues to strip ANSI for the summary view. Clients that want clean output strip on their end.
- **`Attach`**: PTY relay over bidi stream (Phase 3, future work).
- **TLS by default**: `setup` generates a self-signed CA + server cert/key. Server cert includes SANs auto-detected from system hostname + all local IPs. Client stores CA cert alongside credentials. Plaintext fallback via empty TLS config fields (local dev only).
- **Client address format**: `login` stores `host:port` (no scheme). This is the gRPC dial target.
- **Store stays independent of proto**: `store.Run` keeps its own Go types (string-based Status/Mode). Conversion helpers translate to/from proto types. Store never imports `gen/`.
- **Skills invoke the CLI binary**: no more curl. Skills handle orchestration (plan compilation, argument parsing), the binary handles transport. One client implementation, not two.
- **Capacity errors**: `ResourceExhausted` carries active run details in the error message string (JSON-encoded). Preserves current UX where the client shows which runs are blocking.
- **`GetRun` refreshes `log_tail`**: for running sessions, re-reads last 80 lines from disk before responding (existing behavior).
- **`DropRun` stops and discards**: stops the session via `runner.Stop()`, removes the run directory, deletes from store. No partial patch capture. If you wanted the work, you should have picked it up first.
- **`runner.Stop()` handles both modes**: tmux (`tmux kill-session -t ID`) and headless (SIGTERM to tracked PID). Current code only handles tmux; headless runs can't be stopped. Fixed in this rewrite.

## Package Structure

```
outpost/
  main.go
  Makefile
  .golangci.yml
  proto/
    outpost/v1/outpost.proto
    buf.yaml
    buf.gen.yaml
  gen/outpost/v1/              # generated, committed
  internal/
    grpcserver/
      server.go                # gRPC setup, TLS, auth interceptors, graceful shutdown
      handoff.go               # client-streaming Handoff
      runs.go                  # unary: GetRun, ListRuns, DropRun, CleanupRun, HealthCheck
      logs.go                  # server-streaming TailLogs
      patch.go                 # server-streaming DownloadPatch
    grpcclient/
      client.go                # New(), Load(), Close(), all RPC methods
    runner/
      runner.go                # Spawn(), Stop() (tmux + headless)
      archive.go               # Extract()
      patch.go                 # GeneratePatch()
    store/
      store.go                 # in-memory store (unchanged)
      proto.go                 # RunToProto(), ProtoToRun(), StatusToProto(), etc.
    config/
      config.go                # ServerConfig with TLS fields
  cmd/
    setup.go                   # TLS cert generation, updated summary
    serve.go                   # gRPC server with signal handling
    runs.go                    # server-local: grpcclient to localhost
    login.go                   # host:port + --ca-cert
    handoff.go                 # grpcclient streaming upload
    status.go                  # grpcclient GetRun/ListRuns
    pickup.go                  # grpcclient DownloadPatch + git worktree
    drop.go                    # grpcclient DropRun
```

## Touchpoints

Every artifact in the final system:

| Layer | File | Role |
|-------|------|------|
| Proto | `proto/outpost/v1/outpost.proto` | Service + message definitions |
| Proto | `proto/buf.yaml` | Buf module config |
| Proto | `proto/buf.gen.yaml` | Code generation config (protoc-gen-go, protoc-gen-go-grpc) |
| Generated | `gen/outpost/v1/*.pb.go` | Generated protobuf types |
| Generated | `gen/outpost/v1/*_grpc.pb.go` | Generated gRPC stubs |
| Server | `internal/grpcserver/server.go` | Listener, TLS, interceptors, `GracefulStop()` |
| Server | `internal/grpcserver/handoff.go` | Receive metadata, stream chunks to disk, extract, spawn |
| Server | `internal/grpcserver/runs.go` | Unary RPCs: GetRun, ListRuns, DropRun, CleanupRun, HealthCheck |
| Server | `internal/grpcserver/logs.go` | TailLogs: offset tracking, file polling, follow mode, context cancellation |
| Server | `internal/grpcserver/patch.go` | DownloadPatch: 64 KiB chunked file read |
| Client | `internal/grpcclient/client.go` | `New(target, token, opts)`, `Load()`, `Close()`, all RPC methods |
| Runner | `internal/runner/runner.go` | `Spawn()` (tmux + headless), `Stop()` (SIGTERM for headless, tmux kill-session for interactive) |
| Runner | `internal/runner/archive.go` | `Extract()` tar.gz to repo dir, git init, base SHA |
| Runner | `internal/runner/patch.go` | `GeneratePatch()` via git diff |
| Store | `internal/store/store.go` | Thread-safe in-memory Run store |
| Store | `internal/store/proto.go` | `RunToProto()`, `ProtoToRun()` (nil-safe timestamp handling) |
| Config | `internal/config/config.go` | `ServerConfig` with `TLSCert`, `TLSKey`, `TLSCA` path fields |
| CLI | `cmd/setup.go` | Dir creation, TLS cert generation, config generation, systemd |
| CLI | `cmd/serve.go` | Load config, create store, start gRPC server, signal handler |
| CLI | `cmd/login.go` | Parse `host:port token [--ca-cert path]`, write credential files, verify connection |
| CLI | `cmd/handoff.go` | Create archive, open gRPC stream, send metadata + chunks, print result |
| CLI | `cmd/status.go` | `outpost status [id]`: list table or detail with log tail |
| CLI | `cmd/pickup.go` | Download patch via gRPC stream to `.outpost/patches/`, cleanup remote |
| CLI | `cmd/drop.go` | gRPC DropRun, stop session and discard |
| CLI | `cmd/runs.go` | Server-local: dial `localhost:port` with server token |
| Skill | `~/.claude/commands/outpost.md` | Compile plan, call `outpost handoff` |
| Skill | `~/.claude/commands/outpost-status.md` | Call `outpost status` |
| Skill | `~/.claude/commands/outpost-pickup.md` | Call `outpost pickup` |
| Skill | `~/.claude/commands/outpost-drop.md` | Call `outpost drop` |
| Build | `Makefile` | `proto`, `build`, `check`, `fmt` targets |
| Build | `.golangci.yml` | Exclude `gen/` from linting |

## Updated Skills

The skills become thin wrappers. The `outpost` binary handles all transport, auth, and TLS.

### `/outpost` (handoff)

```markdown
Hand off the current implementation plan to a remote Outpost server for execution.

Usage: /outpost [headless|interactive] [--name N] [--branch B] [--max-turns N]

Arguments: $ARGUMENTS

Follow these steps:

1. **Compile the implementation plan.** Summarize everything from this
   conversation into a self-contained markdown document. The remote session
   has ZERO context from this conversation, so include:
   - Full problem description and requirements
   - Specific files to modify and how
   - Any relevant code snippets or patterns
   - Acceptance criteria
   - Testing instructions
   Write the plan to `/tmp/outpost-plan.md` using a heredoc.

2. **Parse arguments.** Defaults: mode=interactive, name=empty, branch=empty,
   max_turns=50. Parse from: $ARGUMENTS

3. **Submit to Outpost** using the outpost CLI:
   ```bash
   outpost handoff --plan /tmp/outpost-plan.md \
     --mode "${MODE:-interactive}" \
     --name "${NAME}" \
     --branch "${BRANCH}" \
     --max-turns "${MAX_TURNS:-50}"
   ```

4. **Clean up:** `rm -f /tmp/outpost-plan.md`

5. **Report results.** Parse the key=value output:
   - Run ID (from `id=...`)
   - Attach command (from `attach=...`, for interactive mode)
   - Check status: `/outpost-status <id>`
   - Pick up when done: `/outpost-pickup <id>`
```

### `/outpost-status`

```markdown
Check the status of Outpost runs.

Usage: /outpost-status [run_id]

Arguments: $ARGUMENTS

Run:
```bash
outpost status $ARGUMENTS
```

Display the output. Then suggest next steps:
- If status is "running": suggest waiting or attaching via SSH
- If status is "complete" and patch_ready is true: suggest `/outpost-pickup <id>`
- If status is "failed": show the log tail and suggest `/outpost-drop <id>`
```

### `/outpost-pickup`

```markdown
Pick up a completed Outpost run.

Usage: /outpost-pickup <run_id>

Arguments: $ARGUMENTS

A run_id is required. If not provided, tell the user to run
`/outpost-status` first.

Run:
```bash
outpost pickup $ARGUMENTS
```

Display the output (patch path, files changed).
The patch is now on disk. Ask the user how they want to apply it:
- Apply to current branch: `git apply <patch>`
- Create a new branch first: `git checkout -b <branch> && git apply <patch>`
- Review first: `cat <patch>` or open in editor

Do NOT apply the patch automatically. Let the user decide.
```

### `/outpost-drop`

```markdown
Drop an Outpost run. Stops the session and discards all work.

Usage: /outpost-drop <run_id>

Arguments: $ARGUMENTS

A run_id is required. If not provided, tell the user to run
`/outpost-status` first.

Run:
```bash
outpost drop $ARGUMENTS
```

Display the output. Confirm the run has been dropped.
```

## CLI Output Contract

All client commands follow these rules so skills and scripts can reliably parse output:

- **stdout**: machine-readable data only. Key=value pairs (one per line), tables, or streamed content.
- **stderr**: human-readable status messages, progress, errors. Prefixed with `error:` on failure.
- **Exit codes**: 0 = success, 1 = error (bad args, network, server error), 2 = run not found/not ready.

Per-command output:

| Command | stdout | stderr |
|---------|--------|--------|
| `handoff` | `id=...\nstatus=...\nattach=...` | Upload progress: `uploading... 2.1 MB / 3.4 MB` |
| `status` (list) | Tab-separated table: `ID\tSTATUS\tMODE\tCREATED` | Nothing |
| `status <id>` | Key=value pairs, then `\n--- log tail ---\n...` | Nothing |
| `status --follow <id>` | Raw log lines as they arrive (one per line) | `following run <id>...` on start, `run completed` on close |
| `pickup` | `patch=<path>\n\n<git diff --stat of patch>` | Progress: `downloading patch...` |
| `drop` | `id=...\nstatus=dropped` | Nothing |
| `login` | Nothing | `Verifying connection... healthy\nCredentials saved.` |

The skills parse stdout with `grep '^key=' | cut -d= -f2-`. Status messages on stderr are displayed to the user but not parsed.

## Starting Point

Delete everything under `internal/`, `cmd/`, and `main.go`. Keep these files untouched:

- `PLAN.md` (this file)
- `Makefile` (will be modified)
- `.golangci.yml` (will be modified)
- `go.mod` / `go.sum` (will be updated via `go get`)
- `.gitignore`

## Go Dependencies

```bash
go get google.golang.org/grpc
go get google.golang.org/protobuf
go get gopkg.in/yaml.v3
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## Build Order

### Phase 1: Proto + Tooling

Install buf: `go install github.com/bufbuild/buf/cmd/buf@latest`

Create `proto/outpost/v1/outpost.proto` (contents in Proto Definition section above).

`proto/buf.yaml`:
```yaml
version: v2
modules:
  - path: outpost/v1
lint:
  use:
    - STANDARD
breaking:
  use:
    - FILE
```

`proto/buf.gen.yaml`:
```yaml
version: v2
plugins:
  - local: protoc-gen-go
    out: ../gen
    opt: paths=source_relative
  - local: protoc-gen-go-grpc
    out: ../gen
    opt: paths=source_relative
```

Updated `Makefile`:
```makefile
GOLANGCI_LINT := /opt/homebrew/bin/golangci-lint

.PHONY: all build build-linux proto lint fmt vet check clean

all: check build

build:
	go build -o bin/outpost .

build-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/outpost-linux .

proto:
	cd proto && buf lint && buf generate

lint:
	$(GOLANGCI_LINT) run ./...

fmt:
	$(GOLANGCI_LINT) fmt ./...

vet:
	go vet ./...

check: vet lint

clean:
	rm -rf bin/
```

Add to `.golangci.yml` under `issues`:
```yaml
  exclude-dirs:
    - gen
```

Generate code: `make proto`. Commit `gen/`. Run `make check`.

### Phase 2: Server + Core

**Config:**
- Add `TLSCert`, `TLSKey`, `TLSCA` string fields to `ServerConfig`
- Keep env override support

**Store:**
- `store.go` unchanged (same in-memory store)
- New `proto.go` with conversion helpers:
  - `RunToProto(*Run) *outpostv1.Run` (handles `FinishedAt *time.Time` to `*timestamppb.Timestamp`, nil-safe)
  - `StatusToProto(Status) outpostv1.RunStatus`
  - `ModeToProto(string) outpostv1.RunMode`
  - Reverse helpers for each

**Runner:**

Package-level process registry for headless PID tracking:
```go
var (
    processes   = make(map[string]*os.Process)
    processesMu sync.Mutex
)
```

`Spawn()` stores `cmd.Process` in the map after `cmd.Start()` for headless runs. The monitoring goroutine deletes the entry on exit.

`Stop(runID, mode string)`:
- Interactive: `exec.Command("tmux", "kill-session", "-t", runID).Run()`
- Headless: look up PID in map, send `SIGTERM`, wait up to 5s, `SIGKILL` if still alive
- No-op if process already exited (not in map)

`Extract()`, `GeneratePatch()` unchanged in logic.

**gRPC server (`internal/grpcserver/`):**

`server.go`:
- `New(cfg, store)` creates `grpc.Server` with TLS creds (if configured) and auth interceptors
- Unary interceptor: check `authorization` metadata, exempt `HealthCheck`
- Stream interceptor: same check, same exemption
- `ListenAndServe()`: listen on `:port`, serve, block
- `GracefulStop()` on SIGINT/SIGTERM

`runs.go` (unary RPCs):
- `GetRun`: look up in store, refresh `log_tail` from disk if status is running, convert to proto
- `ListRuns`: return all runs sorted by created_at desc
- `DropRun`: call `runner.Stop()`, remove run directory, delete from store, return confirmation
- `CleanupRun`: remove run directory from disk, delete from store
- `HealthCheck`: return `{status: "ok"}`

`handoff.go` (client streaming):
- Receive first message, assert it is metadata (reject with `InvalidArgument` if not)
- Validate metadata (plan non-empty, mode valid). Return error immediately on failure.
- Create run directory structure
- Open archive file for writing
- Receive subsequent data chunks, write to archive file
- On client disconnect mid-stream: `defer` cleanup of partial run directory
- After all chunks received: call `runner.Extract()`, create Run in store, call `runner.Spawn()`
- If spawn fails: update store to failed, return error
- Return `HandoffResponse` with id, status=running, attach command

`logs.go` (server streaming):
- Look up run in store (return `NotFound` if missing)
- Open log file. If file doesn't exist:
  - `follow=true`: poll until it appears (run may be pending), respect context cancellation
  - `follow=false`: return `NotFound`
- Read from byte offset 0, send each line as `LogEntry`
- Track byte offset to avoid re-sending
- `follow=true`: poll for new content (~500ms), check run status in store to know when to close
- `follow=false`: read current content and close
- Respect `ctx.Done()` for client disconnection

`patch.go` (server streaming):
- Look up run, verify patch file exists
- Read file in 64 KiB chunks, send each as `DataChunk`

Test all RPCs with `grpcurl`.

### Phase 3: Client + CLI + Skills

**grpcclient (`internal/grpcclient/`):**
- `New(target, token string, opts ...grpc.DialOption)`: dial with TLS or insecure, store connection
- `Load()`: read `~/.outpost-url`, `~/.outpost-token`, `~/.outpost-ca.pem`, call `New()` with appropriate TLS config
- `Close()`: close the underlying connection
- Methods: `Handoff(ctx, planPath, archivePath string, meta HandoffMeta) (*HandoffResult, error)` (streams chunks internally), `GetRun`, `ListRuns`, `DropRun`, `CleanupRun`, `HealthCheck`, `TailLogs(ctx, id, follow) (stream, error)`, `DownloadPatch(ctx, id, destPath) error`

**CLI commands:**
- `cmd/setup.go`: generate TLS certs, write config, print summary. TLS generation approach:
  1. Generate ECDSA P-256 CA key + self-signed CA cert (10yr expiry, `IsCA: true`, `KeyUsageCertSign`)
  2. Generate ECDSA P-256 server key + cert signed by CA (10yr expiry, `ExtKeyUsageServerAuth`)
  3. Server cert SANs: `os.Hostname()`, `"localhost"`, all non-loopback IPs from `net.Interfaces()`
  4. Write PEM files to `~/.outpost/tls/` (ca.pem, ca-key.pem, server.pem, server-key.pem)
  5. Set `tls_cert`, `tls_key`, `tls_ca` paths in config.yaml
- `cmd/login.go`: accept `outpost login <host:port> <token> [--ca-cert path]`. Write `~/.outpost-url`, `~/.outpost-token`, copy CA cert to `~/.outpost-ca.pem` if provided. Call `HealthCheck` to verify connection.
- `cmd/serve.go`: load config, create store, create gRPC server, set up signal handler for graceful shutdown.
- `cmd/handoff.go`: create archive (same `git ls-files | tar czf`), call `grpcclient.Load()`, call `Handoff()`. Print upload progress to stderr (bytes sent / total), result key=value pairs to stdout.
- `cmd/status.go`: `outpost status` lists runs, `outpost status <id>` shows detail, `outpost status --follow <id>` streams logs via TailLogs until run completes.
- `cmd/pickup.go`: call `GetRun` to verify, call `DownloadPatch` to `.outpost/patches/<id>.patch`, print patch path and `git diff --stat` of the patch contents, call `CleanupRun`. No git operations. The user or agent decides how to apply.
- `cmd/drop.go`: call `DropRun`, print confirmation.
- `cmd/runs.go`: server-local command. Dial `localhost:port` using server config token. Call `ListRuns`/`GetRun`. Disk fallback if server unreachable.

**Skills** (`~/.claude/commands/`):
- Rewrite all four skill files as shown in "Updated Skills" section above.
- Skills invoke `outpost` binary, never use curl or access the network directly.

End-to-end test: `setup` -> `login` -> `handoff` -> `status` -> `pickup`.

### Phase 4 (Future): Attach

- Bidi PTY relay implementation in `grpcserver/attach.go`
- New `outpost attach <run_id>` command
- New `/outpost-attach` skill
- Replaces SSH + tmux entirely

## Auth

Token travels in gRPC metadata (`authorization: Bearer <token>`), checked by unary + stream interceptors. `HealthCheck` is exempted. Same constant-time compare as today.

## TLS

- `setup` generates a self-signed CA, server cert, and server key using Go's `crypto/x509` (no openssl dependency). Server cert SANs include system hostname + all non-loopback IP addresses. All written to `~/.outpost/tls/`.
- Config fields: `tls_cert`, `tls_key`, `tls_ca` (file paths). If all empty, server falls back to insecure (local dev only).
- `login` accepts an optional `--ca-cert` path, copies it to `~/.outpost-ca.pem`. Client uses it for `credentials.NewTLS`.
- Bearer token is always required regardless of TLS. TLS protects the token in transit; the token provides authz.

## Error Mapping

| Condition | gRPC Code |
|---|---|
| Bad request (missing plan, invalid mode) | `InvalidArgument` |
| Missing or wrong token | `Unauthenticated` |
| Run not found | `NotFound` |
| At capacity (active runs list in message) | `ResourceExhausted` |
| Server error | `Internal` |

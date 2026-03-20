# Outpost UI

> The single reference for Outpost's user interface: design system, CLI command output, and (eventually) TUI.

---

## Design System

### Principles

- **One screenful.** Key vitals for any command should fit without scrolling.
- **Greppable.** Log lines and table rows are parseable by standard tools.
- **Copy-pasteable.** Every suggested command works if pasted directly.
- **Progressive disclosure.** Dashboard shows summary; drill into a run ID for detail.
- **Destructive actions confirm.** Show what's at risk and offer the safe alternative as a selection option.

### Brand Elements

- **Header**: `OUTPOST v0.1.0` (amber, uppercase) opens every structured output block, followed by the command context on the same line. The name *is* the brand mark, no symbol prefix needed. Mirrors Vite's `VITE v6.2.2` pattern.
- **Run IDs**: Short form `op-XXXX` (4 hex chars), always amber. These are the primary anchor users type and grep for.
- **Version**: Always shown next to the brand name in dim/green text.

### Color Semantics

Each color has exactly one meaning. Do not cross-purpose them.

| Color   | ANSI    | Means                        | Used for                               | Never use it for       |
|---------|---------|------------------------------|----------------------------------------|------------------------|
| Amber   | Yellow  | Brand / identity / IDs       | `OUTPOST`, run IDs, column headers     | Status or content      |
| Green   | Green   | Success / complete           | `✓`, `done`, GET verbs, exit 0         | In-progress states     |
| Cyan    | Cyan    | Active / in-progress         | `⠸ running`, spinner, active turn      | Completed states       |
| Orange  | Yellow  | Warning / caution            | `⚠`, disk low, dirty tree              | Errors                 |
| Red     | Red     | Error / destructive / failed | `✗`, `failed`, DELETE verbs            | Warnings               |
| Purple  | Magenta | Claude / AI activity         | `claude` log prefix, waiting state     | Non-AI content         |
| Blue    | Blue    | Actionable commands / hints  | Next-step suggestions, copy-paste cmds | Labels or status       |
| White   | White   | Primary content / values     | File paths, counts, durations          | Chrome or decoration   |
| Dim     | Gray    | Chrome / labels / secondary  | Timestamps, separators, descriptions   | Primary content        |

### Status Symbols

Symbols always pair with color so output is readable without color support:

| Symbol | Color  | State       |
|--------|--------|-------------|
| `✓`    | Green  | Complete    |
| `⠸`    | Cyan   | Running     |
| `●`    | Green  | Done (dot)  |
| `◉`    | Purple | Waiting     |
| `⚠`    | Orange | Warning     |
| `✗`    | Red    | Failed/Error|

The `OUTPOST` brand name is pure ASCII. All other symbols (`✓` `✗` `⠸` `●` `◉` `⚠` `│` `└` `├`) require UTF-8, which is the baseline requirement. No ASCII fallback mode.

`NO_COLOR` env var and `--no-color` flag strip ANSI escape codes only; symbols and box-drawing characters remain:
- Symbols carry meaning independently of color (`✓` vs `✗` vs `⚠`)
- Tables remain readable without color (alignment is structural)
- Log lines are tab-separated for machine parsing

### Structural Patterns

- **Checklist blocks**: `OUTPOST v{version}` header line, vertical line `│` on the left, `✓`/`✗` prefixed steps, `└` to close
- **Tables**: Column headers in amber, aligned with spaces (no box drawing for data rows)
- **Next-step hints**: Appear after `└` closer, dimmed label + blue command
- **Error recovery**: Error on one line, fix command(s) below as copy-pasteable blue text
- **Confirmation prompts**: `● Selected` in amber, unselected options in dim
- **Log lines**: Fixed-width prefix `run-id timestamp source content`, tab-separated for machine parsing

### Error Philosophy

- **Stop on first failure.** No partial success ambiguity. Checklist ends at the `✗` line.
- **Surface the why.** Show Claude's own explanation when a run fails.
- **Always offer a fix.** Every error includes a copy-pasteable recovery command.
- **Warnings are specific.** "6 turns of work" not "are you sure?". Show actual dirty files, not just a count.

### Output Modes

| Flag         | Effect                                             |
|--------------|----------------------------------------------------|
| `--json`     | Machine-readable JSON, replaces all human output   |
| `--quiet`    | Essential values only, no chrome                   |
| `--no-color` | Strip ANSI codes, keep symbols and structure       |
| `--force`    | Skip confirmation prompts (for scripting)          |
| `--follow`   | Stream updates in real-time (status, logs)         |
| `--tail`     | Alias for `--follow` on `logs` command             |

---

## CLI Commands

### Output Primitives

These are the building blocks; implement them first:

1. **Checklist renderer** — `OUTPOST v{version}` header, `│` left border, `✓`/`✗`/`⠸`/`⚠` prefixed lines, `└` closer
2. **Table renderer** — Amber headers, aligned columns, footer summary with `·` separators
3. **Progress bar** — `████░░░░` style with percentage, used for streaming
4. **Confirmation prompt** — `●` selected / dim unselected, keyboard navigation
5. **Next-step hints** — Dim label + blue command, appears after `└` closer
6. **Log line formatter** — `run-id timestamp source content` with consistent column widths

### Server-Side Commands

#### `outpost server setup <host>`

Provisions a remote server via SSH. Shows a checklist of provisioning steps.

```
$ outpost server setup myserver.grimes.pro

  OUTPOST v0.1.0  Provisioning myserver.grimes.pro

  │  ✓ SSH connection established
  │  ✓ System dependencies installed
  │  ✓ Claude Code CLI verified (v1.0.23)
  │  ✓ TLS certificates generated
  │  ✓ gRPC daemon configured (port 9090)
  │  ✓ systemd service installed
  │  ✓ Firewall rules applied
  │
  └  Ready. Token: op_tk_7f3a...c91d

  Next:
    outpost login myserver.grimes.pro:9090 op_tk_7f3a...c91d
```

**Failure mid-flight** — stops the checklist at the failure point, no partial success ambiguity:

```
$ outpost server setup 192.168.1.50

  OUTPOST v0.1.0  Provisioning 192.168.1.50

  │  ✓ SSH connection established
  │  ✓ System dependencies installed
  │  ✗ Claude Code CLI not found
  │
  │  Error: `claude` not in PATH on remote host.
  │  Install it first: npm install -g @anthropic/claude-code
  │  Then re-run: outpost server setup 192.168.1.50
  └
```

**Design notes:**
- Checklist stops immediately on first failure
- Fix command is always copy-pasteable
- Token output is visually isolated at the bottom of success output
- Next-step hint follows Vite's pattern

#### `outpost serve`

Starts the gRPC daemon. Mirrors Vite's dev server "ready" block.

```
$ outpost serve

  OUTPOST v0.1.0  listening on 0.0.0.0:9090 (TLS)

  ➜  Runs:     /var/lib/outpost/runs
  ➜  Disk:     42.1 GB available
  ➜  Workers:  4 max concurrent
  ➜  PID:      18432
```

**Design notes:**
- Brand word + version + listen address on one line
- Key vitals listed below, everything you'd grep for in one screenful
- No log output after this unless a request comes in

#### `outpost server doctor`

Health check with two-column vitals and a checklist.

**Healthy:**

```
$ outpost server doctor

  OUTPOST v0.1.0  Server Health myserver.grimes.pro

  │  Outpost     v0.1.0           Claude Code  v1.0.23
  │  Uptime      3d 14h 22m      Active runs  2
  │  Disk        42.1 / 100 GB   CPU          12%
  │  Memory      3.2 / 8 GB     TLS cert     valid 89d
  │
  │  ✓ gRPC reachable          ✓ systemd active
  │  ✓ Claude Code auth valid  ✓ disk space OK
  └
```

**Degraded** — warnings and errors surface fix commands:

```
$ outpost server doctor

  OUTPOST v0.1.0  Server Health myserver.grimes.pro

  │  Outpost     v0.1.0           Claude Code  v1.0.23
  │  Uptime      3d 14h 22m      Active runs  2
  │  Disk        91.4 / 100 GB   CPU          12%
  │  Memory      3.2 / 8 GB     TLS cert     expired 3d ago
  │
  │  ✓ gRPC reachable          ✓ systemd active
  │  ✓ Claude Code auth valid  ⚠ disk space low (< 10 GB free)
  │                              ✗ TLS cert expired
  │
  │  Fix TLS: outpost server setup myserver.grimes.pro --renew-certs
  └
```

**Design notes:**
- Three severity tiers: green ✓, orange ⚠, red ✗
- Fix commands only appear when something is broken
- Two-column layout for vitals keeps it scannable
- Degraded values change color inline (disk turns orange, cert turns red)

### Client-Side Commands

#### `outpost login <host:port> <token>`

Saves credentials. Handles TOFU (Trust On First Use) for self-signed certs.

```
$ outpost login myserver.grimes.pro:9090 op_tk_7f3a...c91d

  OUTPOST v0.1.0  Login

  │  ⚠ Self-signed certificate detected.
  │    Fingerprint: SHA256:xK3m...9f2a
  │
  │  Trust this certificate? (TOFU)
  │  ● Yes, pin it
  │    No, abort
  └

  ✓ Credentials saved to ~/.config/outpost/credentials.json
  ✓ Certificate pinned.
```

#### `outpost doctor`

Client-side health check.

```
$ outpost doctor

  OUTPOST v0.1.0  Client Health

  │  ✓ Outpost CLI v0.1.0
  │  ✓ Credentials configured (myserver.grimes.pro:9090)
  │  ✓ Server reachable (14ms)
  │  ✓ Token valid
  │  ✓ Git repo detected (iheartjane/monorepo)
  └
```

### Core Workflow Commands

#### `outpost handoff`

Archives repo + session, streams to server, starts a run.

**In-progress (streaming):**

```
$ outpost handoff --session-id abc123 --mode headless --max-turns 10 --branch wes/premium-healthchecks

  OUTPOST v0.1.0  Handoff → myserver.grimes.pro

  │  ✓ Archiving repo (1,247 files, 18.3 MB)
  │  ✓ Bundling session abc123
  │  ⠸ Streaming to server...  ████████████████░░░░ 82%
```

**Complete:**

```
$ outpost handoff --session-id abc123 --mode headless --max-turns 10 --branch wes/premium-healthchecks

  OUTPOST v0.1.0  Handoff → myserver.grimes.pro

  │  ✓ Archiving repo (1,247 files, 18.3 MB)
  │  ✓ Bundling session abc123
  │  ✓ Streamed (18.3 MB in 2.1s)
  │  ✓ Run started
  │
  └  Run op-7f3a │ headless │ max 10 turns │ branch wes/premium-healthchecks

  Watch:  outpost status op-7f3a --follow
  Logs:   outpost logs op-7f3a --tail
```

**Dirty working tree:**

```
$ outpost handoff --session-id abc123 --mode headless

  OUTPOST v0.1.0  Handoff

  │  ⚠ Working tree has uncommitted changes:
  │      M  app/controllers/stores_controller.rb
  │      A  app/services/new_thing.rb
  │
  │  Include uncommitted changes in handoff?
  │  ● Yes, include working tree as-is
  │    No, only send committed files
  │    Cancel
  └
```

**No server configured:**

```
$ outpost handoff --session-id abc123

  ✗ No server configured.

  Run outpost login <host:port> <token> to connect to a server.
  Need a server? outpost server setup <host>
```

**Design notes:**
- Run ID is the anchor for all downstream commands, always shown prominently
- Short IDs (op-XXXX) are easier to type than UUIDs
- Summary line after `└` packs all config into a scannable bar separated by `│`
- Progress bar for streaming (only long-running step)
- Dirty tree warning shows actual files, not just a count
- Error state is one line + recovery paths (two options because "no server" is ambiguous)

#### `outpost status`

**Dashboard (no args), all runs:**

```
$ outpost status

  OUTPOST v0.1.0  Runs on myserver.grimes.pro

  ID        Branch                          Mode      Turns   Status         Age
  op-7f3a   wes/premium-healthchecks        headless  4/10    ⠸ running      2m
  op-a1c2   wes/schwazze-onboarding         headless  10/10   ● done         18m
  op-d4e5   wes/menu-proxy-cache            interact  6/20    ◉ waiting      1h
  op-f8g9   fix/bloom-hydration-error       headless  3/5     ✗ failed       4h

  4 runs total  ·  1 running  ·  1 done  ·  1 waiting  ·  1 failed
```

**Single run detail:**

```
$ outpost status op-7f3a

  OUTPOST v0.1.0  Run op-7f3a │ ⠸ running

  │  Branch     wes/premium-healthchecks
  │  Mode       headless
  │  Turns      4 / 10
  │  Session    abc123
  │  Started    2m ago
  │
  │  ──── Activity ────
  │  10:42:17  Turn 1  Read CLAUDE.md, analyzed healthcheck requirements
  │  10:42:38  Turn 2  Created app/services/premium_healthcheck.rb
  │  10:43:01  Turn 3  Added Playwright smoke test suite
  │  10:43:24  Turn 4  Running tests...
  │            ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ awaiting turn 5
```

**Completed run:**

```
$ outpost status op-7f3a

  OUTPOST v0.1.0  Run op-7f3a │ ● done

  │  Branch     wes/premium-healthchecks
  │  Mode       headless
  │  Turns      7 / 10 (completed early)
  │  Duration   4m 12s
  │  Patch      +218 −14 across 6 files
  │
  └  outpost pickup op-7f3a
```

**Failed run:**

```
$ outpost status op-f8g9

  OUTPOST v0.1.0  Run op-f8g9 │ ✗ failed

  │  Branch     fix/bloom-hydration-error
  │  Mode       headless
  │  Turns      3 / 5
  │  Duration   2m 47s
  │
  │  Error: Claude Code exited with code 1
  │  Last activity: Tests failed — RSpec exit code 1, 2 examples, 1 failure
  │  Claude's note: "The hydration mismatch requires changes to the
  │    WordPress plugin loader which I don't have access to."
  │
  │  Partial patch available: +42 −8 across 2 files
  │
  ├  outpost pickup op-f8g9      grab partial work
  └  outpost logs op-f8g9        see full output
```

**Design notes:**
- Dashboard uses Docker ps-style table with aligned columns
- Status uses both color AND symbol for non-color terminal support
- `--follow` turns detail view into a live feed with per-turn summaries
- Failed runs surface Claude's own explanation of why it couldn't finish
- Completed runs show diffstat (+lines -lines) as a preview before pickup
- "waiting" status = interactive mode, needs human input
- Footer summary line uses `·` separator for quick counts

#### `outpost logs <id>`

Structured log lines with consistent prefix format: `run-id timestamp source content`.

```
$ outpost logs op-7f3a

op-7f3a 10:42:17 claude I'll start by reading the project requirements...
op-7f3a 10:42:17 tool   Read file: CLAUDE.md
op-7f3a 10:42:19 tool   Read file: app/admin/premium_healthchecks.rb
op-7f3a 10:42:22 claude I see the Active Admin panel. I'll create a service...
op-7f3a 10:42:38 tool   Write file: app/services/premium_healthcheck.rb
op-7f3a 10:42:39 tool   Write file: spec/services/premium_healthcheck_spec.rb
op-7f3a 10:43:01 claude Now adding Playwright smoke tests...
op-7f3a 10:43:01 tool   Write file: test/smoke/premium_healthcheck.spec.ts
op-7f3a 10:43:24 tool   bash: bundle exec rspec spec/services/ (exit 0)
op-7f3a 10:43:52 tool   bash: npx playwright test test/smoke/ (exit 0)
op-7f3a 10:44:29 claude All tests pass. Completing the session.
```

**Design notes:**
- Every line has the same prefix structure, greppable and parseable
- Two source types: `claude` (thinking/reasoning) and `tool` (file ops, bash commands)
- Tool lines further typed: `Read file`, `Write file`, `bash` with exit codes
- `--tail` flag follows in real-time (same format, just streaming)

#### `outpost pickup <id>`

Downloads patch + forked session from a completed run.

```
$ outpost pickup op-7f3a

  OUTPOST v0.1.0  Pickup op-7f3a

  │  ✓ Downloaded patch (4.2 KB)
  │  ✓ Downloaded session fork
  │
  │  Patch:  .outpost/patches/op-7f3a.patch
  │  Files:  +3 new  ~2 modified  -1 deleted
  │
  │    + app/services/premium_healthcheck.rb
  │    + spec/services/premium_healthcheck_spec.rb
  │    + test/smoke/premium_healthcheck.spec.ts
  │    ~ app/admin/premium_healthchecks.rb
  │    ~ config/routes.rb
  │    - app/services/legacy_healthcheck.rb
  │
  └  Apply: git apply .outpost/patches/op-7f3a.patch
```

**Design notes:**
- Git-style `+`/`~`/`-` prefixes with matching green/cyan/red colors
- Full file list so you see exactly what Claude did before applying
- Apply command is copy-pasteable at the bottom

#### `outpost drop <id>`

Stops and discards a run. Destructive action gets a confirmation prompt.

```
$ outpost drop op-d4e5

  OUTPOST v0.1.0  Drop op-d4e5

  │  ⚠ Run has 6 turns of work. Download patch first?
  │
  │  ● Drop without saving
  │    Pickup first, then drop
  │    Cancel
  └

  ✗ Run op-d4e5 stopped and discarded.
```

**Design notes:**
- Warning mentions concrete work at risk ("6 turns"), not generic "are you sure?"
- Offers the safe alternative (pickup first) as a selection option
- Can be skipped with `--force` flag for scripting

#### `outpost convert <id> <mode>`

Switches a running session between interactive and headless.

```
$ outpost convert op-7f3a interactive

  OUTPOST v0.1.0  Convert op-7f3a headless → interactive

  │  ✓ Paused at turn 4
  │  ✓ tmux session created
  │  ✓ Mode switched to interactive
  │
  └  Attach: ssh myserver.grimes.pro -t tmux attach -t op-7f3a
```

---

## TUI

*Coming soon.*

# UI Refresh Plan

> Implement the design system from [ui.md](ui.md) across all CLI commands.

## Current State

Two divergent output styles coexist:
- **Box-drawing** (`╭╮╰╯│`) in doctor, server setup, server doctor
- **Flat header + printField** in handoff, status, pickup, drop, convert

Neither matches ui.md. No color library. No spinners, progress bars, or interactive prompts. stdout/stderr usage is inconsistent. `tabwriter` is the only table renderer.

## Architecture

### Package: `internal/ui`

All rendering lives here. Commands call `ui.*` functions, never `fmt.Print*` directly for human output.

### stdout vs stderr Contract

| Channel | Content |
|---------|---------|
| stdout  | `--json` output, log lines (greppable data) |
| stderr  | All human chrome: headers, checklists, tables, hints, errors, progress |

### Output Mode Globals

Set once at startup from flags/env, consulted by all renderers:

- `ui.Color` (bool) - false when `NO_COLOR` set, `--no-color` flag, or !isatty(stderr)
- `ui.Quiet` (bool) - `--quiet` flag
- `ui.Force` (bool) - `--force` flag
- `ui.JSON` (bool) - `--json` flag (commands handle this themselves)
- `ui.IsTTY` (bool) - isatty(stderr), controls spinners/prompts

---

## Phases

### Phase 1: Foundation (`internal/ui` package)

Create the package with zero visual changes to existing commands.

**Files:**
- `internal/ui/color.go` - 9 named colors, `NO_COLOR` support, `--no-color` flag
- `internal/ui/term.go` - isatty detection, terminal width, ANSI-aware string width
- `internal/ui/writer.go` - `ui.Stderr` / `ui.Stdout` wrappers that respect color/quiet modes
- `internal/ui/symbol.go` - status symbol enum (`✓ ✗ ⠸ ● ◉ ⚠`) with paired colors
- `internal/ui/ui.go` - global config (Color, Quiet, Force, IsTTY), `Init()` function

**Tests:** color stripping, width calculation, isatty mock, symbol rendering with/without color.

### Phase 2: Brand Header + Status Symbols

Replace `printHeader()` and box-drawing headers.

**API:**
```go
ui.Header("Provisioning myserver.grimes.pro")
// Output: OUTPOST v0.1.0  Provisioning myserver.grimes.pro
```

**Migrate:** Every command that calls `printHeader()` or `printBoxTop()`.

### Phase 3: Checklist Component

Replace `printBoxTop/printBoxRow/printCheckItem/printFailItem/printBoxBottom`.

**API:**
```go
cl := ui.NewChecklist("Provisioning myserver.grimes.pro")
cl.Success("SSH connection established")
cl.Success("System dependencies installed")
cl.Fail("Claude Code CLI not found")
cl.Close()
// Error block after close:
cl.Error("claude not in PATH on remote host.")
cl.Fix("npm install -g @anthropic-ai/claude-code")
cl.Fix("outpost server setup 192.168.1.50")
```

**State machine:** Once `Fail()` is called, subsequent `Success()` calls panic (stop on first failure).

**Migrate:** doctor, server setup, server doctor, login, handoff.

### Phase 4: Styled Fields + Detail View

Replace `printField()`.

**API:**
```go
ui.Field("Branch", "wes/premium-healthchecks")
ui.Field("Mode", "headless")
ui.Field("Turns", "4 / 10")
```

**Migrate:** status detail, pickup, convert, handoff result.

### Phase 5: Table Component

Replace `newTable()` + `tabwriter`.

**API:**
```go
t := ui.NewTable("ID", "Branch", "Mode", "Turns", "Status", "Age")
t.Row("op-7f3a", "wes/premium-healthchecks", "headless", "4/10", ui.StatusRunning, "2m")
t.Footer("4 runs total", "1 running", "1 done", "1 waiting", "1 failed")
t.Render()
```

**Features:** Amber headers, status symbols in cells, `·` separated footer, ANSI-aware column alignment.

**Migrate:** status dashboard.

### Phase 6: Next-Step Hints + Error Blocks

**API:**
```go
ui.Hint("Watch", "outpost status op-7f3a --follow")
ui.Hint("Logs", "outpost logs op-7f3a --tail")

ui.Error("No server configured.")
ui.Fix("outpost login <host:port> <token>")
```

**Migrate:** Add to all command success/failure paths per ui.md examples.

### Phase 7: Progress Bar

**API:**
```go
pb := ui.NewProgress("Streaming to server")
pb.Update(sent, total)  // ⠸ Streaming to server...  ████████████████░░░░ 82%
pb.Done()                // ✓ Streamed (18.3 MB in 2.1s)
```

**Edge cases:** indeterminate mode (no total), instant completion (skip render), no-TTY fallback (single line on completion).

**Migrate:** handoff streaming.

### Phase 8: Confirmation Prompt

**API:**
```go
choice := ui.Confirm("Include uncommitted changes in handoff?",
    ui.Option("Yes, include working tree as-is"),
    ui.Option("No, only send committed files"),
    ui.Option("Cancel"),
)
```

**Edge cases:** `--force` selects first option, no-TTY without `--force` returns error, keyboard nav (up/down/enter).

**Migrate:** drop, handoff (dirty tree), login (TOFU).

### Phase 9: Log Line Formatter

**API:**
```go
ui.LogLine("op-7f3a", "10:42:17", "claude", "I'll start by reading...")
ui.LogLine("op-7f3a", "10:42:17", "tool", "Read file: CLAUDE.md")
```

**Features:** Fixed-width prefix columns, `claude` in purple, `tool` in dim, content in white.

**Migrate:** logs command.

### Phase 10: Global Output Mode Flags + Audit

- Wire `--no-color` as a global flag in `main.go`
- Wire `--quiet` as a global flag
- Audit every command's `--json` path for completeness
- Verify pipe detection works (no color, no spinners when piped)

---

## Edge Cases Checklist

- [ ] Piped output / no TTY: spinners degrade to static, prompts require --force
- [ ] ANSI-aware string width for table alignment
- [ ] Checklist stop-on-failure state machine
- [ ] Progress bar: unknown total (indeterminate mode)
- [ ] Progress bar: instant completion (skip render)
- [ ] Prompt with no TTY: error without --force
- [ ] Colored text inside table cells (visual width != byte length)
- [ ] Terminal resize: sample width per render
- [ ] stderr vs stdout contract enforced everywhere
- [ ] Error before any chrome (no header context yet)
- [ ] Version accessible from ui package without import cycle
- [ ] Two-column vitals layout (server doctor)
- [ ] Patch file list prefixes (+/~/- with colors)
- [ ] Age/duration formatting consistency

## Dependencies

No new external dependencies. Use raw ANSI escape codes for color. Use `golang.org/x/term` (already in Go's extended stdlib) for isatty and terminal width.

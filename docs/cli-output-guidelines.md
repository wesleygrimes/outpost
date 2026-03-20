# CLI Output Guidelines

> Foundational rules that every Outpost command must follow.
> For full mockups and per-command specs, see [OUTPOST_CLI_OUTPUT_SPEC.md](../OUTPOST_CLI_OUTPUT_SPEC.md).

## Brand & Identity

- **Header line**: `OUTPOST v{version}` (amber, uppercase) opens every structured output block, followed by the command context on the same line.
- **Run IDs**: Short form `op-XXXX` (4 hex chars), always amber. These are the primary anchor users type and grep for.
- **No symbol prefix** on the brand name. The name is the mark.

## Color Semantics

Each color has exactly one meaning. Do not cross-purpose them.

| Color  | Means                  | Never use it for       |
|--------|------------------------|------------------------|
| Amber  | Brand / identity / IDs | Status or content      |
| Green  | Success / complete     | In-progress states     |
| Cyan   | Active / in-progress   | Completed states       |
| Red    | Error / destructive    | Warnings               |
| Orange | Warning / caution      | Errors                 |
| Purple | Claude / AI activity   | Non-AI content         |
| Blue   | Actionable commands    | Labels or status       |
| White  | Primary content        | Chrome or decoration   |
| Dim    | Chrome / labels        | Primary content        |

## Symbols Are Standalone

Every status symbol is readable without color: `✓` vs `✗` vs `⚠` vs `⠸`. UTF-8 is the baseline; no ASCII fallback. `NO_COLOR` strips ANSI codes only, symbols and box-drawing stay.

| Symbol | Color  | State       |
|--------|--------|-------------|
| `✓`    | Green  | Complete    |
| `⠸`    | Cyan   | Running     |
| `●`    | Green  | Done (dot)  |
| `◉`    | Purple | Waiting     |
| `⚠`    | Orange | Warning     |
| `✗`    | Red    | Failed      |

## Structural Patterns

- **Checklist block**: `│` left border, status-prefixed lines, `└` closer.
- **Tables**: Amber column headers, space-aligned. No box drawing for data rows.
- **Next-step hints**: After the `└` closer, dim label + blue copy-pasteable command.
- **Log lines**: Fixed-width prefix `run-id timestamp source content`, tab-separated for machine parsing.

## Error Philosophy

- **Stop on first failure.** No partial success ambiguity. Checklist ends at the `✗` line.
- **Surface the why.** Show Claude's own explanation when a run fails.
- **Always offer a fix.** Every error includes a copy-pasteable recovery command.
- **Warnings are specific.** "6 turns of work" not "are you sure?". Show actual dirty files, not just a count.

## Output Modes

| Flag         | Effect                                             |
|--------------|----------------------------------------------------|
| `--json`     | Machine-readable JSON, replaces all human output   |
| `--quiet`    | Essential values only, no chrome                   |
| `--no-color` | Strip ANSI codes, keep symbols and structure       |
| `--force`    | Skip confirmation prompts (for scripting)          |
| `--follow`   | Stream updates in real-time                        |

## General Principles

- **One screenful.** Key vitals for any command should fit without scrolling.
- **Greppable.** Log lines and table rows are parseable by standard tools.
- **Copy-pasteable.** Every suggested command works if pasted directly.
- **Progressive disclosure.** Dashboard shows summary; drill into a run ID for detail.
- **Destructive actions confirm.** Show what's at risk and offer the safe alternative as a selection option.

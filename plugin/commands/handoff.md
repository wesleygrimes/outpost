---
description: "Hand off the current implementation plan to a remote Outpost server for execution"
argument-hint: "[headless|interactive] [--name N] [--branch B] [--max-turns N]"
---

Hand off the current session to a remote Outpost server for execution via session continuity.

Usage: /outpost:handoff [headless|interactive] [--name N] [--branch B] [--max-turns N]

Arguments: $ARGUMENTS

Follow these steps:

1. **Detect the current session ID.** Find the most recently modified `.jsonl`
   file in the Claude projects directory for this working directory:
   ```bash
   ls -t ~/.claude/projects/$(pwd | tr '/' '-')/*.jsonl 2>/dev/null | head -1
   ```
   Extract the session ID from the filename (strip path and `.jsonl` extension).
   If no session files found, tell the user this command must be run from
   within a Claude Code session.

2. **Parse arguments.** Defaults: mode=interactive, name=empty, branch=empty,
   max_turns=50. Parse from: $ARGUMENTS

3. **Submit to Outpost** using the outpost CLI:
   ```bash
   outpost handoff --session-id "${SESSION_ID}" \
     --mode "${MODE:-interactive}" \
     --name "${NAME}" \
     --branch "${BRANCH}" \
     --max-turns "${MAX_TURNS:-50}"
   ```

4. **Report results.** Parse the key=value output:
   - Run ID (from `id=...`)
   - Attach command (from `attach=...`, for interactive mode)
   - Mention that the remote session has full conversation context via session continuity
   - Check status: `/outpost:status <id>`
   - View logs: `/outpost:logs <id>`
   - Pick up when done: `/outpost:pickup <id>`
   - Watch for completion: `/outpost:watch <id>`

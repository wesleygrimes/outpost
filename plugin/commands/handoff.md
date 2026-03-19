---
description: "Hand off the current implementation plan to a remote Outpost server for execution"
argument-hint: "[headless|interactive] [--name N] [--branch B] [--max-turns N]"
---

Hand off the current implementation plan to a remote Outpost server for execution.

Usage: /outpost:handoff [headless|interactive] [--name N] [--branch B] [--max-turns N]

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
   - Check status: `/outpost:status <id>`
   - View logs: `/outpost:logs <id>`
   - Pick up when done: `/outpost:pickup <id>`
   - Watch for completion: `/outpost:watch <id>`

---
description: "Check the status of Outpost runs"
argument-hint: "[run_id]"
---

Check the status of Outpost runs.

Usage: /outpost:status [run_id]

Arguments: $ARGUMENTS

Run:
```bash
outpost status $ARGUMENTS
```

Display the output. Then suggest next steps:
- If status is "running": suggest waiting, viewing logs with `/outpost:logs <id>`, or attaching via SSH
- If status is "complete" and patch_ready is true: suggest `/outpost:pickup <id>`
- If status is "failed": show the log tail and suggest `/outpost:drop <id>`

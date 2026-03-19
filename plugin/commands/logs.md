---
description: "View log output from an Outpost run"
argument-hint: "<run_id> [-n lines]"
---

View log output from an Outpost run.

Usage: /outpost:logs <run_id> [-n lines]

Arguments: $ARGUMENTS

A run_id is required. If not provided, tell the user to run
`/outpost:status` first.

Run:
```bash
outpost logs $ARGUMENTS
```

Display the output. If the run is still active, suggest:
- Follow live: `outpost logs <id> --tail`
- Watch for completion: `/outpost:watch <id>`

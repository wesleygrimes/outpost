---
description: "Drop an Outpost run, stopping the session and discarding all work"
argument-hint: "<run_id>"
---

Drop an Outpost run. Stops the session and discards all work.

Usage: /outpost:drop <run_id>

Arguments: $ARGUMENTS

A run_id is required. If not provided, tell the user to run
`/outpost:status` first.

Run:
```bash
outpost drop $ARGUMENTS
```

Display the output. Confirm the run has been dropped.

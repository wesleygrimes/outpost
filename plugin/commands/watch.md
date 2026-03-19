---
description: "Watch an Outpost run in the background, interjecting when it completes or fails"
argument-hint: "<run_id>"
---

Watch an Outpost run in the background, polling every minute and interjecting when it finishes.

Usage: /outpost:watch <run_id>

Arguments: $ARGUMENTS

A run_id is required. If not provided, tell the user to run
`/outpost:status` first.

Follow these steps:

1. **Parse the run ID** from $ARGUMENTS.

2. **Start a background polling loop** using `/loop 1m` with the following check:

   Each iteration, run:
   ```bash
   outpost status <run_id>
   ```

   Parse the `status=` line from the output.

3. **On each poll:**
   - If status is `running` or `pending`: do nothing, stay silent, let the user continue working.
   - If status is `complete` or `failed`:
     a. Run `outpost logs <run_id> -n 50` to get the last 50 lines of output.
     b. Interject to tell the user the run has finished. Include:
        - The run ID
        - The final status (complete or failed)
        - The last 50 lines of log output
        - If complete: suggest `/outpost:pickup <run_id>`
        - If failed: suggest `/outpost:drop <run_id>` or reviewing logs with `/outpost:logs <run_id>`
     c. Stop the loop (the watch is done).
   - If status is `dropped`: stop the loop silently (someone already dropped it).

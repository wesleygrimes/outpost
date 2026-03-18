Kill a running Outpost session and capture any partial work.

Usage: /outpost-kill <run_id>

Arguments: $ARGUMENTS

A run_id is required. If not provided, tell the user to run `/outpost-status` first to find the run ID.

Follow these steps:

1. **Kill the run:**
   ```bash
   outpost kill $RUN_ID
   ```

2. **Report results** from the key=value output:
   - Show the run's final status
   - If patch_ready is true, inform the user they can pick up the partial work:
     `/outpost-pickup <run_id>`
   - If patch_ready is false, note that no changes were captured

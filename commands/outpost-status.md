Check the status of Outpost runs.

Usage: /outpost-status [run_id]

Arguments: $ARGUMENTS

Follow these steps:

1. **If no run_id provided**, list all runs:
   ```bash
   outpost status
   ```
   Display the table output to the user.

2. **If run_id provided**, get detail:
   ```bash
   outpost status $RUN_ID
   ```
   Display the key=value output to the user.

3. **Suggest next steps:**
   - If status is "running": suggest attaching via the attach command or waiting
   - If status is "complete" and patch_ready is true: suggest `/outpost-pickup <id>`
   - If status is "failed": show the log tail and suggest `/outpost-kill <id>` to clean up

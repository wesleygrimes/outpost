Check the status of Outpost runs.

Usage: /outpost-status [run_id]

Arguments: $ARGUMENTS

Follow these steps:

1. **Set up connection:**
   ```bash
   OUTPOST_URL="${OUTPOST_URL:-http://outpost.grimes.pro:7600}"
   OUTPOST_TOKEN="$(cat ~/.outpost-token 2>/dev/null)"
   ```

2. **If no run_id provided**, list all runs:
   ```bash
   curl -s "$OUTPOST_URL/runs" \
     -H "Authorization: Bearer $OUTPOST_TOKEN"
   ```
   Format the response as a table: ID, STATUS, MODE, CREATED.

3. **If run_id provided**, get detail:
   ```bash
   curl -s "$OUTPOST_URL/runs/$RUN_ID" \
     -H "Authorization: Bearer $OUTPOST_TOKEN"
   ```
   Display: ID, name, mode, status, base SHA, final SHA, created/finished times, patch readiness, attach command.

   If the run has a log tail, show the last lines.

4. **Suggest next steps:**
   - If status is "running": suggest attaching via SSH or waiting
   - If status is "complete" and patch is ready: suggest `/outpost-pickup <id>`
   - If status is "failed": show the log tail and suggest `/outpost-kill <id>` to clean up

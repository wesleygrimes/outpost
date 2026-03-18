Kill a running Outpost session and capture any partial work.

Usage: /outpost-kill <run_id>

Arguments: $ARGUMENTS

A run_id is required. If not provided, tell the user to run `/outpost-status` first to find the run ID.

Follow these steps:

1. **Set up connection:**
   ```bash
   OUTPOST_URL="${OUTPOST_URL:-http://outpost.grimes.pro:7600}"
   OUTPOST_TOKEN="$(cat ~/.outpost-token 2>/dev/null)"
   RUN_ID="<run_id from arguments>"
   ```

2. **Kill the run:**
   ```bash
   RESPONSE=$(curl -s -X DELETE "$OUTPOST_URL/runs/$RUN_ID" \
     -H "Authorization: Bearer $OUTPOST_TOKEN")
   ```

3. **Report results:**
   - Parse the JSON response and show the run's final status
   - If patch_ready is true, inform the user they can pick up the partial work:
     `/outpost-pickup <run_id>`
   - If no patch was generated, note that no changes were captured

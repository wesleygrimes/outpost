Check the status of Outpost runs.

Usage: /outpost-status [run_id]

Arguments: $ARGUMENTS

IMPORTANT: Do not use python for any step. Use only bash and curl. Responses are text/plain key=value pairs.

Follow these steps:

1. **Set up connection:**
   ```bash
   OUTPOST_URL="${OUTPOST_URL:-$(cat ~/.outpost-url 2>/dev/null)}"
   OUTPOST_TOKEN="${OUTPOST_TOKEN:-$(cat ~/.outpost-token 2>/dev/null)}"

   if [ -z "$OUTPOST_URL" ] || [ -z "$OUTPOST_TOKEN" ]; then
     echo "Missing Outpost config. Create ~/.outpost-url and ~/.outpost-token"
     exit 1
   fi
   ```

2. **If no run_id provided**, list all runs:
   ```bash
   curl -s "$OUTPOST_URL/runs" \
     -H "Authorization: Bearer $OUTPOST_TOKEN" \
     -H "Accept: text/plain"
   ```
   Each run is separated by `---`. Parse fields with `grep '^field=' | cut -d= -f2-`.
   Display as a table: ID, STATUS, MODE, CREATED.

3. **If run_id provided**, get detail:
   ```bash
   curl -s "$OUTPOST_URL/runs/$RUN_ID" \
     -H "Authorization: Bearer $OUTPOST_TOKEN" \
     -H "Accept: text/plain"
   ```
   The response has one field per line: id, name, mode, status, base_sha, created_at, attach, patch_ready, subdir, and optionally final_sha, finished_at, log_tail.
   Display all fields clearly.

4. **Suggest next steps:**
   - If status is "running": suggest attaching via SSH or waiting
   - If status is "complete" and patch_ready is true: suggest `/outpost-pickup <id>`
   - If status is "failed": show the log tail and suggest `/outpost-kill <id>` to clean up

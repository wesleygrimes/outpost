Kill a running Outpost session and capture any partial work.

Usage: /outpost-kill <run_id>

Arguments: $ARGUMENTS

IMPORTANT: Do not use python for any step. Use only bash and curl. Responses are text/plain key=value pairs.

A run_id is required. If not provided, tell the user to run `/outpost-status` first to find the run ID.

Follow these steps:

1. **Set up connection:**
   ```bash
   OUTPOST_URL="${OUTPOST_URL:-$(cat ~/.outpost-url 2>/dev/null)}"
   OUTPOST_TOKEN="${OUTPOST_TOKEN:-$(cat ~/.outpost-token 2>/dev/null)}"
   RUN_ID="<run_id from arguments>"

   if [ -z "$OUTPOST_URL" ] || [ -z "$OUTPOST_TOKEN" ]; then
     echo "Missing Outpost config. Create ~/.outpost-url and ~/.outpost-token"
     exit 1
   fi
   ```

2. **Kill the run:**
   ```bash
   RESPONSE=$(curl -s -X DELETE "$OUTPOST_URL/runs/$RUN_ID" \
     -H "Authorization: Bearer $OUTPOST_TOKEN" \
     -H "Accept: text/plain")
   ```

3. **Report results.** Parse the response:
   ```bash
   STATUS=$(echo "$RESPONSE" | grep '^status=' | cut -d= -f2-)
   PATCH_READY=$(echo "$RESPONSE" | grep '^patch_ready=' | cut -d= -f2-)
   ```
   - Show the run's final status
   - If patch_ready is "true", inform the user they can pick up the partial work:
     `/outpost-pickup <run_id>`
   - If patch_ready is "false", note that no changes were captured

Pick up a completed Outpost run and apply the patch locally.

Usage: /outpost-pickup <run_id>

Arguments: $ARGUMENTS

IMPORTANT: Do not use python for any step. Use only bash, curl, and git. Responses are text/plain key=value pairs.

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

2. **Verify the run is complete:**
   ```bash
   STATUS_RESPONSE=$(curl -s "$OUTPOST_URL/runs/$RUN_ID" \
     -H "Authorization: Bearer $OUTPOST_TOKEN" \
     -H "Accept: text/plain")
   ```
   Check that status is "complete" or "killed" and patch_ready is "true":
   ```bash
   STATUS=$(echo "$STATUS_RESPONSE" | grep '^status=' | cut -d= -f2-)
   PATCH_READY=$(echo "$STATUS_RESPONSE" | grep '^patch_ready=' | cut -d= -f2-)
   SUBDIR=$(echo "$STATUS_RESPONSE" | grep '^subdir=' | cut -d= -f2-)
   ```
   If not ready, report the current status and stop.

3. **Download the patch:**
   ```bash
   mkdir -p /tmp/outpost-patches
   curl -s -o "/tmp/outpost-patches/$RUN_ID.patch" \
     "$OUTPOST_URL/runs/$RUN_ID/patch" \
     -H "Authorization: Bearer $OUTPOST_TOKEN"
   ```

4. **Create a local worktree and apply:**
   ```bash
   git worktree add -b "outpost/$RUN_ID" "../outpost-$RUN_ID"
   cd "../outpost-$RUN_ID"
   ```
   If the run has a `subdir` value, the patch paths are relative to that subdirectory. Apply with `--directory`:
   ```bash
   if [ -n "$SUBDIR" ]; then
     git apply --directory="$SUBDIR" "/tmp/outpost-patches/$RUN_ID.patch"
   else
     git apply "/tmp/outpost-patches/$RUN_ID.patch"
   fi
   git add -A
   git commit -m "outpost: run $RUN_ID"
   ```

5. **If step 4 fails** (patch doesn't apply cleanly):
   - Report the error
   - Remove the local worktree: `git worktree remove "../outpost-$RUN_ID"`
   - Do NOT clean up the remote run (it stays for retry or SSH inspection)
   - Stop here

6. **Only on success**, clean up the remote run:
   ```bash
   curl -s -X POST "$OUTPOST_URL/runs/$RUN_ID/cleanup" \
     -H "Authorization: Bearer $OUTPOST_TOKEN" \
     -H "Accept: text/plain"
   ```

7. **Clean up local temp:**
   ```bash
   rm -f "/tmp/outpost-patches/$RUN_ID.patch"
   ```

8. **Report results:**
   - Worktree path: `../outpost-$RUN_ID`
   - Branch: `outpost/$RUN_ID`
   - Files changed (from git diff --stat)
   - Next steps: review the changes, then create a PR or merge

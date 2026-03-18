Hand off the current implementation plan to a remote Outpost server for execution.

Usage: /outpost [headless|interactive] [--name N] [--branch B] [--max-turns N]

Arguments: $ARGUMENTS

Follow these steps:

1. **Compile the implementation plan.** Summarize everything from this conversation into a self-contained markdown document. The remote session has ZERO context from this conversation, so include:
   - Full problem description and requirements
   - Specific files to modify and how
   - Any relevant code snippets or patterns
   - Acceptance criteria
   - Testing instructions
   Write the plan to a temp file:
   ```bash
   cat > /tmp/outpost-plan-$$.md << 'PLAN_EOF'
   <your compiled plan here>
   PLAN_EOF
   ```

2. **Parse arguments.** Defaults: mode=interactive, name=empty, branch=empty, max_turns=50. Parse from: $ARGUMENTS

3. **Create a git bundle** of the current repo:
   ```bash
   git bundle create /tmp/outpost-$$.bundle --all
   ```

4. **Submit to Outpost:**
   ```bash
   OUTPOST_URL="${OUTPOST_URL:-http://outpost.grimes.pro:7600}"
   OUTPOST_TOKEN="$(cat ~/.outpost-token 2>/dev/null)"

   RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$OUTPOST_URL/handoff" \
     -H "Authorization: Bearer $OUTPOST_TOKEN" \
     -F "plan=</tmp/outpost-plan-$$.md" \
     -F "bundle=@/tmp/outpost-$$.bundle" \
     -F "mode=${MODE:-interactive}" \
     -F "name=${NAME}" \
     -F "branch=${BRANCH}" \
     -F "max_turns=${MAX_TURNS:-50}")

   HTTP_CODE=$(echo "$RESPONSE" | tail -1)
   BODY=$(echo "$RESPONSE" | head -n -1)
   ```

5. **Clean up temp files:**
   ```bash
   rm -f /tmp/outpost-$$.bundle /tmp/outpost-plan-$$.md
   ```

6. **Report results.** If HTTP 202, parse the JSON response and report:
   - Run ID
   - Attach command: `ssh outpost -t 'tmux attach -t <id>'`
   - Check status: `/outpost-status <id>`
   - Pick up when done: `/outpost-pickup <id>`

   If HTTP 429, report that Outpost is at capacity and show which runs are active.
   If any other error, report the error message.

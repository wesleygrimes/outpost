Hand off the current implementation plan to a remote Outpost server for execution.

Usage: /outpost [headless|interactive] [--name N] [--branch B] [--max-turns N]

Arguments: $ARGUMENTS

IMPORTANT: Do not use python for any step. Use only bash, curl, git, and tar.

Follow these steps:

1. **Compile the implementation plan.** Summarize everything from this conversation into a self-contained markdown document. The remote session has ZERO context from this conversation, so include:
   - Full problem description and requirements
   - Specific files to modify and how
   - Any relevant code snippets or patterns
   - Acceptance criteria
   - Testing instructions
   Write the plan to `/tmp/outpost-plan.md` using the Bash tool with a heredoc.

2. **Parse arguments.** Defaults: mode=interactive, name=empty, branch=empty, max_turns=50. Parse from: $ARGUMENTS

3. **Detect monorepo context.** Check if the current working directory is a subdirectory of the git root:
   ```bash
   SUBDIR=$(git rev-parse --show-prefix)
   SUBDIR=${SUBDIR%/}
   ```
   If `SUBDIR` is non-empty, this is a monorepo subdirectory. The archive and subdir field must reflect this.

4. **Create a tarball** of the working tree (tracked + untracked non-ignored files, no .git). Run from the current working directory so paths are relative to it:
   ```bash
   git ls-files -co --exclude-standard | tar czf /tmp/outpost-archive.tar.gz -T -
   ```

5. **Submit to Outpost:**
   ```bash
   OUTPOST_URL="${OUTPOST_URL:-$(cat ~/.outpost-url 2>/dev/null)}"
   OUTPOST_TOKEN="${OUTPOST_TOKEN:-$(cat ~/.outpost-token 2>/dev/null)}"

   if [ -z "$OUTPOST_URL" ] || [ -z "$OUTPOST_TOKEN" ]; then
     echo "Missing Outpost config. Create ~/.outpost-url and ~/.outpost-token"
     exit 1
   fi

   RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$OUTPOST_URL/handoff" \
     -H "Authorization: Bearer $OUTPOST_TOKEN" \
     -H "Accept: text/plain" \
     -F "plan=</tmp/outpost-plan.md" \
     -F "archive=@/tmp/outpost-archive.tar.gz" \
     -F "mode=${MODE:-interactive}" \
     -F "name=${NAME}" \
     -F "branch=${BRANCH}" \
     -F "max_turns=${MAX_TURNS:-50}" \
     -F "subdir=${SUBDIR}")

   HTTP_CODE=$(echo "$RESPONSE" | tail -1)
   BODY=$(echo "$RESPONSE" | sed '$d')
   ```

6. **Clean up temp files:**
   ```bash
   rm -f /tmp/outpost-archive.tar.gz /tmp/outpost-plan.md
   ```

7. **Report results.** The response is key=value pairs (one per line). Parse with grep/cut:
   - `echo "$BODY" | grep '^id=' | cut -d= -f2-` for the run ID
   - `echo "$BODY" | grep '^attach=' | cut -d= -f2-` for the attach command

   If HTTP 202, report:
   - Run ID
   - Attach command (for interactive mode)
   - Check status: `/outpost-status <id>`
   - Pick up when done: `/outpost-pickup <id>`

   If HTTP 429, report that Outpost is at capacity.
   If any other error, report the error message.

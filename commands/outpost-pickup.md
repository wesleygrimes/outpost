Pick up a completed Outpost run and apply the patch locally.

Usage: /outpost-pickup <run_id>

Arguments: $ARGUMENTS

A run_id is required. If not provided, tell the user to run `/outpost-status` first to find the run ID.

Follow these steps:

1. **Apply the run:**
   ```bash
   outpost pickup $RUN_ID
   ```

2. **If successful**, the output contains worktree path, branch name, and diff stats. Report:
   - Worktree path and branch name
   - Files changed
   - Next steps: review the changes, then create a PR or merge

3. **If it fails** (patch doesn't apply cleanly):
   - Report the error to the user
   - The CLI cleans up the worktree automatically on failure
   - The remote run is NOT cleaned up (stays for retry or SSH inspection)

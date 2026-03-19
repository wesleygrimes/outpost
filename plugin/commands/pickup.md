---
description: "Pick up a completed Outpost run, apply patch, verify, and commit"
argument-hint: "<run_id>"
---

Pick up a completed Outpost run and integrate it into the current branch.

Usage: /outpost:pickup <run_id>

Arguments: $ARGUMENTS

A run_id is required. If not provided, tell the user to run
`/outpost:status` first.

Follow these steps in order:

1. **Check for uncommitted changes.** Run `git status --porcelain`.
   If there is any output (staged or unstaged changes, untracked files that
   matter), STOP and tell the user:
   "You have uncommitted changes. Please stash or commit them before
   picking up an Outpost run."
   Do NOT proceed.

2. **Download the patch.** Run:
   ```bash
   outpost pickup $ARGUMENTS
   ```
   Parse the `patch=` line to get the patch file path.
   If the command fails (no patch ready, run still running, etc.), show the
   error and suggest `/outpost:status <id>` or `/outpost:logs <id>`.

   Also check the output for a `session=<id>` line. If present, the remote
   conversation has been downloaded and can be resumed locally.

3. **Apply the patch.** Run:
   ```bash
   git apply <patch_path>
   ```
   If this fails (conflicts, etc.), show the error and ask the user how
   they want to proceed. Do NOT continue automatically.

4. **Run project checks.** Look at the project's CLAUDE.md for the CI
   command (typically `make check`). Run it.
   - If checks **pass**: proceed to step 5.
   - If checks **fail**: show the failures and ask the user if they want
     to commit anyway, fix the issues, or revert with `git checkout .`.

5. **Commit.** Review the diff with `git diff --cached` (stage first with
   `git add -A`). Write a conventional commit message based on what the
   patch changed. Commit the changes.

6. **Report.** Show the commit hash, summary of changes. If a session ID
   was returned, inform the user they can resume the remote conversation
   with `claude --resume <session-id>` to continue with full context.
   Suggest next steps (push, create PR, etc.).

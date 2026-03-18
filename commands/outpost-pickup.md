Pick up a completed Outpost run.

Usage: /outpost-pickup <run_id>

Arguments: $ARGUMENTS

A run_id is required. If not provided, tell the user to run
`/outpost-status` first.

Run:
```bash
outpost pickup $ARGUMENTS
```

Display the output (patch path, files changed).
The patch is now on disk. Ask the user how they want to apply it:
- Apply to current branch: `git apply <patch>`
- Create a new branch first: `git checkout -b <branch> && git apply <patch>`
- Review first: `cat <patch>` or open in editor

Do NOT apply the patch automatically. Let the user decide.

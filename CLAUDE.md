# Outpost

Remote Claude Code session runner. Single Go binary, two roles: server (`serve`) and client (`handoff`, `status`, `logs`, `pickup`, `drop`).

## CI / Pre-commit

Run before every commit:

```bash
make all
```

This runs `go vet`, `golangci-lint run` (v2, strict config), tests, and builds the binary. Do not commit if this fails.

Full CI (includes GoReleaser config validation):

```bash
make ci
```

## Format

```bash
make fmt
```

Uses gofumpt (stricter than gofmt) via golangci-lint.

## Worktrees

Work on feature branches using git worktrees. Worktrees live as sibling directories (`../outpost-<name>`) on a branch named `<name>`.

```bash
make wt-new name=my-feature     # creates ../outpost-my-feature on branch my-feature
make wt-list                    # list all worktrees
make wt-remove name=my-feature  # remove worktree and directory
make wt-prune                   # clean up stale worktree references
```

Claude Code agent worktrees go in `.claude/worktrees/` (gitignored).

## CLI Output

All terminal output must follow [docs/ui.md](docs/ui.md).

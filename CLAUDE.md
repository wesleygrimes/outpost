# Outpost

Remote Claude Code session runner. Single Go binary, two roles: server (`serve`) and client (`handoff`, `status`, `logs`, `pickup`, `drop`).

## CI / Pre-commit

```bash
make check
```

This runs `go vet`, `golangci-lint run` (v2, strict config), and tests.

Full CI (includes GoReleaser config validation):

```bash
make ci
```

## Build

```bash
make build
```

## Format

```bash
make fmt
```

Uses gofumpt (stricter than gofmt) via golangci-lint.

## CLI Output

All terminal output must follow [docs/cli-output-guidelines.md](docs/cli-output-guidelines.md). Full per-command mockups are in [OUTPOST_CLI_OUTPUT_SPEC.md](OUTPOST_CLI_OUTPUT_SPEC.md).

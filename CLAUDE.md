# Outpost

Remote Claude Code session runner. Single Go binary with three subcommands: `setup`, `serve`, `runs`.

## CI / Pre-commit

```bash
make check
```

This runs `go vet` and `golangci-lint run` (v2, strict config).

## Build

```bash
make build
```

## Format

```bash
make fmt
```

Uses gofumpt (stricter than gofmt) via golangci-lint.

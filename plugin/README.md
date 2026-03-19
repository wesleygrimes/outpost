# Outpost Plugin for Claude Code

Remote Claude Code session runner. Hand off implementation plans to a remote server for execution.

## Commands

- `/outpost:handoff` - Hand off an implementation plan to a remote Outpost server
- `/outpost:status` - Check the status of Outpost runs
- `/outpost:logs` - View log output from a run
- `/outpost:pickup` - Pick up a completed run's patch
- `/outpost:drop` - Stop and discard a run
- `/outpost:watch` - Watch a run in the background, interjecting when it finishes

## Installation

```bash
curl -fsSL https://git.grimes.pro/wesleygrimes/outpost/raw/branch/main/install.sh | bash
```

This installs the `outpost` binary and registers the plugin with Claude Code.

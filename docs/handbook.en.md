# DevToolbox user handbook

**Languages:** [中文](./handbook.md) · English

Website: https://dev-toolbox.tonyxyz.com

## What it does

Start from a **project** and manage two things:

1. Which runtime stack the project needs (Node / Java / Maven / Gradle / Android SDK / Go / Python).
2. Which Git identity the project should use.

Anything that changes the environment or Git config requires an **explicit confirm**. Scan and doctor stay read-only.

## Install

Download from [Releases](https://github.com/xyztony999/dev-toolbox/releases):

| Artifact | Notes |
|----------|--------|
| Windows installer / portable zip | Per-user install; no PATH edits; no automatic system config |
| Linux desktop tar.gz / deb | Needs `libwebkit2gtk-4.1` |
| Linux CLI deb / rpm / tar.gz | Depends on `git` |

Verify against `SHA256SUMS` in the same release.

## Desktop client

1. Register or scan local projects after launch.
2. Open a project to see runtime requirements and the current Git identity.
3. When installing or switching runtimes, review the preview plan (version, mirror, checksum, target, size), then confirm.
4. Applying a Git identity is also preview-first; apply is a **full overwrite** (managed keys missing from the profile are cleared).
5. Theme can follow the system, or be fixed to light / dark in settings.

Note: the GUI **cannot** rewrite an already-open parent terminal. New sessions and child processes launched by the tool pick up the selected environment.

## Common CLI commands

```text
# Projects
dev-toolbox project scan <path>

# Runtimes
dev-toolbox runtime discover [dir]
dev-toolbox runtime install <tool> <selector> --confirm
dev-toolbox runtime legacy devswitch inspect|preview|import

# Git identity
dev-toolbox git profiles
dev-toolbox git use <id> --confirm
dev-toolbox git current
dev-toolbox git bind <id> <dir> --confirm
dev-toolbox git scan
dev-toolbox git audit --expect <id> --strict
dev-toolbox git doctor
dev-toolbox git legacy preview|import
```

Use `--confirm` (or the flag documented by `--help`) for a real apply; without it, commands are usually dry-run / preview.

## Security boundaries

- Does not run discovered scripts, wrappers, Git hooks, or project commands.
- Installs write only under DevToolbox-owned user directories.
- Credential tokens are never stored, logged, or echoed.
- See [Security notes](./security.en.md).

## Migrating from older tools

- Runtimes: `runtime legacy devswitch …` (inspect / preview, then import).
- Git identity: `git legacy preview|import` (from gitid `config.json`).

Migration always previews and backs up first; import is idempotent by source summary and does not hijack old command names.

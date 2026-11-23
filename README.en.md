<p align="center"><img src="assets/logo.svg" width="128" alt="DevToolbox logo"></p>

# DevToolbox

[![CI](https://github.com/xyztony999/dev-toolbox/actions/workflows/ci.yml/badge.svg)](https://github.com/xyztony999/dev-toolbox/actions/workflows/ci.yml)
[![Website](https://img.shields.io/badge/website-dev--toolbox.tonyxyz.com-a34b2a)](https://dev-toolbox.tonyxyz.com)
[![Release](https://img.shields.io/github/v/release/xyztony999/dev-toolbox?display_name=tag)](https://github.com/xyztony999/dev-toolbox/releases/latest)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux-lightgrey)](https://github.com/xyztony999/dev-toolbox/releases)

**Languages:** [中文](./README.md) · English

Website: https://dev-toolbox.tonyxyz.com

A toolbox for multi-stack project runtimes and Git identity. Start from a project, see which runtime and which Git identity it should use, then confirm the change explicitly.

- Shared Go application service used by CLI and desktop.
- Go WebView host (Windows WebView2 / Linux WebKitGTK)—no Electron, and Node is not the Git core.
- Vue 3 + TypeScript + Ant Design Vue UI, built with Vite and embedded by Go.
- Driven by the ai-project-engine pipeline: design → Gate 1 → implement → Gate 2 (`profiles/dev-toolbox/workspace/`).

## Capabilities

### Project context

- Register / scan / rescan local projects: detect Git worktrees (including nested repos and worktrees) and build modules; extract version requirements from `package.json` engines, `.nvmrc`, `pyproject`, `pom`, Gradle toolchain/Wrapper, with source and confidence. Read-only—does not run project scripts.

### User-level runtimes

- Static discovery, inspection, and registration for Node / Java / Maven / Gradle / Android SDK / Go / Python.
- Download and install from fixed providers: preview plan (version / mirror / checksum / target / size) → explicit confirm → checksum verify → staging unpack → atomic place. Cancel and partial Android package retry supported.
- DevSwitch user-state migration (preview / backup / idempotent import). Global defaults change only when set explicitly; project and session choices do not silently rewrite them.

### Git identity

- Multi-profile archives (name / email / signingkey / gpg.format / custom keys / URL rewrite / account picker). Apply is full overwrite: managed keys missing from the profile are cleared so the previous identity does not linger.
- Dry-run and apply share one algorithm; batch multi-repo results; `includeIf` directory binding (Windows `gitdir/i` real-path match; drift warning before unbind).
- Repo scan (✓/✗/⚠/⊕), commit audit with strict gate (mailmap suggestions only—never writes `.mailmap`), read-only Git doctor, per-repo mirror push, credentials via git credential protocol into the system helper (token never stored, logged, or echoed).
- Legacy gitid (`config.json`) preview / backup / idempotent import.

### Desktop client

- Go WebView2 host + embedded single-file frontend; bridge exposes a whitelist only; async work does not freeze the window.
- Light / dark theme follows the system; app icon matches brand assets.

## Downloads

Prebuilt artifacts: [Releases](https://github.com/xyztony999/dev-toolbox/releases)

- **Windows desktop**: Inno Setup installer (per-user, no admin, no PATH edits, no auto config) or portable zip.
- **Linux desktop** (amd64): portable tar.gz and deb; needs system `libwebkit2gtk-4.1` (Ubuntu 24.04+ / recent Fedora).
- **Linux CLI**: deb (Debian/Ubuntu), rpm (Fedora/RHEL/openSUSE), plus manual tar.gz/zip. Depends on git.
- All artifacts ship with SHA256SUMS.

## Build

Requires Go ≥ 1.24, Node (frontend build only), and on Windows MinGW-w64 (CGO) plus WebView2 Runtime.

```powershell
# Static CLI (no CGO)
go build -o dist/dev-toolbox.exe ./cmd/dev-toolbox

# Desktop client
npm --prefix frontend run build:desktop   # writes internal/desktop/ui/index.html
$env:CGO_ENABLED = '1'                    # needs MinGW-w64 gcc
go build -tags gui -ldflags="-H windowsgui" -o dist/dev-toolbox-gui.exe ./cmd/dev-toolbox

# Frontend demo mode (mock data, no disk writes)
npm --prefix frontend install
npm --prefix frontend run dev
```

CLI examples:

```text
dev-toolbox project scan <path>
dev-toolbox runtime discover [dir] / install <tool> <selector> --confirm
dev-toolbox runtime legacy devswitch inspect|preview|import
dev-toolbox git profiles / use <id> --confirm / current / bind <id> <dir> --confirm
dev-toolbox git scan / audit --expect <id> --strict / doctor
dev-toolbox git legacy preview|import
```

## Security boundaries

- Scan and doctor are read-only; never run project scripts, wrappers, hooks, or candidate binaries.
- Installs write only under DevToolbox user directories; no PATH / registry / shell profile edits; no automatic default switch.
- Git config truth is Git itself; writes go through `git config` only after explicit confirm.
- Credential tokens enter the system helper via the git credential protocol only—never profiles, logs, or echoes.
- Legacy migration: preview → backup → idempotent import by source summary; does not hijack old commands or co-write old files.

More detail: [Security notes](./docs/security.en.md) · [User handbook](./docs/handbook.en.md)

## Repository layout

- `cmd/` `internal/`: CLI, desktop host, and core (project / runtimecatalog / runtimeinstall / userstate / gitidentity / desktop).
- `frontend/`: Vue 3 + Ant Design Vue sources and build scripts.
- `profiles/`: ai-project-engine assets—business knowledge, D0–D4 design, Gate reports, module index.
- `docs/`: handbook, security, merge direction, inventory.
- `website/`: marketing site (Cloudflare Workers Static Assets).
- `tools/`: inventory scripts and icon generator.

## Origins & acknowledgements

Git identity semantics come from [gitid](https://github.com/xyztony999/gitid); runtime management and download providers from [DevSwitch](https://github.com/xyztony999/devswitch) (same author, MIT). Migration is validated against original sandbox behavior specs; upstream projects stay independent. See [NOTICE](./NOTICE).

Third-party: [webview_go](https://github.com/webview/webview_go) (MIT), [ulikunitz/xz](https://github.com/ulikunitz/xz) (BSD-3-Clause), Vue / Ant Design Vue / Vite (respective licenses).

## License

[MIT](./LICENSE) © 2026 Xinyi Zhang

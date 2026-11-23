# Security notes

**Languages:** [中文](./security.md) · English

## Trust model

DevToolbox is a single-user local desktop tool. CLI and desktop call the same Go application service—there is no server, multi-tenant model, or remote API. The security boundary is “local user ↔ system/project config”, not “remote attacker ↔ service”.

## Path parameters are a feature, not a vulnerability

Static analysis often flags “function takes a path and reads files” as path traversal. Several core features intentionally let the user name a directory and gather evidence **read-only**:

- `runtime discover/inspect`: identify runtimes under a user-chosen directory (or common install locations); read version markers (for example JDK `release`) and expected binaries (`bin/java.exe`, etc.) without executing them.
- `project scan`: enumerate manifests under a user-chosen root.
- `git audit`: read `git log` for a user-chosen repository.

These reads run with the current user’s privileges, cross no privilege boundary, and never concatenate paths into a shell. External processes use argument arrays (see the whitelist and sanitization in `internal/gitidentity/gitrun.go`). Path-traversal findings on these entry points are **accepted design**.

## Boundaries that actually apply

- The desktop bridge exposes a whitelist only (`internal/desktop/desktop.go`). The UI cannot pass arbitrary commands or download URLs. Install URLs come from fixed provider templates with host allowlists and public DNS checks.
- Credential tokens go straight into the system helper via the git credential protocol—never written to disk, logs, or responses.
- Unpack rejects path traversal, symlinks, and oversized payloads; conflicting same-version targets do not overwrite unknown directories.
- Tests run in an isolated sandbox (temporary `HOME` / `APPDATA` / `GIT_CONFIG_GLOBAL`) and do not mutate the real user environment.

## Reporting issues

Please report vulnerabilities privately through GitHub Security Advisories (Security tab) rather than public issues.

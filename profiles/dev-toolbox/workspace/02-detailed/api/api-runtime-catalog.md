# Go 应用服务契约 — 本机运行时发现与登记

版本：v1.0 | 日期：2026-10-27 | 模块：runtime-migration | 状态：Gate 1 待检查

> CLI/GUI 直接调用共用 Go 应用服务；不是 HTTP API。所有结果不包含凭据。

## DTO

`RuntimeCandidate`: `candidateId`, `tool`, `version`, `major`, `home`, `binary?`, `source`, `evidence[]`, `availability`, `confidence`, `duplicateOf?`, `digest`。版本未知使用空字符串并以 `confidence=unknown` 表示。

`DiscoveryResult`: `scanId`, `status(complete|partial|cancelled)`, `candidates[]`, `issues[]`。

## 操作契约

| 操作 | 参数 | 返回 | 作用域/副作用 |
|---|---|---|---|
| `runtime.discover` | 可选 `tools[]`, `includePath=true`, `roots[]`, `scanId` | DiscoveryResult | 只读；roots 必须是用户显式选择的目录；context 可取消 |
| `runtime.inspectPath` | `tool`, `home` | RuntimeCandidate 或错误 | 只读单目录检查；不得执行候选 |
| `runtime.register` | `candidates[]:{candidateId,digest,tool,home,source}`, `expectedRevision`, `confirm=true` | `{state:UserRuntimeState,counts:{created,linked,skipped}}` | 只写 DevToolbox 用户状态；路径+tool 去重；服务器重检候选摘要 |
| `runtime.getUserState` | 无 | UserRuntimeState | 只读，含全局选择与清单 |
| `runtime.setGlobalDefault` | `runtime`, `installationId`, `expectedRevision`, `confirm=true` | UserRuntimeState | 只更新显式工具的用户级默认 |

## 错误码与并发

`RUNTIME_INVALID_TOOL`, `RUNTIME_INVALID_PATH`, `RUNTIME_CANDIDATE_INVALID`, `RUNTIME_CANDIDATE_STALE`, `USER_RUNTIME_REVISION_CONFLICT`, `RUNTIME_INSTALLATION_NOT_FOUND`, `RUNTIME_INSTALLATION_UNAVAILABLE`, `RUNTIME_CONFIRMATION_REQUIRED`, `RUNTIME_SCAN_CANCELLED`, `RUNTIME_PERMISSION_DENIED`。错误响应含 requestId、稳定 code 和不含秘密的 message。

register 在一个状态锁内校验 revision、复核静态候选 digest、按工具+规范化 home 去重并原子写入。取消发生于提交前则不写状态；提交阶段不响应取消并返回确定结果。源目录和系统环境永不作为写目标。

## 更新记录

| 版本 | 日期 | 变更类型 | 说明 |
|---|---|---|---|
| v1.0 | 2026-10-27 | initial | 发现、检查、登记及用户默认契约 |

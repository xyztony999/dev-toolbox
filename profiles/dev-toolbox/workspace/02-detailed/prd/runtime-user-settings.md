# PRD 补充 — 用户级运行时状态保留

版本：v1.0 | 日期：2026-10-27 | 归属：user-runtime-state-migration | 状态：本切片 Gate 1 PASS

## 目标

保留 DevSwitch 用户级 `state.json` 中的全局当前选择和运行时清单，使 DevToolbox 在保留原安装目录的前提下接管用户当前使用的运行时设置。Project 与 Session 的临时/目录级选择作为覆盖层，不能修改用户全局默认。

## 数据映射

| DevSwitch 字段 | DevToolbox 概念 | 迁移规则 |
|---|---|---|
| `schema` | LegacySource.schema | 未知 schema 时只展示，不导入 |
| `current[tool]` | GlobalRuntimeSelection[tool] → RuntimeInstallation ID | 通过 `tool + canonical home` 精确关联；目标不存在时保留原路径与 unresolved 状态 |
| `runtimes[].tool` | RuntimeInstallation.runtime | 保留七工具标识；不映射到项目的 RuntimeRequirement |
| `version`, `major` | RuntimeInstallation version fields | 保留原值和来源，不升级或规范化覆盖 |
| `home`, `binary` | LocalBinding/runtime location | 本机私有，原位引用，不复制/移动；保留原路径字符串用于恢复 |
| `source` | RuntimeInstallation.source | 保留 system/imported/downloaded 等已知值；未知值原样保留 |
| `label`, `vendor` | RuntimeInstallation display metadata | 非空时保留 |

DevSwitch `state.json` 没有镜像偏好字段；单次 `--mirror`/`DEVSWITCH_MIRROR` 不迁移为持久设置。用户 shell hooks、shim、PATH 与环境变量是派生生效物，不在本状态导入操作内改写。

## 业务规则

1. 旧状态检测只读；用户发起迁移后先显示来源文件、schema、工具选择、运行时条目和不可用路径。
2. 执行前把读取到的原文件字节写入 DevToolbox 自己的数据目录备份，并记录来源标识、时间、摘要和目标 schema。
3. 导入要求用户确认；相同来源摘要重复导入为幂等，不产生重复实例或覆盖用户在 DevToolbox 中后续修改的全局选择。
4. 迁移期间不写旧 `state.json`，不挪动运行时目录，不与 DevSwitch 双写；原工具继续可单独使用。
5. 缺失路径不丢弃或自动扫描替代；记录 unresolved，并允许用户之后显式重新关联。
6. 项目/会话选择与 GlobalRuntimeSelection 分离；只有“设为全局默认”操作能够更新全局选择。
7. 用户可查看迁移前备份摘要；将备份恢复到 DevToolbox 当前状态的自动流程留待单独设计，不在本切片执行。
8. 未识别工具条目或缺少必要字段的条目在预览中计入“未支持”，原状态备份仍保留；用户可继续导入已识别项。

## 本切片明确暂不覆盖

完整七工具扫描、手动发现/导入、安装 provider、shim/shell hook 接管、PATH 生效、卸载及 gitid 身份迁移需在相应模块 D0-D4 中逐项设计。本切片不授权运行下载器、写 OS 设置或重写 shell profile。

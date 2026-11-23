# DevToolbox 用户手册

**语言：** 中文 · [English](./handbook.en.md)

官网：https://dev-toolbox.tonyxyz.com

## 它做什么

以**项目**为入口，管理两件事：

1. 这个项目需要哪套运行时（Node / Java / Maven / Gradle / Android SDK / Go / Python）。
2. 这个项目该用哪个 Git 身份。

所有会改环境或 Git 配置的操作都要**显式确认**；扫描与体检只读。

## 安装

从 [Releases](https://github.com/xyztony999/dev-toolbox/releases) 下载：

| 产物 | 说明 |
|------|------|
| Windows 安装包 / 便携 zip | 用户级安装，不改 PATH，不自动改系统配置 |
| Linux 桌面 tar.gz / deb | 需要 `libwebkit2gtk-4.1` |
| Linux CLI deb / rpm / tar.gz | 依赖 `git` |

校验：对照同目录 `SHA256SUMS`。

## 桌面客户端

1. 启动后登记或扫描本机项目。
2. 打开项目，查看运行时要求与当前 Git 身份。
3. 安装或切换运行时时，先看预览计划（版本、镜像、校验、目标、体积），再确认。
4. 应用 Git 身份时同样先预览；应用为**全量覆盖**（档案没有的受管键会被清掉）。
5. 主题可跟随系统，或在设置里固定浅色 / 深色。

说明：GUI **不能**改写已经打开的父终端环境；新开会话或由工具启动的子进程才会带上选定环境。

## CLI 常用命令

```text
# 项目
dev-toolbox project scan <path>

# 运行时
dev-toolbox runtime discover [目录]
dev-toolbox runtime install <tool> <selector> --confirm
dev-toolbox runtime legacy devswitch inspect|preview|import

# Git 身份
dev-toolbox git profiles
dev-toolbox git use <id> --confirm
dev-toolbox git current
dev-toolbox git bind <id> <dir> --confirm
dev-toolbox git scan
dev-toolbox git audit --expect <id> --strict
dev-toolbox git doctor
dev-toolbox git legacy preview|import
```

`--confirm`（或产品约定的确认开关）用于真跑；省略时通常为 dry-run / 预览语义，以命令帮助为准。

## 安全与边界

- 不运行扫描到的脚本、Wrapper、Git hook 或项目命令。
- 安装只写入 DevToolbox 自有用户目录。
- 凭据 token 不落盘、不进日志、不回显。
- 详见 [安全说明](./security.md)。

## 从旧工具迁移

- 运行时：`runtime legacy devswitch …`（先 inspect / preview，再 import）。
- Git 身份：`git legacy preview|import`（来自 gitid 的 `config.json`）。

迁移前会预览与备份；导入按来源摘要幂等，不抢占旧命令名。

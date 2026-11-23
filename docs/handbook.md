# DevToolbox 用户手册

**语言：** 中文 · [English](./handbook.en.md)

官网：https://dev-toolbox.tonyxyz.com

## 它做什么

按**项目**来管两件事：

1. 这个项目要用哪套运行时（Node / Java / Maven / Gradle / Android SDK / Go / Python）
2. 这个项目该用哪个 Git 身份

会改环境或 Git 配置的操作，都要你先确认。扫描和检查只读，不会乱动。

## 安装

到 [Releases](https://github.com/xyztony999/dev-toolbox/releases) 下载：

| 文件 | 说明 |
|------|------|
| Windows 安装包 / 便携 zip | 装到当前用户，不改 PATH，也不自动改系统配置 |
| Linux 桌面 tar.gz / deb | 需要系统里有 `libwebkit2gtk-4.1` |
| Linux 命令行 deb / rpm / tar.gz | 依赖 `git` |

下载后对照同目录的 `SHA256SUMS` 校验一下。

## 桌面客户端

1. 启动后，登记或扫描本机项目。
2. 打开某个项目，看它的运行时要求和当前 Git 身份。
3. 安装或切换运行时：先看预览（版本、镜像、校验、目标路径、体积），再确认。
4. 切换 Git 身份：也是先预览。真正应用时会**完整覆盖**——档案里没有的受管项会被清掉，避免残留上一身份。
5. 主题可以跟系统走，也可以在设置里固定浅色 / 深色。

注意：已经打开的终端，环境改不了；新开的终端，或由本工具拉起的子进程，才会带上选定环境。

## 常用命令

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

加 `--confirm` 才是真正执行；不加一般是预览。具体以 `dev-toolbox <命令> --help` 为准。

## 安全边界

- 不会去跑扫到的脚本、Wrapper、Git hook 或项目命令。
- 安装只写到 DevToolbox 自己的用户目录。
- 凭据 token 不落盘、不写日志、不回显。
- 详见 [安全说明](./security.md)。

## 从旧工具迁过来

- 运行时：`runtime legacy devswitch …`（先 `inspect` / `preview`，再 `import`）
- Git 身份：`git legacy preview|import`（来自 gitid 的 `config.json`）

迁移会先预览并备份；导入可重复执行，也不会抢走旧工具的命令名。

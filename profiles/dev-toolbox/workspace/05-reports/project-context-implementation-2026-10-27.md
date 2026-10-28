# project-context 实现进度

日期：2026-10-27。Gate 1 报告：`gate1-project-context-2026-10-27.md`。

## 已实现

- Go `internal/project`：白名单清单只读扫描、Git 普通仓库/worktree/嵌套仓库发现、模块与 Node/Python/Java/Go/Maven/Gradle/Android SDK 证据提取、unknown/conflict 标记。
- 本机项目登记 JSON 存储：用户配置目录、文件权限收紧、原子替换、登记锁、重扫 revision 校验；部分扫描保留上次完整的发现快照。
- Go CLI：`project scan|add|list|show|rescan|remove`，登记和移除有显式确认；扫描不运行项目脚本、Wrapper 或 Git 命令。
- Vue 3 + TypeScript + Ant Design Vue：项目预览/列表/详情交互，定义白名单桥接消息类型。
- 开发模式演示客户端：独立内存 mock 和单文件 `dist-demo/index.html`，包含 Node/Python/Java/Android 多模块样例；默认生产构建不包含 fixture。
- 迁移来源基线：记录在 `migration-baseline-2026-10-27.md`；未修改来源仓库。

## 验证与未完成

- `go build ./...`：通过。
- `go vet ./...`：通过。
- `npm run typecheck`：通过。
- `npm run build`：通过；Vite 提示主 chunk 约 1.5 MB，后续可拆分。
- `npm run build:demo`：通过；生成约 1.5 MB 单文件演示客户端。
- 未添加或运行测试；Gate 2 未执行。
- UI 尚未接入 Go WebView 宿主/原生桥接，当前不能作为桌面应用运行；进度事件与界面取消尚未接入。
- 版本解析器是静态初版，尚未用来源仓库行为样本和隔离 fixture 覆盖；Java/Gradle 语法变体、冲突判定和跨平台权限场景仍需 Gate 2 前验证。
- `runtime-migration`、`git-identity-migration` 及其他模块仍为待办；此模块 Gate 1 不代表其他模块可直接开始代码迁移。

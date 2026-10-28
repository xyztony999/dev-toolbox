# user-runtime-state-migration 实现进度

日期：2026-10-27。设计门控：`gate1-user-runtime-state-migration-2026-10-27.md`（PASS）。来源基线：`migration-baseline-2026-10-27.md`。

## 已实现

- 新增独立的 `user-runtime-state.json`，与项目登记的 `projects.json` 分开存储。
- 只读探测 DevSwitch 默认用户配置目录下的 `state.json`；只接受常规文件和 schema 1，不跟随符号链接，限制文件大小。
- 预览七种工具的 `current` 与 `runtimes` 映射，保留 version/major/home/binary/source/label/vendor；缺失路径全局选择标记 unresolved，未知工具/字段计入 unsupported。
- 用户确认后先将旧文件原字节备份到 DevToolbox 自有目录，再幂等导入；校验预览 digest，原子写入新状态，不写旧文件、不搬移运行时目录。
- 提供 `runtime state`、`runtime legacy devswitch inspect|preview|import` 和明确确认的 `runtime set-default` CLI 操作。
- Vue 页面展示全局默认、路径状态和导入预览；开发演示桥使用虚构数据。项目选择与用户全局选择分开。

## 验证与边界

- `go build ./...`、`go vet ./...`：通过。
- `npm run typecheck`、`npm run build`：通过。
- 未添加或运行测试；Gate 2 仍待执行。
- 未读取真实 DevSwitch 用户配置，也未调用导入命令；来源文件与安装目录保持未触碰。
- Go WebView 原生桥接尚未接通，所以桌面页面尚不能实际读写此新状态；CLI 与后端服务已可构建。
- 自动恢复备份、七工具扫描/下载/切换的完整行为以及 shell hook/PATH 迁移仍需后续设计与验证。

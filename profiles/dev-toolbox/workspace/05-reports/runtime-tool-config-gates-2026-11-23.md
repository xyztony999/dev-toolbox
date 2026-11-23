# Gate 1 + Gate 2 — runtime-tool-config（合并 tab + 配置面泛化）

日期：2026-11-23 | 输入：d0 / api / interaction（runtime-tool-config）| Gate1：**PASS** | Gate2：**PASS**

## Gate 1（设计）
- 需求澄清：用户全选 node/python/maven 配置 + 版本管理泛化 + java JAVA_HOME + android ANDROID_HOME，并追加"其它环境也看需要什么"。
- 配置面按工具原生事实源分级：可写（go env / npm config / pip config / maven 托管 mirror）与只读（JAVA_HOME/ANDROID_HOME/GRADLE_USER_HOME/cargo 镜像），只读项说明杠杆，不代写系统环境变量与复杂 TOML（rule 2/5/10）。
- maven 仅托管 id=dev-toolbox 的 mirror，写前备份、幂等、可移除，不动用户其它 mirror（rule 8）。
- 子进程 CREATE_NO_WINDOW（沿用 goenv 教训）；测试隔离 GOENV/HOME/DEVTOOLBOX_MAVEN_SETTINGS（rule 9）。
- D0↔API↔PageSpec 字段/错误码/confirm 一致。

## Gate 2（交付）
- 自动化：`go test ./...` 全绿；toolconfig 6 例（maven upsert 保留用户 mirror+备份+幂等、remove、dryRun 不写、confirm 强制、只读守卫×4、golang 委托）；`vue-tsc` 通过；`build:desktop`+gui 构建通过。
- 桥接：新增 runtime.toolConfigRead/toolConfigWrite；移除被取代的 runtime.goEnvRead/goEnvSet（白名单收紧）。
- 前端：合并「全局默认」+「Go 管理」为单「运行时」tab，a-collapse 每工具分区（版本管理+环境配置）；默认/卸载/注销泛化到全部工具。
- 真机：解锁后像素截图自查通过——单「运行时」tab、每工具折叠头（工具名+默认摘要）、展开即加载配置（java 显示 JAVA_HOME 只读 chip+杠杆说明；golang 可编辑 GOPROXY/GOPATH/GOBIN+预设、GOROOT 只读）；无安装工具显示引导空态。
- 自查发现并修复：误用未注册组件 `a-collapse-pane`（正确为 `a-collapse-panel`），曾导致分区平铺、配置不加载（e76127e）。
- 未覆盖：node/python/maven 的真实写路径未在本机执行（避免改用户 ~/.npmrc/pip/settings），由单测+dryRun 契约覆盖；建议人工点一次预览确认。
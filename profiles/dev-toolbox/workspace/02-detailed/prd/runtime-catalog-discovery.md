# PRD — 本机运行时发现与登记

版本：v1.0 | 日期：2026-10-27 | 模块：runtime-migration | 状态：Gate 1 待检查

## 1. 目标与范围

让开发者发现并登记本机已有的 Node、Java、Maven、Gradle、Android SDK、Go、Python 安装，为已存在的用户级默认选择提供可管理的本机清单。只读扫描和登记是本 PRD 唯一新增能力。

不包含下载/安装/卸载、写 PATH/注册表/shell profile、项目会话覆盖、Git 身份、扫描项目 Wrapper 或执行运行时版本命令。

## 2. 领域模型

- RuntimeCandidate：一次发现产生的临时候选，含 candidateId、tool、version、major、home、binary、source、evidence、status、digest。
- RuntimeInstallation：用户确认后持久化的实例，含 installationId、tool、version、major、home、binary、source、label/vendor、availability。
- GlobalRuntimeSelection：已有 user-runtime-state 中每工具一条选择。登记安装不自动改变该选择，除非现有产品规则中第一次显式选择动作另行调用。
- DiscoveryIssue：候选路径的受限访问、未知格式、缺失元数据或重复说明。

## 3. 功能需求

| ID | 需求 | 验收标准 |
|---|---|---|
| RT-01 | 扫描来源 | 只读 PATH、平台常见根及用户指定目录；不全盘递归 |
| RT-02 | 静态识别 | 七工具使用白名单可执行文件/元数据；绝不执行候选 |
| RT-03 | 版本不明 | 无可信版本证据时返回 unknown，不猜版本 |
| RT-04 | 去重 | 同工具相同规范化 home 合并，保留全部来源证据 |
| RT-05 | 用户登记 | 预览候选后须确认；登记前重新核实摘要与目录 |
| RT-06 | 全局默认 | 新登记实例不会自动覆盖既有用户选择；修改继续使用 revision 和确认 |
| RT-07 | 隔离 | 只写 DevToolbox 用户状态，不写来源目录、PATH、注册表、shell profile |
| RT-08 | 部分失败 | 可继续展示成功候选，同时列明失败路径、原因和重试动作 |
| RT-09 | 平台 | Windows/Linux；对不支持的来源明确返回 unsupported，不假报成功 |

## 4. 约束与安全

候选数量和路径读取范围有上限；不跟随候选根外符号链接，不递归所有用户目录。扫描与 register 请求支持 context 取消；重复登记幂等。用户配置的读取/写入隔离由独立 HOME/USERPROFILE/APPDATA/XDG_CONFIG_HOME 验证。

## 5. 状态流转

`idle → scanning → preview(complete|partial) → selected → registering → registered`。扫描取消转为 `cancelled` 且无副作用；空结果停留在 preview；登记失败保留 preview 供重试；摘要过期返回 `stale` 并要求重新扫描；关闭面板或取消确认只清理临时候选。

## 6. 更新记录

| 版本 | 日期 | 变更类型 | 说明 |
|---|---|---|---|
| v1.0 | 2026-10-27 | initial | 七工具本机候选扫描与显式登记 |

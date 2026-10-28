# DevToolbox 项目指令

本仓是 Go 核心 + Go WebView + Vue 3 + TypeScript + Ant Design Vue 的新产品。当前处于初始化与设计准备阶段。

## 引擎入口

使用安装的 `ai-project-engine` 插件，不复制或修改引擎 payload。所有 engine-cli 操作显式传 `--root D:/Project/dev-toolbox`，入口按插件 manifest 解析。

执行设计、生成或验证前，必须显式读取：

1. 根 `CLAUDE.md` 的 Profile 绑定，以及 `.claude/settings.json`、`.claude/.profile-mounts.json`。
2. `profiles/dev-toolbox/profile.yaml` 及其 `knowledge.inject` 与四类 knowledge。
3. 激活的两个栈 manifest 及其 rules、specs；执行相应阶段时读取对应 agent。
4. 插件引擎规则及当前阶段 skill。

Codex 不会自动执行 CLAUDE.md 的 @import。激活后应显式加载，不能把 .claude 文件存在等同于规则已生效。

## 产品纪律

- CLI、桌面和托盘复用同一 Go 应用服务；前端只展示与发出结构化请求。
- Git 当前状态由 Git 自身解析；身份应用必须保留原项目的全量覆盖、目录绑定、凭据不落盘等行为。
- 项目环境优先作用于启动的会话/子进程；GUI 无法直接改写已经打开的父终端环境。
- 仓库共享版本要求与本机路径、Git 身份绑定分离；凭据不进入项目文件、日志或广播状态。
- 不自动执行扫描到的脚本、Wrapper、Git hook 或项目命令；安装/切换属于明确用户动作。
- 测试隔离 HOME/USERPROFILE/APPDATA/XDG_CONFIG_HOME 和 Git 配置、凭据 helper；不得改真实 PATH、注册表、shell profile、用户 Git 配置。
- 原项目是迁移来源，未经任务要求不修改、删除或搬移；迁移前记录最新 HEAD 和工作区差异，不能假设旧快照仍然有效。
- 新业务代码须先完成所需设计与门控；初始化报告不得冒充 Gate 1/2 通过。


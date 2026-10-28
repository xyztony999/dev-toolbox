# 业务概念与边界

- 使用者：需要在多个个人/团队项目间切换的开发者；本机应用，未引入服务端多租户或账号系统。
- Project：用户登记的项目集合，可包含多个仓库与多个构建模块。
- Repository：Git 工作树；worktree、普通仓库和嵌套仓库需单独识别，不能只找 .git 目录。
- Module：带独立清单的子路径；版本要求与父项目不必相同。
- RuntimeRequirement：带来源与作用域的版本要求；未声明、歧义、冲突要如实表达。
- RuntimeInstallation：本机安装的运行时，含版本、路径、来源、所有权与工具种类。
- GlobalRuntimeSelection：本机用户为每种工具显式选择的默认运行时；只存在于用户级状态，不等于项目或会话选择。
- RuntimeCatalog：本机可用/已导入运行时清单，包含路径与来源；已安装目录和清单元数据分开管理。
- IdentityProfile：Git 作者、签名、URL 重写、账号选择等配置档案，不存 token。
- LocalBinding：本机项目/目录对运行时与身份的关联，个人路径不写进共享要求。
- SessionOverride：仅作用于启动的会话/子进程；不得反向修改 GlobalRuntimeSelection。
- EffectiveContext：按目录和会话求值的当前环境；Git 实际值由 Git 自身提供。
- ApplyPlan：变更计划，区分作用域、副作用、需要安装的能力；执行结果允许部分失败。
- Diagnostic：只读问题发现，显示严重程度、来源和建议，修复是另一个显式操作。

首批流程：登记或扫描项目 → 发现模块与要求 → 对比本机运行时/Git 状态 → 只读诊断 → 明确作用域后应用或启动会话。此为流程方向，详细状态机与交互由 D0–D4 定义。

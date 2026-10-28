# 知识来源

登记日：2026-10-27。以下是来源记录，不是用户批准完整产品设计的声明。

- S-01：本聊天用户要求：D:/Project/dev-toolbox；注册 profile；Go 核心 + WebView + Vue 3 + Ant Design Vue；扫描现有项目决定合并方向。类型：用户确认约束。
- S-02：D:/Project/gitid/README.zh-CN.md、AGENTS.md、cli/package.json、desktop/package.json、profiles/gitid/rules/product-rules.md、cli/lib/core.js。类型：现有实现和产品不变量。
- S-03：D:/Project/devswitch/README.md、go.mod、frontend/package.json、frontend/src/tools.ts、internal/models/models.go、internal/service/autoswitch.go、internal/gui/gui.go。类型：现有实现；README 对七工具能力描述滞后，模型和注册表优先。
- S-04：根 docs/project-inventory.json。类型：20 个一级目录的静态清单；归档/附带代码/缺失清单分开记录。
- S-05：根 docs/project-inventory.md、docs/merge-direction.md。类型：本轮分析形成的工程方向，具体接口与交互仍需设计。
- S-06：已安装 ai-project-engine 0.4.0 payload 的 profile 契约、命令规格与引擎规则。类型：接入规范。

来源仓工作区会变化；2026-10-27 读取时 gitid HEAD 为 e33e7288596ead7a6ccc11d07ec6803b19a5e39d，devswitch HEAD 为 5e3fc4ada6725f0ae552994c3262b53ccade5730。这里只记录采样，不锁定后续迁移版本；迁移前重新检查 HEAD/dirty 状态。

未推断用户尚未选择的数据库、云服务、完整 API、产品版本号、安装标识或远端仓库地址。此轮没有执行 kb 生成命令，也没有把工程建议标为用户确认项。

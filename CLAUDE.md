# DevToolbox

多技术栈项目开发环境与 Git 身份管理工具箱。技术路线：Go 共用核心 + Go WebView + Vue 3 + TypeScript + Ant Design Vue。

初始化依据和范围见 `README.md`、`docs/project-inventory.md`、`docs/merge-direction.md`。Profile 绑定节由 engine-cli use/off 维护。

## Profile 绑定

当前激活：**dev-toolbox**（business，depends: go-core, go-webview-vue3-antd）

@profiles/dev-toolbox/CLAUDE.md

> 引擎各阶段执行前，按 skills 内「Profile 绑定」节完成槽位替换（manifest：profiles/dev-toolbox/profile.yaml）。
> 切换/摘除：/profile use <name> / /profile off

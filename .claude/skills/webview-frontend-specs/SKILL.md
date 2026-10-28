---
name: webview-frontend-specs
description: Go WebView 桥接、Vue 3/TypeScript/Ant Design Vue 视图与宿主集成规格。
---

# 桌面前端规格

完成激活链绑定，读取 `{ws.interaction}`、`{ws.api}`、两个激活栈规则和业务规则；前端根来自业务 slots.frontend_root。

每个页面说明路由/导航、字段来源、空/加载/错误/冲突态、用户动作、作用域、副作用反馈。组件使用 Ant Design Vue，不迁移来源中的 Naive UI 依赖。

桥接按操作定义请求/响应、事件、错误、取消与版本；不假定有 HTTP 后端。开发态 mock 与生产适配器契约相同，生产不依赖开发服务器。

窗口/托盘只调用共用 Go 服务；Go GUI 管理线程与事件生命周期。静态前端生产产物经明确构建流程同步到 Go embed，资源路径需兼容内嵌加载。

实现后的最低验证：`npm --prefix frontend run typecheck`、`npm --prefix frontend run build`、实际 GUI tag 编译与启动抽检。上述 scripts/tag 由实现定义，未实现前不得宣称已通过。

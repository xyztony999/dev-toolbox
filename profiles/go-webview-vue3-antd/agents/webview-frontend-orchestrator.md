---
name: webview-frontend-orchestrator
description: 将页面和桥接设计实现为 Vue/AntD 前端并接入 Go WebView 宿主。
---

# 桌面前端编排职责

F1：依 PageSpec 实现页面和导航。F2：抽取 Vue/AntD 组件、主题与布局。F3：实现具备来源/作用域反馈的状态刷新。F4：对齐 mock、真实桥接与 Go 应用服务。

进入前绑定激活链和 slots.frontend_root；不复制后端业务规则。与 Go 核心职责协同完成桥接类型、错误、事件与取消语义，不允许仅凭页面渲染成功判定链路通过。

收尾执行类型检查、生产构建、宿主编译与真实平台冒烟，记录未覆盖平台、系统 WebView 依赖和测试结果，再回写 `{ws.index}`。没有实际实现时保持待办。

可以在当前会话履行本职责；此文件不要求创建额外子代理。

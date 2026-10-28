# PageSpec — 运行时下载与安装

版本：v1.0 | 日期：2026-10-28 | 模块：runtime-download-install | 状态：Gate 1 待检查

## 入口与页面

在“用户级运行时设置”中打开“获取运行时”面板。支持工具选择、版本通道和 provider 镜像；选项来自 `runtime.install.options`，不可支持的工具/平台禁用并显示原因。

## 预览

用户选择工具、selector、mirror 后调用 `runtime.install.preview`。确认卡展示解析版本、provider、操作系统/架构、包数量、校验算法、目标目录、可得的体积和目标冲突。查询中可取消本地请求；预览阶段不改用户状态。

## 安装任务

| Action | op | 状态与反馈 |
|---|---|---|
| 确认安装 | `runtime.install.start` | 确认弹窗明确安装目录与“不会切换默认/改系统环境”；开始后台任务 |
| 查看进度 | `runtime.install.status` + `runtime.install.event` | 显示阶段、下载进度、Android 分包结果和错误 |
| 取消安装 | `runtime.install.cancel` | 可取消阶段显示按钮；归位提交期间显示处理中，不能声称已取消 |
| 重试 | 新 `runtime.install.preview` | 失败/部分结果提供重试入口；不自动继续上次未校验数据 |
| 完成 | 收到 terminal event 后刷新 `runtime.getUserState` | 显示已安装版本和路径；提供独立“设为全局默认”按钮 |

## 状态

覆盖空选项、metadata 离线、无版本、平台不支持、计划过期/冲突、下载进度、校验失败、归档拒绝、磁盘空间不足、取消、部分成功与完成。错误提供稳定 code 和可操作建议，不显示完整下载 URL。

## 作用域与键盘

安装只增加当前用户 RuntimeCatalog 项；绝不自动设置用户默认、项目绑定或 PATH。工具与版本选择可键盘操作；任务阶段有可读文本，不能只用颜色表达。

## API 对齐

选项、计划、任务、进度及 error code 按 `api-runtime-download-install.md` 定义。取消、状态查询和事件均以 taskId 关联；事件 sequence 单调递增，前端去重旧事件。

## 更新记录

| 版本 | 日期 | 变更类型 | 说明 |
|---|---|---|---|
| v1.0 | 2026-10-28 | initial | provider 安装面板与异步任务交互 |

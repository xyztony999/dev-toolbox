# Gate 1 验证报告 — session-launcher

检查日期：2026-11-03 | 设计版本：v1.0 | 范围：上下文求值（八工具注入/未注入）、终端枚举与启动、项目/仓库两级入口

## 检查结果

| 检查项 | 结果 | 依据 |
|---|---|---|
| G1-01 架构一致性 | PASS | D0/BA/AA/FL/PRD/API/PageSpec 统一：新 internal/session 服务复用 userstate/gitidentity 只读解析；环境组装规范落在 AA/D3 一处 |
| G1-02 PRD 场景与角色 | PASS | 全未选择、部分不可用、无终端、cwd 失效、多会话并存均有覆盖；SL-05 明确"不碰全局"验收；SL-09 注入可被人直接验证 |
| G1-03 API 契约完整 | PASS | preview/open 两操作参数/响应/副作用齐全；SessionToolState/TerminalId/错误码与环境组装规范（PATH 去重前置、JAVA_HOME、其余不注入）定义完整 |
| G1-04 页面状态与动作 | PASS | 终端面板（预览/cwd/终端选择/启动确认）、仓库快捷入口、空态与禁用态齐备 |
| G1-05 API 与页面字段对齐 | PASS | 面板完全由 preview 数据驱动；open 参数与面板选择一一对应 |
| G1-06 状态流转 | PASS | 无状态机：preview 幂等只读 → open 显式动作，与契约一致 |
| G1-07 枚举一致 | PASS | SessionToolStatus/TerminalId 已登记公共枚举池（code+label） |
| G1-08 接口形态 | PASS | 本地工具类 Go API + 白名单桥接，无 HTTP |
| G1-09 changelog | PASS | 七份文档均含 v1.0 记录 |
| G1-10 状态索引 | PASS | session-launcher 已写入 INDEX |

## 边界强调

- ADR-007 落实：注入仅限启动的子进程树；SL-05 要求启动前后全局选择与环境变量零变化（B 阶段以此为测试断言）。
- Windows 终端启动用 CREATE_NEW_CONSOLE（可见窗口）——与既有"隐藏控制台"原语方向相反，属有意例外，实现时需显式分支。

## 判定

**PASS** — 可进入实现。

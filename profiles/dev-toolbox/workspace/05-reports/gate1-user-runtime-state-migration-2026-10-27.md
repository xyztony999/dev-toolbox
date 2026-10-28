# Gate 1 验证报告 — user-runtime-state-migration

**日期**：2026-10-27（Asia/Shanghai）  
**版本**：v1.0  
**激活链**：dev-toolbox + go-core + go-webview-vue3-antd  
**范围**：仅 DevSwitch 用户级 `state.json` 预览、备份、幂等导入与全局选择展示/显式更新。

## PRD 就绪

| 检查 | 结果 | 证据 |
|---|---|---|
| 目标、角色、范围 | PASS | 本机 DevSwitch 用户；明确限定用户级状态迁移 |
| 领域对象与关系 | PASS | UserRuntimeState、GlobalRuntimeSelection、RuntimeInstallation、LegacyImportRecord |
| 工具输入/输出（T9） | PASS | 来源 schema/digest、映射预览、异常项、导入结果和错误状态均已定义 |
| 幂等与部分失败 | PASS | digest 去重、备份先行、目标原子替换、旧来源只读；备份恢复操作留后续设计 |
| 用户级/项目级/会话级边界 | PASS | 项目/会话覆盖不得调用 setGlobalDefault；Git 身份独立 |

## D1-D4 对齐

| 检查 | 结果 | 说明 |
|---|---|---|
| BA/AA/FL | PASS | 用户状态发现→预览→确认导入映射到 Reader/Mapper/Store 与设置/迁移视图 |
| 领域模型 | PASS | 旧 current 精确映射到 installation；无法关联时保留 unresolvedHome |
| Go 操作契约 | PASS | inspect、preview、import、getUserState、setGlobalDefault 定义了作用域/副作用/幂等/错误 |
| PageSpec 对齐 | PASS | 展示字段和用户动作均可映射到契约；敏感路径不进日志/广播 |
| 冲突/状态 | PASS | unknown schema 阻断；重复摘要幂等；缺失路径保留待处理；来源变化要求重预览 |

## 技术栈适配

该切片是本机工具状态迁移，无 HTTP/CRUD/SQL/MQ。按 Go 栈操作型服务规格和 WebView 白名单桥接规则审核。七工具扫描、安装 provider、shell hook/环境变更不在本 Gate 范围。

## 判定

**总体判定：PASS**。该切片可进入实现。此结果不代表完整 `runtime-migration` 或 `project-context` Gate 2 通过，也不授权修改旧 `state.json`、安装目录、PATH、注册表或 shell profile。

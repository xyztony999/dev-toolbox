# Gate 1 验证报告 — git-gpg-keygen

检查日期：2026-11-27 | 设计版本：v1.0 | 范围：为已有 IdentityProfile 生成/选用 OpenPGP 与 SSH 签名密钥并写回签名字段；含 options/preview/create/list/select/export

**Profile**：dev-toolbox（stacks: go-core, go-webview-vue3-antd）  
**策略**：W1 — WARN 项仅建议、不阻塞进入实现  
**只读**：未改业务代码

## 输入产物

| 阶段 | 路径 |
|------|------|
| D0 | `00-architecture/d0-git-gpg-keygen.md` |
| D1 | `business-arch` / `application-arch` / `function-list`-git-gpg-keygen.md |
| D2 | `02-detailed/prd/git-gpg-keygen.md` |
| D3 | `02-detailed/api/api-git-gpg-keygen.md` |
| D4 | `02-detailed/interaction/git-gpg-keygen.md` |
| 枚举 | `01-solution/common-spec/common-enums.md`（GitKeyType / Algorithm / Expiry） |
| INDEX | `workspace/INDEX.md` → modules.git-gpg-keygen |

## 引擎检查项（G1-01~G1-11）

| 检查项 | 结果 | 依据 |
|--------|------|------|
| G1-01 架构与模块设计一致 | PASS | D0/BA/AA/FL/PRD/API/PageSpec 统一：挂 git 域增强、扩展 `internal/gitidentity`（KeygenService），无新聚合根；写回既有 IdentityProfile 签名三字段；CLI/桌面共用服务 |
| G1-02 PRD 场景与角色 | PASS | 单角色本机开发者；KG-01~15 覆盖探测/类型分支/算法有效期/档案绑定/口令/预览/生成/列表/选用/导出/双入口/平台/测试隔离；异常与部分成功见 §8 |
| G1-03 API 契约完整 | PASS | 六操作均有请求/响应/副作用；写操作 `confirm`；错误码 11 项（含工具缺失、参数非法、部分成功写档失败、导出覆盖）；CLI 与桥接白名单对齐 |
| G1-04 交互覆盖 | PASS | 档案编辑「提交签名」内嵌生成/选用/导出；无新路由/侧栏；空态、工具缺失、部分成功、二次确认、口令清空与防重复提交均有 |
| G1-05 API↔交互字段对齐 | PASS | 类型/算法/有效期/口令/enableGpgSign/outputPath/overwrite/confirm 与 D3 一致；六 Action ↔ 六 op；`app.pickFolder` 为既有宿主能力 |
| G1-06 状态机 | PASS | 工具类无业务状态机；流为 options→填参→preview→create\|select→可选 export，与 PRD §6 / API 一致 |
| G1-07 枚举统一无冲突 | PASS | GitKeyType / GitKeyAlgorithm / GitKeyExpiry 已入 common-enums（code+label）；合法组合与 PRD §7、API §3.3 一致；复用既有 GitSigningFormat；无重复定义 |
| G1-08 接口形态 | PASS | 本地工具类 `git.*` 操作（非 HTTP）；路径风格对齐既有 git-identity；符合本产品 Go 应用服务约定（非企业三级 HTTP） |
| G1-09 changelog | PASS | D0/D1×3/PRD/API/PageSpec 均含 v1.0 \| 2026-11-27 更新记录 |
| G1-10 状态索引 | PASS | INDEX 已有 `git-gpg-keygen`；D0–D4 stages complete；本报告后 gate1 记为 warn（W1 可继续） |
| G1-11 跨阶段模型一致 | PASS | 领域过程对象与 DTO 对应（Request/Plan/Result/LocalKeySummary/PublicKeyExport）；PageSpec Action 与 API 操作一一对应；关联聚合根 IdentityProfile 贯穿 |

## PRD 就绪（skill 展开，工具类）

| 层 | 通过 | 状态 |
|----|------|------|
| 第一层 G1–G5（目标/范围/角色/对象/关系） | 5/5 | PASS |
| 第二层 T9（工具 I/O）+ T1a（被引用：写回 git-identity） | 适用项通过 | PASS |
| 第三层 N1–N3（非功能/异常/规则） | 已写 | INFO |

不适用：T2–T8（非业务流程/台账/集成对接主类型）。

## API / 前后端契约（适配本栈）

| 编号 | 检查项 | 状态 |
|------|--------|------|
| CT-001 | Action op ↔ API 操作 | PASS（6/6） |
| CT-002 | 表单/结果字段 ↔ DTO | PASS |
| CT-003 | 枚举 code 前后端一致 | PASS |
| CT-004 | 全部 Action 有 API | PASS |
| — | 无独立 HTTP dict 端点；选项来自 `git.keygenOptions` | PASS |

企业 Java 模板中的 HTTP/Result\<T\>/DDL/operList/MQ 等章对本模块 **N/A**（Go 本地应用服务 + WebView 白名单）。

## 产品 / 栈规则对照（设计层）

| 规则 | 结果 | 说明 |
|------|------|------|
| product-rules 3（凭据不落盘） | PASS | 口令/私钥不进档案、日志、广播、桥接回显；CLI `--passphrase-stdin` |
| product-rules 4（单一 Go 服务） | PASS | CLI/GUI 共用 KeygenService |
| product-rules 9（测试隔离） | PASS | 临时 GNUPGHOME / SSH 目录；可注入可执行路径 |
| GC-03（参数数组、禁 shell 拼接） | PASS | AA/API 明确 |
| GC-05 / GC-06（沙箱、可取消、稳定错误分类） | PASS（设计） | 实现期按 GC 验证 |
| WV-03 / WV-04 / WV-06 | PASS | 白名单六操作；前端不拼命令；敏感字段不进事件 |

## WARN 建议项（W1，不阻塞）

1. **PageSpec §4.1 持久化文案**：仍先写「保存档案后才会持久化」再订正为服务端已写 ProfileStore。实现时 UI 文案只保留最终结论「已更新档案的签名字段」，避免歧义。
2. **命名表层差异**：PRD 过程对象名 `KeygenRequest` vs API `KeygenCreateRequest`；交互文案用 git 键名（signingkey / gpg.format），API/档案 DTO 用 camelCase（signingKey / gpgFormat）——与既有 git-identity 一致。实现以 `api-git-identity` 的 `GitProfile` 与本模块 API 为准。
3. **SSH `comment`**：API 可选、默认 `devtoolbox:{profileId}`；D4 未提供编辑控件。可接受；若产品要可改注释，需补交互字段后再实现。
4. **桥接命名**：`git.keygen*` 与 `git.keyList/Select/Export` 前缀并存，CLI 统一挂 `git keygen`——已文档化，实现时保持映射表即可。

## 总结

| 项 | 值 |
|----|-----|
| **判定** | **WARN**（W1） |
| CRITICAL 失败 | 0 |
| 可继续实现 | **是** |
| 建议 | 实现前按上列 WARN 固化文案与命名口径；不必回滚 D0–D4 重开设计 |

**结论**：设计→代码门控可通过（带建议）；可进入后端/前端实现。

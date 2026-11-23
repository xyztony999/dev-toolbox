# Gate 1 — go-management 设计门控

日期：2026-11-23 | 输入：d0-go-management / api-go-management / interaction/go-management | 结论：**PASS**

## 检查项

| # | 检查 | 结果 |
|---|---|---|
| 1 | 需求澄清完整（两项能力，用户确认） | PASS |
| 2 | 事实源：go env 读/-w 写，不另维护真值（rule 1/4） | PASS |
| 3 | 显式动作+预览+确认，不隐式改全局/系统环境（rule 2/6） | PASS |
| 4 | 卸载归属保护：仅 downloaded 且 RuntimeRoot 内可删目录；外部只注销（rule 8） | PASS |
| 5 | 不执行项目命令；go env 为工具内省（rule 7） | PASS |
| 6 | 测试隔离 GOENV/HOME/USERPROFILE/APPDATA/XDG_*（rule 9） | PASS（设计已列） |
| 7 | D0↔API↔PageSpec 字段/错误码/confirm 一致 | PASS |
| 8 | 边界明确（GOROOT 只读、不做 shim、不接管所有权） | PASS |

## 风险与缓解
- go env -w 写用户 GOENV：测试用临时 GOENV 隔离；真机验证仅读+一次可逆写（direct↔原值）。
- 删目录误伤：归属判定双重条件（Source+路径前缀），dryRun 预览列路径，confirm 才执行。

## 结论
设计可进入 B/F。无阻断项。
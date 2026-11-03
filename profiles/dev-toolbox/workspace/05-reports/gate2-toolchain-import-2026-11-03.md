# Gate 2 验证报告 — toolchain-import

检查日期：2026-11-03 | 范围：rustup/uv 只读探测器、rust Inspect 与登记、Cargo/rust-toolchain 项目识别、前端工具清单扩展 | 设计：gate1-toolchain-import-2026-11-03.md

## 检查结果

| 检查项 | 结果 | 依据 |
|---|---|---|
| GR-01 激活链规则扫描 | PASS | 产品纪律逐条：只读零执行（未调用任何 rustup/uv/rustc 进程）、不夺所有权（零写 ~/.rustup 与 uv 目录）、登记显式确认；go-core GC-01..09 过 |
| GR-02 编译通过 | PASS | `go build ./...` + `GOOS=linux go build ./...` 双过（本机 Windows 交叉预检先行） |
| GR-03 新接口完整链路测试 | PASS | 沙箱测试：rustup 双工具链（semver exact / channel unknown+默认标记）、uv 目录过滤与版本解析、rust 登记与旧状态 rust 键自愈；cargo/rust-toolchain 三种钉子解析（project 包新增 3 测试） |
| GR-04 测试编译通过 | PASS | `go test ./internal/... -count=1` 全绿 |
| GR-05 前后端契约对齐 | PASS | 零新增操作；前端两处工具清单与 mock 均加入 rust；候选字段（source=rustup/uv、evidence channel/default）与 D3 一致 |
| GR-06 对齐记录 | PASS | 本报告 + D3 契约文档 |
| 真机验证（TH-10） | PASS | Windows 真机：discover 输出 rust（channel:stable + rustup default 标记）与 uv python 3.13.5；GUI 截图复核候选行与全局默认 rust 行；deck-loop src-tauri 识别为 cargo 模块 |
| Linux 真机 | **WARN** | 探测器 Linux 默认路径按官方文档规则实现、CI 编译通过，但未在真实 Linux 桌面验证（路径错误表现为安静无候选，无副作用） |

## 顺手修复（真机数据暴露的既有缺陷）

- golang 版本带 `\ntime` 尾巴（VERSION 多行文件未取首行）、java 版本/厂商带 `\r"`（release 文件 CRLF）——staticVersion 已修，真机复验 `1.27.3` / `21.0.6` 干净。

## 判定

**WARN**（仅 Linux 真机项）— 自动化与 Windows 真机链路全部通过；Linux 探测路径待真实发行版验证，不阻塞交付。后续如需闭环：在任一 Linux 机器跑 `dev-toolbox runtime discover` 确认 rustup/uv 候选出现。

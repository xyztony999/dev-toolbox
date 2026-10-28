# 安全说明

## 信任模型

DevToolbox 是本机单用户桌面工具：CLI 与桌面客户端调用同一 Go 应用服务，没有服务端、没有多租户、没有远程接口。安全边界在"本机用户 ↔ 系统/项目配置"，而不是"远程攻击者 ↔ 服务"。

## 路径参数是功能，不是漏洞

静态分析工具会把"函数接受路径参数并读取文件"标记为 path-traversal。本产品的多项核心能力就是让用户指定目录并**只读**采集证据：

- `runtime discover/inspect`：按用户指定目录（或常见安装位置）识别运行时，读取版本标记文件（如 JDK `release`）与预期二进制存在性（`bin/java.exe` 等），不执行任何文件。
- `project scan`：按用户指定根目录枚举清单文件。
- `git audit`：按用户指定仓库读取 `git log`。

这些读取以当前用户权限进行，不跨任何权限边界，也不以路径拼接执行命令（外部进程一律参数数组，见 `internal/gitidentity/gitrun.go` 的白名单与净化）。因此上述入口被标记的 path-traversal 属于**接受的设计**。

## 实际生效的边界

- 桌面桥接只开放白名单操作（`internal/desktop/desktop.go`），前端不能传任意命令或任意下载地址；安装下载地址由固定 provider 模板生成并做 host allowlist + 公网 DNS 校验。
- 凭据 token 只经 git credential 协议直通系统 helper，不落盘、不进日志与响应。
- 解包安装拒绝路径穿越/符号链接/超限体积；同版本目标冲突时不覆盖未知目录。
- 测试在隔离沙箱（临时 HOME/APPDATA/GIT_CONFIG_GLOBAL）运行，不改真实用户环境。

## 报告问题

发现漏洞请优先通过 GitHub Security Advisories（Security 页签）私密报告，避免公开 Issue。

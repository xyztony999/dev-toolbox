# D0 — 用户级运行时设置·Go 管理（go-management）

版本：v1.0 | 日期：2026-11-23 | 域：runtime（支撑） | 启动模式：增量 | 澄清：主会话 AskUserQuestion（用户勾选两项能力）

## 需求（澄清后）

在「用户级运行时设置」为 Go 增加两类管理能力，补齐现有"发现/安装/全局默认"之外的缺口：

1. **Go 环境配置管理**：以 `go env` 为事实源查看现状；显式设置用户级 `GOPROXY`/`GOPATH`/`GOBIN`（含国内代理预设与 direct 回退）；只读展示 `GOROOT`/`GOVERSION`/`GOMODCACHE`/`GOENV`/`GOOS`/`GOARCH`。
2. **Go 安装版本管理**：列出已登记/已下载的 Go 版本；切换全局默认（复用既有）；卸载/清理某个安装——产品自有（downloaded）目录可删，外部登记目录只注销不删。

## 边界与不变量（对齐 product-rules）

- `go env` 读写均以 go 自身为事实源；读=`go env -json`（只读内省，等同 git config 读取，不执行项目命令）；写=`go env -w/-u`，**显式用户动作**，预览+确认，不隐式改全局（rule 1/2/6）。
- 卸载仅清理本产品拥有的资产（rule 8）：`Source=="downloaded"` 且 `LocalHome` 位于 `RuntimeRoot()` 内才允许删目录；登记/legacy/user 外部安装只移除记录，绝不删外部目录。删除前预览列出将删路径。
- 不修改系统 PATH/注册表/shell profile（rule 2）；`go env -w` 只写 go 的用户 env 文件（GOENV），属 go 原生用户配置。
- token/凭据不涉及；env 值不含敏感信息，但 GOPROXY 可能含 userinfo 的私有代理——展示时不额外广播，仅本机桥接返回。
- 测试隔离 HOME/USERPROFILE/APPDATA/XDG_* 与 GOENV；不写真实 go env/PATH（rule 9）。
- 单一 Go 应用服务来源（rule 4）：CLI/GUI 复用同一 internal 服务。

## 不做

- 不管理 GOROOT 写入（只读展示，避免与安装目录冲突）；不做多版本并行 shim/自动切换；不执行 `go install`/`go build` 等项目命令；不接管 go 工具链所有权。

## 依赖

- userstate（State/Installation/Selection/SetGlobalDefault）、runtimeinstall（RuntimeRoot 归属）、session（golang 解析，仅参考）。
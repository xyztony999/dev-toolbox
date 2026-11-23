# API — Go 管理（go-management）

版本：v1.0 | 日期：2026-11-23 | 模块：go-management | 桥接前缀：`runtime.`（白名单注册）| 事实源：go env / userstate

## 服务（internal/goenv、internal/userstate 扩展）

### goenv.Read(ctx) → GoEnvReport
定位 go 二进制：优先 userstate 全局默认 golang 安装的 BinaryPath，回退 `exec.LookPath("go")`；都无 → `GO_NOT_AVAILABLE`。
执行 `go env -json`（只读）。返回：
```
GoEnvReport{ goBinary, goVersion, envFile,            // GOENV 路径
  values: map[string]string,                          // GOPROXY/GOPATH/GOBIN/GOROOT/GOMODCACHE/GOOS/GOARCH/GOVERSION...
  editable: ["GOPROXY","GOPATH","GOBIN"],
  presets: [ {id:"default",label:"官方 proxy.golang.org",value:"https://proxy.golang.org,direct"},
             {id:"cn",label:"goproxy.cn（国内）",value:"https://goproxy.cn,direct"},
             {id:"aliyun",label:"阿里云镜像",value:"https://mirrors.aliyun.com/goproxy/,direct"},
             {id:"direct",label:"direct（离线/直连）",value:"direct"} ] }
```

### goenv.Set(ctx, key, value, unset, dryRun, confirm) → GoEnvSetResult
- key 必须在 editable，否则 `GO_ENV_KEY_NOT_EDITABLE`。
- dryRun：只读当前值，返回 change，不写。
- 非 dryRun 需 confirm，否则 `CONFIRMATION_REQUIRED`；执行 `go env -w key=value`（unset 时 `go env -u key`），失败 `GO_ENV_WRITE_FAILED`。
- 返回 `{ change: GoEnvChange{key,from,to,action}, report: GoEnvReport(写后) }`。

### userstate.RemoveInstallation(id, expectedRevision, deleteFiles, dryRun, confirm) → RemoveResult
- 找不到 → `RUNTIME_INSTALL_NOT_FOUND`；revision 不符 → `RUNTIME_REVISION_CONFLICT`。
- 归属判定：`Source=="downloaded"` 且 `LocalHome` 在 `RuntimeRoot()` 内 → ownedByProduct=true。
- deleteFiles=true 且 !ownedByProduct → `RUNTIME_REMOVE_NOT_OWNED`（外部目录绝不删）。
- dryRun：返回预览不写。非 dryRun 需 confirm。
- 执行：ownedByProduct&&deleteFiles → os.RemoveAll(LocalHome)；移除记录；若被删的是全局默认 → 清除该 selection。返回 `{ state, removed, dirDeleted, dirPath, wasDefault }`。

## 桥接 op

| op | payload | 返回 |
|---|---|---|
| runtime.goEnvRead | {} | GoEnvReport |
| runtime.goEnvSet | {key,value,unset,dryRun,confirm} | GoEnvSetResult |
| runtime.removeInstallation | {runtime,id,expectedRevision,deleteFiles,dryRun,confirm} | RemoveResult/预览 |

复用：runtime.getUserState（列表+默认）、runtime.setGlobalDefault（切换默认）。

## 错误码
GO_NOT_AVAILABLE / GO_ENV_KEY_NOT_EDITABLE / GO_ENV_WRITE_FAILED / RUNTIME_INSTALL_NOT_FOUND / RUNTIME_REMOVE_NOT_OWNED / RUNTIME_REVISION_CONFLICT / CONFIRMATION_REQUIRED。

## 测试隔离
goenv 测试设 GOENV 指向临时文件、隔离 HOME/USERPROFILE/APPDATA/XDG_*；用真实 go（若 CI 无 go 则 skip 写用例，读用例可 mock）。RemoveInstallation 测试在临时 RuntimeRoot 造 downloaded/外部两类目录验证归属。
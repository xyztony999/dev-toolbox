# API — 运行时统一管理（toolconfig）

版本：v1.0 | 日期：2026-11-23 | 桥接前缀：`runtime.` | 事实源：各工具原生配置

## 类型
```
ConfigItem{ key, label, value, editable, source, presets:[{id,label,value}], note? }
ToolConfigReport{ tool, binary?, items:[ConfigItem], note? }   // note: 工具不可用/只读原因
ConfigChange{ key, from, to, action:set|unset }
ToolConfigSetResult{ change, report }
```

## 服务 internal/toolconfig
- Read(ctx, tool) → ToolConfigReport。按 tool 分派适配器；工具二进制不可用 → items 空 + note（不报错，UI 显示只读说明）。
- Set(ctx, tool, key, value, unset, dryRun, confirm) → ToolConfigSetResult。仅 editable 键可写；dryRun 只读预览；写需 confirm。

### 适配器
| tool | 读 | 写 | 可写键 | 预设 |
|---|---|---|---|---|
| golang | 委托 goenv.Read | goenv.Set | GOPROXY/GOPATH/GOBIN | 同 goenv |
| node | `npm config get registry` | `npm config set/delete registry` | registry | npmjs / npmmirror |
| python | `pip config get global.index-url` | `pip config set/unset global.index-url` | global.index-url | pypi / 清华 / 阿里云 |
| maven | 解析 ~/.m2/settings.xml 的 mirror | 托管 mirror(id=dev-toolbox) 插入/替换/移除，写前备份 | mirror.url | 阿里云 / 腾讯 / huawei |
| java | JAVA_HOME=默认安装 home（只读） | — | — | — |
| androidsdk | ANDROID_HOME=env 或默认安装 home（只读） | — | — | — |
| gradle | GRADLE_USER_HOME=env 或 ~/.gradle（只读） | — | — | — |
| rust | ~/.cargo/config.toml 的 replace-with（只读） | — | — | — |

二进制解析：node→默认安装目录的 npm(.cmd)；python→默认安装 python 执行 `-m pip config`；均回退 LookPath。maven/rust/java 等读文件/env，无需执行。

## 桥接 op
| op | payload | 返回 |
|---|---|---|
| runtime.toolConfigRead | {tool} | ToolConfigReport |
| runtime.toolConfigWrite | {tool,key,value,unset,dryRun,confirm} | ToolConfigSetResult |

复用：runtime.getUserState / setGlobalDefault / removeInstallation（版本管理泛化）。

## 错误码
TOOL_NOT_AVAILABLE / TOOL_CONFIG_KEY_NOT_EDITABLE / TOOL_CONFIG_WRITE_FAILED / TOOL_CONFIG_NOT_OWNED(maven 移除他人 mirror) / CONFIRMATION_REQUIRED。

## maven 托管 mirror 规则
settings.xml 不存在→创建含 <settings><mirrors> 骨架；存在→在 <mirrors> 内按 id=dev-toolbox 幂等替换，无则插入；unset→仅移除 id=dev-toolbox 块。写前复制 settings.xml.devtoolbox.bak（首次）。只解析/托管该 id，用户其它 mirror 原样保留。
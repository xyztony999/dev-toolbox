# PageSpec — 用户级运行时设置·Go 管理

版本：v1.0 | 日期：2026-11-23 | 模块：go-management | 入口：用户级运行时设置新增「Go 管理」页签

## 页签结构

「Go 管理」页签含两个分区，沿用既有 section/toolbar 语言与深色 token。

### 分区一：Go 环境配置
- 顶部来源行：当前 go 二进制路径 + goVersion + envFile（GOENV），只读。go 不可用 → a-empty「未找到 go，请先在『获取运行时』安装或『发现与登记』登记」。
- 只读概览：GOROOT / GOMODCACHE / GOOS / GOARCH（chip）。
- 可编辑三项（GOPROXY / GOPATH / GOBIN）各一行：label+键名、当前值、输入框、[预览] [应用]。
  - GOPROXY 行附预设下拉（官方/goproxy.cn/阿里云/direct），选择即填入输入框。
  - 预览（dryRun）→ 行内「当前 → 将写入」；应用需 confirm 弹窗「写入 go 用户配置（go env -w），只影响本机 go，不改系统环境」。
  - 清空值=unset（go env -u），确认文案说明"恢复 go 默认"。
- 失败回显错误条；GO_ENV_KEY_NOT_EDITABLE 等翻人话。

### 分区二：Go 版本管理
- 列表：userstate 中 runtime==golang 的安装，每行 version + label + source 标签（已下载/已登记）+ 默认标记 + 路径。
- 行操作：[设为默认]（复用 setGlobalDefault，confirm）；[卸载/注销]。
  - downloaded：弹窗预览"将删除目录 <path> 并移除记录"，confirm 后删目录+记录。
  - 外部登记：弹窗"仅移除登记记录，不删除外部目录 <path>"，confirm 后只注销。
  - 若为当前默认：提示"将同时清除全局默认"。
- 空状态引导去「获取运行时」/「发现与登记」。

## 状态与键盘
覆盖 go 不可用、无安装、写失败、revision 冲突、外部目录保护；表单可键盘；状态有可读文本；写操作均预览+确认。

## API 对齐
字段/错误码/confirm 与 api-go-management.md 一致；前端不推断归属，ownedByProduct 由服务端返回。
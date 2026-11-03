# PRD — 仓库 Git Remote 管理

版本：v1.0 | 日期：2026-11-03 | 模块：git-remote-management | 状态：Gate 1 待检查

## 1. 目标

在单仓库维度完整管理 git remote：查看全量信息、新增、删除、重命名、修改 fetch 地址；镜像推送既有能力整合进同一入口。主入口在项目详情页仓库卡片。

## 2. 功能需求

| ID | 需求 | 验收标准 |
|---|---|---|
| GR-01 | remote 列表 | 返回每个 remote 的 fetch url、pushurl 列表、镜像标记、http(s) 生效账号选择器 |
| GR-02 | 新增 | 名字校验 `^[A-Za-z0-9._-]+$`；url 协议白名单（http/https/ssh/git）；重名报 GIT_REMOTE_EXISTS |
| GR-03 | 删除 | 不存在的 remote 报 GIT_REMOTE_NOT_FOUND；确认弹窗展示其当前地址；删除后列表即时刷新 |
| GR-04 | 重命名 | `git remote rename` 语义（迁移全部 remote.* 键）；新旧名字同过校验 |
| GR-05 | 修改 fetch 地址 | 仅改 remote.<name>.url；同 url 白名单 |
| GR-06 | 镜像整合 | 既有 mirrorSet/mirrorUnset 行为不变，可在同一面板操作 |
| GR-07 | 就地入口 | 项目详情仓库卡片可打开管理面板并完成上述全部操作 |
| GR-08 | 无网络 | 全流程零网络请求；不做连通性测试 |
| GR-09 | 注入防御 | remote 名含非法字符（`..`、换行、空白、`/`）即拒；url 含换行即拒 |

## 3. 状态与流转

无状态机；每次操作返回操作后快照。

## 4. 更新记录

| 版本 | 日期 | 变更类型 | 说明 |
|---|---|---|---|
| v1.0 | 2026-11-03 | initial | remote 全生命周期管理与就地入口 |

# 迁移来源基线

记录时间：2026-10-27（Asia/Shanghai）。目的：锁定本次迁移审阅所依据的来源版本与工作区差异。来源目录仅只读访问。

| 仓库 | HEAD | 工作区差异 | 处理 |
|---|---|---|---|
| `D:/Project/devswitch` | `5e3fc4ada6725f0ae552994c3262b53ccade5730` | `git status --porcelain=v1` 为空；暂存/未暂存 diff 均为空 | 仅以此 HEAD 作为迁移来源基线；未修改 |
| `D:/Project/gitid` | `e33e7288596ead7a6ccc11d07ec6803b19a5e39d` | `.claude/.profile-mounts.json` 有 2 行未提交修改（2 insertions/2 deletions）；无其他状态 | 保留现有差异；未修改 |
| `D:/Project/dev-toolbox` | unborn HEAD（尚无提交） | 初始文件均未跟踪 | 本仓库工作区承载本次设计 |

后续迁移前如来源 HEAD 或差异变化，重新记录并对比；不得将上述快照当成来源仓库当前状态的永久保证。

# PageSpec — 用户级运行时设置·运行时（合并 tab）

版本：v1.0 | 日期：2026-11-23 | 模块：runtime-tool-config | 取代：原「全局默认」+「Go 管理」两 tab

## 结构
单 tab「运行时」= 每工具一个折叠分区（a-collapse），header 显示 工具名 + 当前默认摘要；body 三段：
1. 默认版本：该工具已安装版本按钮组「设为默认」（复用 setGlobalDefault），当前默认高亮。
2. 版本管理：版本行（version/source 标签/路径/默认标记）+「设为默认」「卸载|注销」（复用 removeInstallation，confirm 预览归属）。
3. 环境配置：toolConfigRead 的 items。
   - 可写项：label+键名、当前值、输入框、预设下拉（如有）、[预览][应用][恢复默认]；预览行内「当前→将写入」；应用 confirm 弹窗说明写到哪里（go env / ~/.npmrc / pip 配置 / settings.xml 托管块）。
   - 只读项：chip 展示 value + note 说明杠杆（如 JAVA_HOME「切换默认版本即改变；会话注入以此为准」）。
   - 工具不可用：分区内 a-empty + note，不报错打断整页。

## 交互与状态
- 进入 tab 并行加载 userstate + 各工具 toolConfigRead（失败单工具降级 note）。
- 写操作单飞（pending），其余按钮置灰；错误翻人话回显页顶。
- 折叠默认展开当前有"非默认/可写配置"的工具？保持默认全收起，header 摘要足够扫描；用户点开感兴趣的。
- 键盘可达、窄屏分区内堆叠。

## API 对齐
字段/错误码/confirm 与 api-runtime-tool-config.md 一致；归属与只读判定由服务端给出，前端不推断。
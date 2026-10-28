# 服务边界候选

尚无已实现服务或桥接接口。以下标识用于后续设计分工，不承诺 API 名称与签名。

- ProjectDiscovery：静态扫描项目、仓库、模块和版本证据；只读。
- RuntimeCatalog：本机安装、已选默认、provider 支持范围及约束匹配。
- RuntimeSelection：有作用域的计划/应用、shim 与会话环境。
- GitIdentity：档案、Git 实际配置、includeIf 绑定、镜像与凭据协议。
- ContextLauncher：启动指定上下文的终端/子进程。
- Diagnostics：聚合运行时、PATH、JDK/Wrapper 与 Git 诊断。
- LegacyMigration：旧配置探测、差异、备份、幂等导入及结果。

统一结果形态、错误码、事件和取消语义由 API 设计确定；不得在此伪造已有端点或服务地址。

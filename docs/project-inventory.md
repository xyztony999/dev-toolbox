# D:/Project 技术栈盘点

日期：2026-10-27（Asia/Shanghai）。扫描覆盖新项目之外的 20 个一级目录；目录可能包含多个仓库/子模块，不将其等同于 20 个独立可运行应用。

方法：ripgrep 枚举文件，读取 package.json、Go/Python/Java/Rust/原生构建清单；记录版本标记、锁文件和容器文件是否存在。排除依赖安装目录、构建产物、引擎 profiles/payload、虚拟环境等。扫描工具同时使用末位排除 glob 和路径段过滤，避免依赖目录中 `big.js` 等同名目录被正向 glob 重新包含。

机器可读证据：`project-inventory.json`，各条目路径相对其项目目录。归档与 MAVLink 附带清单标记为 archived/bundled，不作为同等新增语言需求。不运行项目脚本，不读取 .env、凭据或私有业务配置。

## 项目清单与影响

| 一级目录 | 证据与技术栈 | 对 DevToolbox 的影响 |
|---|---|---|
| 8 | package.json：React 18、CRA、Bootstrap、Firebase | Node 项目识别；框架名不等于 Node 版本要求 |
| 9 | package.json：React 18、CRA、Bootstrap、Firebase | 同上；与 8 分别登记，不猜测是否重复或仍活跃 |
| ai-project-engine | engine/cli 与 engine/gui：Node >=18、Vue 3、Vite；src-tauri/Cargo.toml：Tauri 2 | Node + Rust 的多模块组合；检测系统 WebView 依赖 |
| debateTimer | React 19、Vite 6；server/package.json：Express、MongoDB 驱动 | 前后端子模块；MongoDB 为外部依赖线索，不自动安装数据库 |
| deck-loop | client：Vue 3.5、Ant Design Vue 4.2、Vite 5、Tauri 2；server-node：NestJS 10；两侧 Node >=20 | Node/Rust 混合仓库；可复用现有 AntD 前端经验 |
| devswitch | go.mod：Go 1.24；Vue 3/Naive UI；保留 Python 旧实现 | Go 运行时管理和宿主为迁移来源；Naive UI 替换为 AntD |
| gitid | Node >=18 CLI；Electron 33、Vue 3、AntD 4 | Git 语义与测试迁移到 Go；不沿用 Electron 运行时 |
| kindle-calendar | requirements.txt：Flask、Pillow、YAML、requests 等；Python 源码 | Python 解释器与项目 venv 分层；仅凭依赖文件不推断 Python 精确版本 |
| Knowlecy | 多个 Poetry/FastAPI 服务；爬虫明确 Python >=3.11,<3.12；其他模块 ^3.11；MongoDB/Chroma 等库 | 不能只提供 Python latest；记录版本约束、Poetry 与外部依赖线索 |
| Kulaa | Maven/Spring Boot 2.7.15、java.version=1.8；Next 15.4.5、React 19.1 | JDK 8 声明与新 Node 前端并存；子模块独立配置 |
| Kulaa(Unimelb) | Java 与 TS/TSX 源码快照；未找到项目构建清单 | 标记“证据不完整”；不能从扩展名推断具体框架/运行时版本或可构建性 |
| onlineEducation | Gradle/Spring Boot 2.7.11、sourceCompatibility=17；React/CRA 客户端；Gulp/Sass 模板；有归档 | 同仓多构建方式；归档降权；编译目标与执行 JDK 分离 |
| pdfjs | 静态 HTML/JS；未发现构建清单 | 可作为静态项目；不能仅因目录名假定 npm/pdf.js 构建要求 |
| py32f403 | GCC/Makefile：arm-none-eabi-*、Cortex-M4；附带 MAVLink/Pymavlink | 先检测/导入 ARM 工具链和 Make；不把附带 C#/Java 生成器都列为首批运行时 |
| SEG2105 | Android Gradle Plugin 8.4.1、Wrapper 8.6、compileSdk 31、Java 编译目标 11 | Android SDK 与构建 JDK、Java 目标分离；优先项目 Wrapper |
| SEG2105F21Project_Group_9 | Android 工程、Wrapper 8.9、compileSdk 31、Java 编译目标 11 | 与另一工程独立读取 Wrapper，不能全局切一套 Gradle 代替 |
| SLS | Android AGP 8.3.1、Wrapper 8.4、compileSdk 34、Java 目标 1.8；两个 FastAPI/Poetry 服务 ^3.11 | Android + Python 多根项目；目标 Java 8 不等于 Gradle 启动用 JDK 8 |
| SWEN30006 | 两个 Java/Gradle 工程；project2 显式 Java toolchain 19 | 增加非 LTS JDK 需求；不能把 Java 固定成 8/11/17/21 列表 |
| TinyPost | React 19、TypeScript、Vite 5、浏览器 sql.js | Node 项目；sql.js 不等同于本机数据库服务器需求 |
| unigate | Maven/Spring Boot 3.5.16、Java 21；Vue 3.5、Vite 8、Node >=22.12.0 | 支持 Node 精确最低补丁约束；前后端独立运行时 |

## 盘点形成的优先级

1. 先覆盖 Node、Java、Python、Go，保留 Maven、Gradle、Android SDK 已有能力。Java 来源包括 1.8/11/17/21 声明或编译目标，以及显式 toolchain 19；不强行归一。
2. 优先实现项目/子模块与版本来源展示，识别 package.json engines、.nvmrc、pyproject、pom、Gradle toolchain 与 Wrapper。无版本声明时显示 unknown，而不是擅自选 latest。
3. 管理“解释器/工具链选择”；npm/pnpm/yarn/Corepack、Poetry/uv/venv、Maven/Gradle Wrapper 各自保留其职责，不全局重装或覆盖项目工具。
4. Rust/Tauri 是真实需求，安排检测、已有工具链导入、宿主依赖体检；Rust 自动安装/升级后置并遵守 rustup 的所有权。
5. 嵌入式工具链先检测/导入；Docker、MongoDB、PostgreSQL、MySQL、Redis 先作为只读依赖诊断扩展候选，不扩张为数据库/容器全生命周期管理器。

清单版本来自本机源码声明，不代表已安装、已验证兼容或安全版本。Java sourceCompatibility 是编译目标，Gradle JVM 与 Java toolchain 还需分别解析。^3.11 与 >=3.11,<3.12 也不能当作同一约束。后续跨平台解析器必须保留来源、约束原文与冲突信息。

## 后续验收样本

- unigate：Java 21 + Node >=22.12.0 双模块。
- Knowlecy/LiteratureCrawler：严格 Python 3.11 约束及项目隔离。
- deck-loop：Node >=20 + Rust/Tauri + Vue/AntD。
- SLS：Android Wrapper/JDK/Java 目标 + Python 多根组合。
- SWEN30006/project2：显式 Java toolchain 19。
- Kulaa(Unimelb)：缺失构建清单时不误报“已就绪”。

以上是真实样本的后续只读验收计划，尚未执行产品验收。

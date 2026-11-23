# D0 — 运行时统一管理（合并「全局默认」+「Go 管理」为单 tab，配置面泛化）

版本：v1.0 | 日期：2026-11-23 | 域：runtime | 启动模式：增量（go-management 之上） | 澄清：主会话多选（用户全选并追加 java/android）

## 需求（澄清后）
- 移除独立「全局默认」与「Go 管理」两个 tab，合并为单个「运行时」tab：每工具一个折叠分区，内含 默认版本 / 已安装版本管理 / 环境配置。
- 版本管理（切换默认、卸载/注销）泛化到全部工具（RemoveInstallation 本就通用）。
- 环境配置按各工具**原生事实源**，能写则显式写（预览+确认），不能安全写的只读+说明杠杆：
  - golang：go env（GOPROXY/GOPATH/GOBIN 可写；GOROOT 等只读）〔已有〕
  - node：npm registry（npm config get/set/delete，~/.npmrc）可写，含国内镜像预设
  - python：pip index-url（pip config get/set/unset）可写，含国内镜像预设
  - maven：~/.m2/settings.xml 托管 mirror（id=dev-toolbox，写前备份、幂等替换、可移除）可写
  - java：JAVA_HOME 只读（=默认 java 安装 home；杠杆=切换默认；会话注入以此为准）
  - androidsdk：ANDROID_HOME 只读（env 或默认安装 home）
  - gradle：GRADLE_USER_HOME 只读（env 或 ~/.gradle）
  - rust：cargo 镜像只读（~/.cargo/config.toml 的 replace-with；提示手工编辑，不代写 TOML）

## 边界（对齐 product-rules）
- 可写项均为工具原生用户级配置（go env / npm / pip / maven settings 托管块），显式动作+预览+确认；不写系统 PATH/注册表/shell profile（rule 2）。
- maven 写 settings.xml 前备份（rule 8 迁移/写旧配置的备份精神），仅托管 id=dev-toolbox 的 mirror，不动用户其它 mirror。
- java/android/gradle/rust 不代写：JAVA_HOME/ANDROID_HOME 属会话注入范畴（已有），cargo TOML 结构复杂不代改，只读+指引（rule 5/10 尊重所有权）。
- 子进程一律 CREATE_NO_WINDOW（GUI 派生控制台子进程锁屏挂起教训）。
- 测试隔离 HOME/USERPROFILE/APPDATA/XDG_*；maven/npm/pip 写用例用临时目录/假二进制或纯函数级测试（rule 9）。

## 不做
多 mirror 管理 UI、cargo/gradle 代写、系统级环境变量写入、per-project 覆盖。
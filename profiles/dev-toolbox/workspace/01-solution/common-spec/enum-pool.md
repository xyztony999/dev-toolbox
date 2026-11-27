# 业务枚举候选（draft）

工具种类继承来源：node、java、maven、gradle、androidsdk、golang、python。Rust 与 ARM 工具链为后续扩展候选。

作用域候选：global、directory、repository、session。具体编码、优先级与适用操作在 D1/D3 中定义，当前不可作为已发布接口常量。

Git 签名密钥生成（git-gpg-keygen）：`GitKeyType` / `GitKeyAlgorithm` / `GitKeyExpiry` 已在 `common-enums.md` 定稿（D3 v1.0）；与既有 `GitSigningFormat` 配合使用，无 code 冲突。

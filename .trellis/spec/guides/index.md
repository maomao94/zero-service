# 思考指南索引

> Guides 只帮助 AI 判断“开始前要想什么”。实现契约放在 `../backend/*.md`，不要在 guide 中复制 backend 细节。

## 路由表

| Guide | When to read | Output expected | Then read |
| --- | --- | --- | --- |
| [code-reuse-thinking-guide.md](./code-reuse-thinking-guide.md) | 新增工具、SDK、client、常量、配置、协议转换，或相似逻辑出现多次 | 找到可复用位置，决定复用、扩展还是保留服务私有 | [`go-zero-conventions.md`](../backend/go-zero-conventions.md)，必要时 [`directory-structure.md`](../backend/directory-structure.md) |
| [cross-layer-thinking-guide.md](./cross-layer-thinking-guide.md) | 任务跨 `.api` / `.proto`、Logic、Model/SDK、配置、外部系统、前端或消息协议 | 画清数据流、契约源、生成脚本、消费者和验证点 | [`go-zero-conventions.md`](../backend/go-zero-conventions.md)、[`error-handling.md`](../backend/error-handling.md)，SocketIO 读 [`socketiox-contracts.md`](../backend/socketiox-contracts.md) |
| [documentation-guide.md](./documentation-guide.md) | 修改 README、docs/、CONTRIBUTING 或文档索引 | 确认文档层级、保留内容、链接和重复清理范围 | 只读相关项目文档，不读 backend 代码规范 |
| [release-tagging-guide.md](./release-tagging-guide.md) | 打 tag、创建 GitHub Release 或发布版本 | 先形成 release plan，等待用户批准后再执行 | 需要 Git 操作时遵循用户批准和仓库规则 |

## Guide 使用规则

- 只读当前任务触发的 guide。
- Guide 产出是问题清单和路由决定，不是代码契约。
- 需要签名、字段、错误矩阵、Good/Base/Bad、测试断言时，回到 `../backend/` 的 canonical spec。
- 修改字段、枚举、常量、配置、Topic、Method 或错误码前，先搜索所有引用。
- 如果踩到稳定实现规则，更新 backend spec；如果只是新的思考检查项，更新 guide。

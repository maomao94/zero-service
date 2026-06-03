# Design: 优化 Trellis spec 文档

## Boundaries

本任务只整理 `.trellis/spec/**` 和任务规划产物。实施阶段不得修改业务代码、Trellis 脚本或 AI 平台配置，除非发现当前 spec 结构已经被脚本约束阻塞并获得用户确认。

## Spec Taxonomy

### `backend/` code-spec

用于实现前的可执行规范，内容应回答“怎么写才安全”：

- 具体命名、目录、代码生成流程。
- API/RPC/DB/infra/跨层协议的签名和契约。
- 错误矩阵、Good/Base/Bad cases、测试断言点。
- 禁止模式和正确示例。

### `guides/` thinking guide

用于实现前的思考清单，内容应回答“开始前要检查什么”：

- 搜索哪些现有实现。
- 哪些边界需要确认。
- 什么时候回到某个 backend spec。
- 不复制 backend spec 的契约细节。

## AI Loading-Cost Principles

- Minimize first read：每个入口文件只告诉 AI 任务应读哪些最小必要 spec，不把所有规则复制到入口。
- Maximize precision：同一规则只有一个 canonical source，避免 AI 在多个相似版本之间选择错误。
- Preserve executable detail：压缩原则性文字，保留签名、payload、错误矩阵、Good/Base/Bad、测试断言等能直接指导实现的内容。
- Split by decision boundary：只有当不同任务会读取不同部分时才拆分；如果拆分后所有任务仍必须全读，则保留同文件并压缩。
- Make routing explicit：新增或保留的每个 spec 都要在 index 中说明“何时读”，而不是只有文件名。

## Cleanup Strategy

1. Index first：保持 `backend/index.md` 和 `guides/index.md` 是入口。
2. One source of truth：重复规则只保留在最具体的专题 spec；通用索引只写短说明和链接。
3. Long-file handling：超过约 200 行的文件必须检查是否存在以下问题：
   - 多个场景堆叠在一个文件，导致读取成本过高。
   - 同一 7-section 模板重复但没有足够新增契约价值。
   - 指南型问题混入 code-spec。
   - 通用规则重复出现在 `coding-standards.md`、`quality-guidelines.md` 或专题文件。
4. Split only when useful：拆文件仅用于降低按需读取成本并提升规范命中准确性；如果拆分会制造更多跳转或导致任务仍要全读，则改为压缩原文件。
5. Preserve contracts：涉及协议、错误、并发、SocketIO payload、antsx Invoke 行为的关键契约不得删除，只能合并、重命名、压缩或迁移。

## Candidate Changes

- `socketiox-guidelines.md`：最可能拆分。保留包结构、常用 API、事件处理、并发禁忌；把多个具体 UpSocketMessage 场景整理为更短的 contracts 小节，按最佳实践可拆成 `socketiox-contracts.md`。主文件必须说明普通 SocketIO 开发读主文件，协议 payload/上下行契约开发再读 contracts 文件。
- `error-handling.md`：保留网关/RPC/错误工厂主路径，压缩重复模式示例；把错误码体系链接到 `../../../docs/error-codes.md` 和 `../../../third_party/extproto.proto`，避免引用不存在的 `code.md`。
- `antsx-invoke-guidelines.md`：保留核心签名、选型、错误处理、取消语义和测试要求；压缩内部设计决策和重复错误示例。
- `coding-standards.md`：定位为全局协作和安全规则。代码生成、命名、验证命令若已由 `go-zero-conventions.md` 或 `quality-guidelines.md` 承载，应改为链接或短摘要。
- `guides/*.md`：检查是否过度展开为教程；保留短清单和指向 backend spec 的链接。

## Compatibility

- `.trellis/scripts/common/packages_context.py` 会扫描 `.trellis/spec/` 下除 `guides` 外的目录作为 layer。本任务保留 `backend/` layer，不改变脚本发现模型。
- 新增或拆分的 backend 文件必须加入 `backend/index.md`，并写明触发条件、canonical source 和避免重复读取的路线。
- 新增或拆分的 guide 文件必须加入 `guides/index.md`，且只能作为短 checklist 和 spec 跳转入口。
- 相对链接统一使用同目录 `./file.md`、跨层 `../backend/file.md`，或从 backend spec 指向仓库文档的 `../../../docs/*.md` / `../../../third_party/*.proto`。

## Rollback Shape

- 每个文件清理应尽量独立，可通过 git diff 单独回退。
- 如果拆分效果不好，保留原文件结构，仅做去重和索引优化。

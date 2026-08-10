# Design: Refresh Repository Trellis Specs

## Current State

- 仓库是 `module zero-service` 的单 Go module，生产代码分布于 `app/`、`aiapp/`、`socketapp/`、`gtw/`、`facade/`、`common/` 和 `model/`。
- 当前 Spec 已采用 `backend/` 代码规范与 `guides/` 思考指南的两层结构，共包含目录、编码、质量、go-zero、契约生成、生命周期、错误、公共包、GORM、并发、消息、调度及主要领域规范。
- 初步占位搜索未发现模板文本，说明本任务重点是证据审计、边界校准与增量刷新，而不是首次填充模板。

## Evidence Model

每个 Spec 结论至少使用一种证据：

1. 契约源：`.proto`、`.api`、typed protocol、配置结构。
2. 实现：构造函数、Logic、store、client、scheduler、hook 或 adapter。
3. 测试：成功、失败、边界、并发或兼容性断言。
4. 项目文档：`README.md`、`docs/architecture.md`、`docs/development.md`。
5. 重复模式：多个服务或公共包中一致出现的结构。

单个偶然实现、Mock、历史快照和生成代码不能单独升级为全局规则。

## Audit Boundaries

- 基础架构：`README.md`、`docs/architecture.md`、`go.mod`、顶层源码目录、服务 `gen.sh`。
- 通用服务模式：代表性的 gRPC、HTTP/BFF、Socket.IO、AI 服务及其 `internal/` 分层。
- 公共基础设施：`common/` 中有稳定调用方和测试的 client、store、并发、消息、调度、GIS、协议与 AI 包。
- 领域链路：Trigger、ISP、IEC 104、DJI、GIS、实时事件和 AI/MCP。
- Spec 自身：文件集合、索引路由、相对链接、证据路径、占位文本、重复或冲突规则。

## Update Strategy

- 先建立“Spec 文件 -> 触发范围 -> 证据路径 -> 风险”的审计矩阵。
- 对每份现有规范抽样读取其主要证据文件和测试；失效路径必须修正，无法证实的强规则降级或删除。
- 只有出现清晰、可独立查找的稳定所有权边界时才新增 Spec；否则补入最接近的现有文件。
- 索引最后更新，避免导航先于正文变化。
- 不修改源码；若发现实现缺陷，只记录为研究结论或后续任务候选，不通过 Spec 虚构目标行为。

## Verification

- 比较 `backend/index.md`、`guides/index.md` 与磁盘实际 Markdown 文件集合。
- 搜索模板占位和空泛词，并人工判断正常语境下的 `TODO`/placeholder 引用。
- 校验 Spec 中仓库相对路径存在；通配符、符号名和命令示例单独人工审核。
- 检查新增规则的证据路径和跨文件术语一致性。
- 运行 `git diff --check`，并确认 diff 仅包含 Spec 和任务管理文件。

## Rollback

- 每份规范保持小范围、独立更新；若某条规则证据不足，保留原文或删除新增内容，不扩大到源码修改。
- 若审计发现现有两层结构不再适用，先更新设计并请求评审，不在实施中临时大规模搬移文件。

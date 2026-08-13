# 审计 Authorization 传播链路

## Goal

在不改变运行行为的前提下，建立 Authorization、身份 claims 和 auth context 从 HTTP/Socket.IO 入口，经 gRPC/MCP 到下游消费者的完整数据流与信任边界，为后续 typed context key、claim 规范化和传播策略实施提供证据。

## Requirements

- 枚举所有原始 Authorization 写入、读取、复制、验证、转发和存储位置，并标注入口、transport、目标服务和用途。
- 枚举 `user-id`、`user-name`、`dept-code`、`auth-type` 的来源和消费方，区分已验证身份、调用方提供 metadata 与进程内派生值。
- 识别 HTTP、Socket.IO、gRPC 和 MCP 边界的信任假设，确认下游是否重新验证 token 或直接信任传播 claims。
- 审计 MCP `_meta`、raw `_meta` context、日志、trace、错误和持久化路径是否可能暴露原始 token。
- 记录 gRPC 重复 Authorization、空值/非空值并存和身份 metadata 冲突的当前行为及安全影响。
- 按调用链分类：必须用户委托、只需 claims、应使用 service credential、无需认证传播。
- 提出目标传播模式、兼容迁移顺序、配置/灰度方式和回滚方案，但本子任务不实施代码或配置变化。
- 以用户确认的默认拒绝基线分类：原始用户 token 默认不跨服务传播，仅显式批准的 RPC/MCP 委托 allowlist 可标记为 `user-token`；其余链路归入 `claims-only`、`service-token`、`none` 或待业务 owner 确认。
- 明确后续三个子任务的依赖、输入决策和不可混合的行为变化。
- 保持现有 Authorization、gRPC metadata、MCP `_meta`、claim mapping、`b64:` 和 context 行为不变。
- 完成规划后必须等待用户开发前确认；审计产出完成后再次等待用户开发后/交付确认，确认前不得自动进入下一个子任务。

## Acceptance Criteria

- [x] 形成覆盖 HTTP、Socket.IO、gRPC、MCP 和关键业务消费者的传播矩阵，包含文件/符号证据。
- [x] 每条原始 token 跨边界链路都有来源、目标、用途、验证点、信任等级和推荐策略。
- [x] 明确哪些链路需要用户委托，哪些可改为 claims-only、service-token 或 none。
- [x] 重复/冲突 Authorization 和身份 metadata 的当前行为与建议行为均有记录。
- [x] token 在日志、trace、错误、raw `_meta` 和持久化中的泄漏面完成核查。
- [x] 提供兼容迁移、部署顺序、观测指标和回滚方案，不要求本任务修改生产代码。
- [x] 形成供 `typed-auth-context-keys`、`normalize-auth-claims`、`enforce-authorization-policy` 使用的决策输入。
- [x] 用户已审查并确认审计规划后才开展审计交付，审计结果也经用户确认后才归档。

## Out Of Scope

- 不修改任何 Go 生产代码、配置、proto、生成文件或部署清单。
- 不删除或收缩现有 Authorization 传播。
- 不实施 typed context key 或 claim normalization。
- 不升级 `b64:` metadata 编码协议。

## Key Decision

- 用户已确认采用默认拒绝基线：原始用户 token 默认不跨服务传播，只有明确进入 RPC/MCP 委托 allowlist 的链路才能使用 `user-token`。该决定仅约束本审计的分类和后续设计，本审计任务不修改运行行为。

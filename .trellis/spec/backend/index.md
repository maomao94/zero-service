# 后端开发规范索引

> 这是 AI 路由表。先按任务触发条件选择最小必要规范，不要默认全文读取所有文件。

## 读取顺序

1. 所有后端实现先读本索引。
2. 只读取任务命中的 `When to read` 行。
3. 跨层、错误码、协议或公共组件变更时，再读对应专题规范。
4. 稳定新规则回填到 canonical source，不在多个文件重复定义。

## 规范路由表

| 文件 | When to read | Canonical source for |
| --- | --- | --- |
| [coding-standards.md](./coding-standards.md) | 任意后端改动前，需要全局 AI 协作、安全、命名和 Git 边界 | 全局协作纪律、敏感信息边界、API/gRPC 命名摘要 |
| [go-zero-conventions.md](./go-zero-conventions.md) | 改 `.api`、`.proto`、Handler/Server、Logic、ServiceContext、配置或服务目录 | go-zero 分层、`gen.sh` 流程、ServiceContext、公共组件清单 |
| [directory-structure.md](./directory-structure.md) | 不确定代码应放在服务内部、`common/`、`model/`、`facade/`、`docs/` 时 | 顶层目录职责和落点判断 |
| [database-guidelines.md](./database-guidelines.md) | 改 model、SQL、事务、缓存、DB 读写或持久化格式 | 数据库访问、事务和缓存约定 |
| [error-handling.md](./error-handling.md) | 改 HTTP/gRPC 错误、错误码、错误包装、网关错误 handler 或日志传播 | 项目错误工厂、错误码行为、透传和包装规则 |
| [quality-guidelines.md](./quality-guidelines.md) | 完成实现前做质量门禁，或修改公共组件和生成代码 | 禁止模式、测试范围、验证策略、公共组件安全修复边界 |
| [logging-guidelines.md](./logging-guidelines.md) | 新增或修改日志，尤其是外部系统、高频路径、敏感数据路径 | 日志字段、脱敏、噪声控制 |
| [socketiox-guidelines.md](./socketiox-guidelines.md) | 改 `common/socketiox/` API、Session、房间、事件处理、容器或并发实现 | SocketIO 包 API、并发规则、禁止模式 |
| [socketiox-contracts.md](./socketiox-contracts.md) | 改 SocketIO 上下行 payload、事件名、`UpSocketMessage`、统计和房间分页协议 | SocketIO 跨层协议契约、payload、错误矩阵、测试断言 |
| [antsx-invoke-guidelines.md](./antsx-invoke-guidelines.md) | 使用或修改 `antsx.Invoke`、`InvokeAllSettled`、Reactor 并行编排 | Invoke 签名、选型、取消、panic 防护、测试断言 |
| [antsx-promise-guidelines.md](./antsx-promise-guidelines.md) | 使用或修改 `antsx.Promise`、并行组合（All/AllSettled/Race/Any）、`ReplyPool` | Promise 签名、四象限选型、泄漏防护、错误语义、ReplyPool 设计决策、Stats 统计 |
| [trellis-template-policy.md](./trellis-template-policy.md) | 修改 Trellis 模板、平台适配或用户数据区 | Trellis 模板更新和验证策略 |
| [iec104-control-commands.md](./iec104-control-commands.md) | 新增/修改 ieccaller 控制方向 gRPC 接口、core.go Send*Cmd、Kafka 广播 consumer | IEC 104 typed 命令全链路、typeId 映射、qualifier 约定、六层文件修改清单 |

## 外部契约链接

- [错误码规范](../../../docs/error-codes.md) 是用户可见 HTTP/gRPC 映射说明。
- [extproto.proto](../../../third_party/extproto.proto) 是项目错误码枚举和 proto extension 的源码。

## 开发前检查

- [ ] 读 `coding-standards.md`，确认全局协作、安全和命名边界。
- [ ] 若改 go-zero 服务或契约，读 `go-zero-conventions.md`，确认 `.api` / `.proto` -> `gen.sh` -> Logic 流程。
- [ ] 若改错误返回，读 `error-handling.md`，不要新建独立错误码体系。
- [ ] 若改 SocketIO 协议，先读 `socketiox-guidelines.md`，再读 `socketiox-contracts.md`。
- [ ] 若新增复用能力，先搜索 `common/` 和相邻服务，必要时读 `../guides/code-reuse-thinking-guide.md`。
- [ ] 只加载命中的专题规范，避免把本目录所有 Markdown 注入上下文。

## 交付检查

- [ ] 变更范围只覆盖任务需求。
- [ ] `.api` / `.proto` 变更已执行对应 `gen.sh` 并检查生成 diff。
- [ ] 错误码、日志、配置和敏感信息符合对应专题规范。
- [ ] 执行相关构建、测试或 `git diff --check`，未执行项说明原因。
- [ ] 如发现可复用稳定规则，更新 canonical source，而不是写到多个文件。

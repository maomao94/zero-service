# Backend 编码规范

## 目录

### 基础规范

- [coding-standards.md](./coding-standards.md) - 全局协作纪律、命名、安全和 Git 边界
- [go-zero-conventions.md](./go-zero-conventions.md) - go-zero 服务结构、代码生成、三层架构
- [directory-structure.md](./directory-structure.md) - 项目目录结构和模块划分
- [error-handling.md](./error-handling.md) - 错误处理策略和错误码规范
- [logging-guidelines.md](./logging-guidelines.md) - 日志级别、格式和上下文传递
- [quality-guidelines.md](./quality-guidelines.md) - 代码质量和审查标准
- [database-guidelines.md](./database-guidelines.md) - 数据库操作和 Model 层规范

### 通信与协议

- [socketiox-guidelines.md](./socketiox-guidelines.md) - SocketIO 包 API、Session、房间、并发规则
- [socketiox-contracts.md](./socketiox-contracts.md) - SocketIO 事件名、payload、跨层协议契约
- [messaging-guidelines.md](./messaging-guidelines.md) - 消息队列和异步通信规范
- [iec104-control-commands.md](./iec104-control-commands.md) - IEC104 控制命令协议

### 领域模块

- [drc-concurrency.md](./drc-concurrency.md) - DRC Manager 锁模型、锁顺序、字段保护、heartbeatCancel 所有权
- [antsx-invoke-guidelines.md](./antsx-invoke-guidelines.md) - Antsx 并行任务编排（Invoke/InvokeAllSettled）
- [antsx-promise-guidelines.md](./antsx-promise-guidelines.md) - Antsx Promise 异步结果容器与组合
- [antsx-replypool-guidelines.md](./antsx-replypool-guidelines.md) - Antsx ReplyPool 异步请求/应答池
- [mr-concurrency.md](./mr-concurrency.md) - go-zero mr 并发工具（Finish/MapReduce）分页+并发查询模式
- [bytex-contracts.md](./bytex-contracts.md) - Modbus 字节/寄存器工具包合约

### 前端/UI

- [uix-framework.md](./uix-framework.md) - UIX 框架规范

### 元规范

- [trellis-template-policy.md](./trellis-template-policy.md) - Trellis 模板策略

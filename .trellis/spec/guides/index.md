# 思考指南

> 这些指南用于在编码前补齐容易遗漏的跨层、复用和边界问题。只在任务相关时读取，不作为默认全文上下文。

## 可用指南

| Guide | Purpose | When to Use |
| --- | --- | --- |
| [Code Reuse Thinking Guide](./code-reuse-thinking-guide.md) | 查找已有实现，减少重复封装 | 新增工具函数、SDK、client、配置、常量或相似逻辑时 |
| [Cross-Layer Thinking Guide](./cross-layer-thinking-guide.md) | 梳理 API/RPC、Logic、Model、配置和外部系统边界 | 功能跨 3 层以上、契约变化或数据格式变化时 |

## 触发条件

### 跨层思考

- 功能涉及 `.api` / `.proto`、Logic、Model/SDK、配置、数据库或前端多个层次。
- 数据格式在 API、gRPC、数据库、消息队列、MQTT、SSE、Socket、AI Provider 之间转换。
- 多个服务或消费者依赖同一字段、状态或事件。
- 不确定某段逻辑应该放在 Logic、common、model、client 还是配置层。

### 复用思考

- 正在写与现有实现相似的代码。
- 同一模式出现 3 次以上。
- 正在新增工具函数、公共封装、常量、配置或客户端。
- 正在修改字段、枚举、状态机、Topic、Method、错误码或配置 key。

## 修改前规则

修改任何值、字段、枚举、常量、配置、Topic、Method 或错误码前，先用可用搜索工具查找所有引用，再决定是否需要同步修改。

## 使用方式

1. 编码前：只阅读当前任务触发的指南。
2. 编码中：发现重复、跨层或边界不清时回到指南检查。
3. 编码后：如果踩到稳定经验，沉淀到对应指南或 backend spec。

核心原则：少量前置思考，换取更少返工和更少跨层 bug。

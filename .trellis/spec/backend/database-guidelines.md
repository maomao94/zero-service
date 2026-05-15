# 数据库规范

> 涉及 MySQL、PostgreSQL、SQLite、TDengine、Redis、Kafka 或缓存时，先复用现有 model、client、cache、config 和 `common/` 封装。

## 基本原则

- 不在 Logic 中直接拼接连接串、账号、密码或环境参数。
- 数据库、Redis、消息队列和第三方客户端通过配置结构和 `ServiceContext` 注入。
- 先查现有 model、DAO、cache、client 和相邻服务实现，再新增持久化代码。
- 复杂数据流先画清：接口契约 → Logic → Model/SDK/Client → 存储或消息系统。
- 所有跨层调用传递 `context.Context`，便于超时、取消和链路追踪。

## Model 和生成脚本

项目提供模型生成脚本：

```bash
cd model
sh genModel.sh
sh genPgModel.sh postgres <table_name>
sh genModelSql.sh
```

- 使用脚本生成模型后，必须检查生成代码 diff。
- 生成代码非必要不手改；需要调整字段、索引或表结构时，优先改 SQL/schema 或生成配置。
- 业务逻辑不要绕过 model/client 直接访问底层连接，除非相邻模块已有同类模式。

## SQL 变更

- 表结构、初始化数据、修复数据等独立 SQL，应放入项目约定 SQL 目录；如果目标模块已有 SQL 目录，优先跟随模块现有位置。
- SQL 文件名建议：`yyyyMMdd-{需求号或Trellis任务号}-{简短说明}.sql`。
- SQL 内容要能和 Trellis task、Backlog 条目或变更说明关联，方便追踪上线影响。
- 不在 SQL、配置或日志中提交真实账号、密码、连接串、内网地址或对象存储配置。

## 查询和事务

- 简单 CRUD 优先复用生成 model 方法。
- 批量、事务和聚合查询先找同库同服务的既有写法。
- 需要事务时显式说明事务边界、提交条件和回滚条件，不把多个外部系统操作伪装成单数据库事务。
- Redis/cache 更新要明确缓存 key、TTL、失效策略和数据一致性边界。

## 常见错误

- 新增功能前未搜索已有 model/client/cache，导致重复封装。
- 为单个 Logic 私有逻辑创建过度通用的公共 DAO。
- 手写生成模型文件，后续生成时被覆盖。
- 忽略 `context.Context`，导致超时、取消和链路追踪失效。
- 将真实数据库连接、远程地址或账号写入示例、日志、文档或提交信息。

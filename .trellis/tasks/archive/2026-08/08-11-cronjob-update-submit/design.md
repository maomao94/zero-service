# CronJob 更新与提交接口设计

## Contract

在 `TriggerRpc` 中新增：

```protobuf
rpc UpdateCronJob(UpdateCronJobReq) returns (UpdateCronJobRes);
rpc SubmitCronJob(SubmitCronJobReq) returns (SubmitCronJobRes);
```

三种写接口使用独立消息。`UpdateCronJobReq` 是扁平完整配置，首字段 `job_id` 定位 Trigger 任务，后续配置字段与 Create/Submit 逐项对齐：

```protobuf
message UpdateCronJobReq {
  string job_id = 1;
  // 其余完整配置字段
}

message UpdateCronJobRes {
  string job_id = 1;
  string next_run = 2;
  string task_code = 3;
}
```

Update 只按 `job_id` 定位并保留持久化 `task_code`，调用方不能修改任务身份。Submit 只按 `SubmitCronJobReq.task_code` 定位，不接受 `job_id`。协议消息相互独立，共享的配置编译工具位于 Logic helper。

## Logic Boundaries

- 从现有 `CreateCronJobLogic` 提取服务私有的请求转换函数，统一执行 proto 校验、JSON 校验、`CompileSchedule`、`CronJobExtra` 序列化和 `TaskConfig` 构建。
- `CreateCronJobLogic` 继续只调用 `Insert`。
- `UpdateCronJobLogic` 先 `GetByID`，把原 `ID`、`TaskCode` 和 `Status` 写入新配置后调用 `Update`。
- `SubmitCronJobLogic` 先 `GetByCode`：找到则沿用 `ID`、`Status` 并 `Update`；未找到则 `Insert`。
- Submit 的 `Insert` 若返回 `ErrDuplicate`，再次 `GetByCode`：找到则更新，仍未找到则返回记录已存在，覆盖软删除记录占用唯一键的情况。
- Logic 只组合领域 Store 方法，不直接使用 GORM 或 SQL。

## State Ownership

- 请求拥有任务名称、规则、优先级、超时、payload 和 Trigger 扩展配置。
- 更新路径保留持久化任务的 `ID` 和 `Status`，因此不会隐式启用被禁用任务。
- DB Store 的 `Update` 继续拥有运行字段保护：保留成功历史和 `scheduled_time`，在途时不覆盖 lease `next_run`。
- 新规则编译出的 `NextRun` 仅在没有在途执行时立即写入；有在途执行时当前 lease 完成路径按 Store 既有语义收尾。

## Error Mapping

- 请求、JSON 或 RRULE 无效：参数错误。
- Update 目标不存在或已软删除：`Code__1_02_RECORD_NOT_EXIST`。
- Create 重复、Submit 遇到软删除占用：`Code__1_02_RECORD_ALREADY_EXIST`。
- Store 其他错误：`Code__1_02_DB`。

## Compatibility

- Proto 只新增 RPC 和消息，不修改已有字段号或现有 RPC。
- Submit 使用独立请求/响应，Create、Update、Submit 的字段校验与注释保持一致，旧客户端不受影响。
- 不迁移数据库，不改变 `task_code` 永久唯一契约。

## Concurrency

数据库唯一索引是并发首次提交的最终约束。Submit 使用“查询、尝试插入、冲突后二次查询并更新”的收敛流程，不通过数据库方言特定 Upsert SQL 实现，以保持现有 Store 抽象及 MySQL/PostgreSQL/SQLite 行为一致。

## Rollback

回滚新增 RPC、Logic、测试和生成产物即可；无数据库迁移或持久化格式变化。

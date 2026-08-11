# CronJob 更新与提交接口实施计划

## Implementation

- [x] 修改 `app/trigger/trigger.proto`，新增扁平 `UpdateCronJobReq/Res`、独立 `SubmitCronJobReq/Res`，并对齐三种写请求的字段、校验与注释。
- [x] 执行 `app/trigger/gen.sh`，审查生成的 pb、validate、server/logic 骨架和 OpenAPI diff。
- [x] 提取 Create/Update/Submit 共用的 CronJob 请求校验与 `TaskConfig` 构建逻辑，保持 Create 行为不变。
- [x] 实现 `UpdateCronJobLogic`：按 `job_id` 查询、保留原 task_code/ID/状态并调用现有 Store `Update`。
- [x] 实现 `SubmitCronJobLogic`：按 `task_code` 查询后 Insert/Update，并处理 Insert 唯一冲突后的二次查询收敛。
- [x] 补充 Logic 测试，覆盖严格创建、Update 成功/不存在/状态保持、Submit 创建/更新/软删除冲突及合法规则。
- [x] 补充 Store 状态所有权测试，确认配置更新不覆盖控制状态；未扩展 `common/crontask.TaskStore`。

## Validation

```bash
cd app/trigger && ./gen.sh
go test ./app/trigger/internal/logic ./app/trigger/internal/cronjob
go test ./common/crontask
go test -race ./app/trigger/internal/logic ./app/trigger/internal/cronjob
git diff --check
git status --short
```

## Review Gates

- [x] 生成文件只反映新增 RPC/消息，没有插件版本导致的无关重排。
- [x] `CreateCronJob` 行为和错误映射未发生回归。
- [x] Submit 唯一冲突后二次查询仍找不到时不会误恢复软删除记录。
- [x] Update/Submit 均保留持久化状态，且 Store 继续保护运行历史和 lease 字段。
- [x] 最终 diff 不包含数据库迁移、公共 Store 接口变化或无关重构。

## Rollback Point

Proto 生成后先确认生成 diff，再进入手写 Logic；若生成工具产生大范围无关变化，停止实现并先解决工具链一致性。

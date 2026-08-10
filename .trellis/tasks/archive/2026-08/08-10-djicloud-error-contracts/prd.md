# 规范 DJI Cloud 接口错误语义

## Goal

先规范 `app/djicloud/djicloud.proto` 的 RPC 响应契约和错误语义，再生成代码并调整 Logic 实现。明确区分 DJI 设备业务结果与平台/gRPC 调用错误，避免平台错误被编码成 `code=-1` 的正常响应。

## Requirements

- 保留 `CommonRes` 作为需要等待 DJI ACK 的命令响应，表达 `code`、`message`、`tid` 和 `reason_code`。
- DJI 设备已返回非零 `result` 时，使用 `CommonRes` 返回设备业务失败；调用链本身的参数、数据库、OSS、MQTT、超时和配置错误使用 gRPC error。
- DRC `drc/down` 即发即忘接口继续使用只表达 `seq` 的响应；发布失败使用 gRPC error。
- 平台自有查询和管理接口不通过 `CommonRes` 表达平台失败，平台失败使用 extproto gRPC error。
- `AckHmsAlert` 改为平台专用成功响应，不再复用含 DJI 错误字段的 `CommonRes`。
- 除明确共享的 `CommonRes` 外，RPC 专用响应类型统一使用 `<RpcName>Res`，修正平台接口中方法名与响应名不成对的问题。
- 自定义飞行区提交/删除接口保留包含 `tid/reason_code` 的响应，因为它们在平台数据库/OSS编排后调用 DJI `FlightAreasUpdate` 并等待设备 ACK；平台阶段失败使用 gRPC error，DJI ACK 失败使用包装响应。
- Proto 变更后使用 `app/djicloud/gen.sh` 生成代码；不得手工修改生成文件。
- 在实现阶段修正受影响 Logic 的错误返回，并补充成功、平台失败、设备 ACK 失败和边界输入验证。

## Acceptance Criteria

- [ ] `djicloud.proto` 明确标注 DJI ACK、DRC fire-and-forget 和平台自有 RPC 三类契约。
- [ ] `AckHmsAlert` 使用平台专用响应类型；其失败不再通过 `code=-1` 返回。
- [ ] `IsDeviceOnline`、`GetDeviceDetail`、`GetDeviceOsdSnapshot`、`GetDeviceStateSnapshot`、`GetFlightTaskProgressLast`、`QueryDrcStatus` 的响应类型与 RPC 名成对。
- [ ] 平台自有接口列表与响应类型、错误边界一致，且未误删自定义飞行区响应中的 DJI 关联字段。
- [ ] 生成代码与 proto 一致，`app/djicloud/gen.sh` 可重复执行。
- [ ] Logic 改造后平台错误返回 extproto gRPC error，DJI ACK 业务错误仍保留 `tid/reason_code`。
- [ ] 目标包测试、`git diff --check` 和生成文件审查通过。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.

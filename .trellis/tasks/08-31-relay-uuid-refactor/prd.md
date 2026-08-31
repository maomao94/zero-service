# 调整中继逻辑：UUID标识 + app:stream key

## Goal

将中继系统的 Redis key 从 `host_port/app/stream` 格式改为 `app:stream` 格式，同时引入 UUID 作为中继会话标识，防止 Asynq 回调误操作新中继任务。

## Background

### 当前实现
- **UID**: `host_port/app/stream`（如 `127.0.0.1_1935/live/stream`），由 `CanonicalUID()` 生成
- **Redis key**: `oryx:relay:state:{uid}`、`oryx:relay:lease:{uid}`、`oryx:relay:lock:{uid}`
- **Registry**: Sorted Set，member = uid
- **中继场景**: 同一 SRS 流平台地址，app + stream 唯一

### 问题
1. UID 包含 host:port 信息，但业务维度只需 app + stream 唯一
2. Asynq 回调（补拉/补停）时无法区分新旧中继任务，可能误操作

## Requirements

### 核心变更
1. **Redis key 改为 `app:stream` 格式**
   - `oryx:relay:state:{app}:{stream}`
   - `oryx:relay:lease:{app}:{stream}`
   - `oryx:relay:lock:{app}:{stream}`
   - Registry Sorted Set member 也改为 `app:stream`

2. **引入 UUID 作为中继会话标识**
   - 开启中继时生成 UUID，写入 `RelayState.UUID` 字段
   - Asynq 回调时校验 UUID 是否匹配当前 state 中的 UUID
   - 不匹配则跳过（说明已被新中继覆盖）

3. **开启中继校验**
   - 启动时检查 `state:{app}:{stream}` 是否存在
   - 存在则拒绝启动，要求业务侧先停止

4. **停止中继逻辑**
   - 通过 `app + stream` 构造 key 停止（保持原逻辑）

### 保持不变
- `StopRelayPull` 通过 app + stream 停止的逻辑
- `StopRelayByAppStream` 本地停止逻辑
- 广播停止逻辑
- ffmpeg 进程管理逻辑

## Acceptance Criteria

- [ ] 开启中继时 Redis key 格式为 `oryx:relay:state:{app}:{stream}`
- [ ] 开启中继时生成 UUID 写入 state
- [ ] 同一 app+stream 重复开启中继被拒绝（返回错误）
- [ ] Asynq 补拉回调时校验 UUID，不匹配则跳过
- [ ] Asynq 补停回调时校验 UUID，不匹配则跳过
- [ ] 停止中继通过 app+stream 正常工作
- [ ] Registry Sorted Set 使用 app:stream 作为 member
- [ ] 现有测试通过

## Out of Scope

- 修改 ffmpeg 进程管理逻辑
- 修改广播协议
- 修改业务层 API 接口参数

## Technical Notes

### 涉及文件
- `app/oryxserver/internal/relay/state.go` - Store、RelayState、key 生成
- `app/oryxserver/internal/relay/registry.go` - StartRelay、StopRelay、Reconcile
- `app/oryxserver/internal/logic/startrelaypulllogic.go` - 开启中继逻辑
- `app/oryxserver/internal/logic/stoprelaypulllogic.go` - 停止中继逻辑
- `app/oryxserver/internal/task/reconcile.go` - 补拉任务
- `app/oryxserver/internal/task/stop.go` - 补停任务

### 关键点
- `CanonicalUID()` 函数需要重写或移除
- `ParseTarget()` 保持不变（用于解析 app/stream）
- `RelayState` 结构体新增 `UUID` 字段
- `ReconcilePayload` 和 `StopPayload` 新增 `UUID` 字段

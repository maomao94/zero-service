# djicloud MQTT Hooks 规范

> `app/djicloud/internal/hooks/` 负责将 `djisdk.Client` 的 MQTT 上行分发到业务层：解析消息、写入数据库、推送 WebSocket。

## 文件组织

| 文件 | 处理的 Topic | 职责 |
|------|-------------|------|
| `register.go` | — | 集中注册所有 handler 到 `djisdk.Client`，依赖通过 `RegisterDjiClientOptions` 注入 |
| `sys_status_up.go` | `sys/.../status` | 处理 `update_topo` → 更新 `dji_device` + `dji_device_topo` |
| `telemetry_up.go` | `thing/.../osd` + `thing/.../state` | 设备 OSD 遥测 + State 快照 → 在线上报 + 快照 Upsert + WebSocket 推送 |
| `event_notify_up.go` | `thing/.../events` | 航线任务进度/就绪、返航信息、HMS 告警、远程日志进度、OTA、PSDK 自定义数据 |
| `mqtt_request_up.go` | `thing/.../requests` | 设备主动请求（组织绑定、飞行区查询），返回安全默认值 |
| `mqtt_drc_up.go` | `thing/.../drc/up` | DRC 上行（心跳 → 刷新 DRC 管理器 + WebSocket 推送，非高频 → 落库） |
| `store_helper.go` | — | 工具函数：时间转换、JSON 序列化、版本提取、任务状态文本 |
| `online_cache.go` | — | 在线缓存辅助函数 `IsOnline`、`OnlineValue` |

## 注册模式（register.go）

通过 `RegisterDjiClientOptions` 结构体注入所有外部依赖：

```go
type RegisterDjiClientOptions struct {
    DB                 *gormx.DB
    OnlineCache        *collection.Cache
    DrcManager         *drc.Manager
    PushCli            socketpush.SocketPushClient
    DisableOsdSQLTrace bool
}
```

在 `svc/servicecontext.go:129-135` 调用：

```go
hooks.RegisterDjiClient(djiCli, hooks.RegisterDjiClientOptions{
    DB:          db,
    OnlineCache: onlineCache,
    DrcManager:  drcMgr,
    PushCli:     pushCli,
    DisableOsdSQLTrace: c.Telemetry.DisableOsdSQLTrace,
})
```

**规则**：DB / OnlineCache / DrcManager / PushCli 都由 svc 层创建后注入，hooks 包内不持有全局变量。

## MQTT 上行 → DB 写入通用模式

每个 handler 遵循相同的 7 步流程：

1. **nil check** — data == nil 时直接 return
2. **Log receipt** — `logx.Infof` 或 `logx.Debugf`
3. **提取 gateway_sn** — 从 data.Gateway 获取，缺失则 log error 并 return
4. **时间转换** — `reportTime(data.Timestamp)`，为 0 时用 `time.Now()` 兜底
5. **构建 GORM struct + updateData** — upsert 必须的字段放入 struct，按条件更新的字段放入 map
6. **GORM upsert** — `Where(uniqueKey).Assign(updateData).FirstOrCreate(&record)`
7. **错误处理** — 只 log error，**永不 panic**，不阻塞 MQTT 回调

参考文件：
- `telemetry_up.go:54-73`
- `event_notify_up.go:62-101`
- `sys_status_up.go:42-105`

### GORM Upsert 签名说明

```go
c := db.WithContext(ctx)
deviceWhere := map[string]any{"device_sn": deviceSn}
updateData := map[string]any{"gateway_sn": gatewaySn, "is_online": true}
if err := c.Where(deviceWhere).Assign(updateData).FirstOrCreate(&device).Error; err != nil {
    // log only, never panic
}
```

`FirstOrCreate` 行为（GORM v2 `finisher_api.go`）：
- 记录不存在：用 struct 字段 + Assign map 创建
- 记录存在：仅更新 Assign map 中的字段

`Assign(map[string]any{})` （空 map）是安全的：`Updates(emptyMap)` 是 no-op，不会产生 UPDATE SQL。

## update_topo 处理（sys_status_up.go）

`NewStatusHandler` 在事务中执行以下步骤（参考 `sys_status_up.go:42-105`）：

1. **Upsert 网关设备自身**（`DjiDevice`）：`GatewaySn = gatewaySn`，标记在线上报时间
2. **清理过期的 topo 条目**：删除当前 gateway_sn 下不在新报告中的 sub_device_sn
3. **Upsert 每个子设备的 topo 条目**（`DjiDeviceTopo`）：`gateway_sn + sub_device_sn` 唯一
4. **Upsert 子设备记录**（`DjiDevice`）：
   - **Domain=0/1**（飞机/挂载负载）：不更新 `GatewaySn`，仅确保记录存在。蛙跳场景下绑定关系通过 `DjiDeviceTopo` 维护。
   - **Domain=3**（机巢/其他）：更新 `GatewaySn`，按原有逻辑覆盖。

### 蛙跳策略

飞机可能同时与多个机巢建立拓扑关系：
- `DjiDeviceTopo` 允许同一 `sub_device_sn` 出现在不同 `gateway_sn` 下
- `DjiDevice.GatewaySn` 对飞机（Domain=0）和挂载负载（Domain=1）**仅由 OSD/State 更新**（反映当前通信通道），update_topo 不覆盖
- 查询设备列表时通过 `topo_gateway_sn` 字段（`ListDevicesReq`）按拓扑绑定关系过滤

## OSD/State 处理（telemetry_up.go）

### OSD Handler
- 更新 `onlineCache`（内存 TTL=60s）中的设备 + 网关在线状态
- 更新 `DjiDevice`：`is_online=true`、`last_online_at`、`gateway_sn`
- Upsert `DjiDeviceOsdSnapshot`（按 device_sn 唯一）
- WebSocket 推送到 `thing/product/{device_sn}/osd` room（异步 goroutine，`context.WithoutCancel`）
- `disableOsdSQLTrace` 控制是否跳过 SQL 日志（高频写入场景）

### State Handler
- 提取 `firmware_version`、`hardware_version`（通过 `extractDeviceVersions`，marshal/unmarshal round-trip）
- 更新 `DjiDevice`：`gateway_sn`、版本（空值不上屏覆盖）
- Upsert `DjiDeviceStateSnapshot`
- WebSocket 推送到 `thing/product/{device_sn}/state` room

**注意**：State 不刷新在线状态（OSD 是唯一的在线刷新源）。

## 事件处理（event_notify_up.go）

| 事件方法 | Handler | 写入策略 |
|----------|---------|----------|
| `flighttask_progress` | `NewFlightTaskProgressHandler` | Upsert 2 表：`dji_dock_flight_task`（按 flight_id） + `dji_dock_device_flight_task_state`（按 gateway_sn） |
| `flighttask_ready` | `NewFlightTaskReadyHandler` | Insert-only 到 `dji_flight_task_ready` |
| `return_home_info` | `NewReturnHomeInfoHandler` | Insert-only 到 `dji_return_home_event` |
| `hms` | `NewHmsEventNotifyHandler` | Insert-only 逐条到 `dji_hms_alert` |
| `fileupload_progress` | `NewRemoteLogFileUploadProgressHandler` | Insert-only 到 `dji_remote_log_event` |
| `custom_data_transmission_from_psdk` | `HandleCustomDataFromPsdkEvent` | 仅 log |
| `custom_data_transmission_from_esdk` | `HandleCustomDataFromEsdkEvent` | 仅 log |
| `ota_progress` | `HandleOtaProgressEvent` | 仅 log |

## DRC Up 处理（mqtt_drc_up.go）

- 高频消息（`heart_beat`、`osd_info_push`、`hsi_info_push`、`delay_info_push`、`drc_initial_state_subscribe`）：不落库，只刷新 `drcMgr.OnDeviceHeartbeat` + WebSocket 推送
- 非高频消息：根据 `DrcUnmarshalUpData` 解析结果写入 `DjiDrcUpEvent`（Insert-only）
- 使用 `djisdk.DrcUpPayloadSummary` 记录简短摘要

## Request 处理（mqtt_request_up.go）

返回安全默认值，不依赖数据库：
- `airport_organization_get` → 空组织
- `airport_bind_status` → status=0
- `flight_areas_get` → 空列表
- 未知 method → success with nil output

## 设备在线管理

三层架构（参考 `svc/servicecontext.go:68-69` + `svc/device_online_refresher.go`）：

1. **内存缓存** `collection.Cache(dockOnlineTTL=60s)` — OSD 到达时 set，`SetOnlineChecker` + `IsDeviceOnline` 读取
2. **DB 字段** `DjiDevice.is_online` — OSD 时 set true
3. **Cron** `DeviceOnlineRefreshCron` — 每 15s 扫描，将 `last_online_at < now-60s` 的设备置为 false

参考文件：
- `internal/hooks/online_cache.go`
- `internal/svc/device_online_refresher.go`

## 常见陷阱

1. **Handler 必须是非阻塞的**：所有 hook handler 只在 panic/log error 后 return，不阻塞 MQTT 回调循环。WebSocket 推送应使用 `threading.GoSafe` 异步执行。
2. **`extractDeviceVersions` 的 marshal/unmarshal round-trip**：`state.Data` 是 `any`，嵌套结构可能导致静默空版本。
3. **`reportTime(0)` 回退到 `time.Now()`**：当设备上行 timestamp=0 或缺失时使用系统时间。
4. **State 不刷新在线状态**：不要参照 OSD 的做法在 State handler 中修改 `is_online`。
5. **update_topo 的 Domain 决定 GatewaySn 更新策略**：Domain=0/1 跳过覆盖，其他 Domain 按原有逻辑覆盖。修改 Domain 判断条件时须同步更新 `dji_device.go` 注释和 `djicloud.proto` 的 `DeviceInfo.gateway_sn` 说明。

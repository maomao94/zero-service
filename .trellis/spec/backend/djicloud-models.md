# djicloud GORM 模型规范

> `app/djicloud/model/gormmodel/` 定义大疆云平台业务的数据库表模型。

## 模型总览

| 表名 | 模型 | 写入策略 | 唯一键 |
|------|------|---------|--------|
| `dji_device` | `DjiDevice` | Upsert | `device_sn` |
| `dji_device_topo` | `DjiDeviceTopo` | Upsert + 全量替换 | `gateway_sn + sub_device_sn` |
| `dji_device_osd_snapshot` | `DjiDeviceOsdSnapshot` | Upsert | `device_sn` |
| `dji_device_state_snapshot` | `DjiDeviceStateSnapshot` | Upsert | `device_sn` |
| `dji_hms_alert` | `DjiHmsAlert` | Insert-only | auto increment |
| `dji_dock_flight_task` | `DjiDockFlightTask` | Upsert | `gateway_sn + flight_id` |
| `dji_dock_device_flight_task_state` | `DjiDockDeviceFlightTaskState` | Upsert | `gateway_sn` |
| `dji_flight_task_ready` | `DjiFlightTaskReady` | Insert-only | auto increment |
| `dji_remote_log_event` | `DjiRemoteLogEvent` | Insert-only | auto increment |
| `dji_return_home_event` | `DjiReturnHomeEvent` | Insert-only | auto increment |
| `dji_drc_up_event` | `DjiDrcUpEvent` | Insert-only | auto increment |
| `dji_fly_region` | `DjiFlyRegion` | Insert-only | `file_id`（单独）+ `bucket_name + file_name`（联合 idx_bucket_file） |
| `dji_fly_region_sync_status` | `DjiFlyRegionSyncStatus` | Insert-only | auto increment |

所有模型嵌入 `gormx.LegacyBaseModel`（int64 `id` + `create_time`/`update_time` + 软删除 `delete_time`/`del_state`，**不含 VersionMixin**）。

## 关键模型详解

### DjiDevice（dji_device.go）— 设备主表

```go
type DjiDevice struct {
    gormx.LegacyBaseModel
    DeviceSn        string       `gorm:"column:device_sn;uniqueIndex;not null"`
    GatewaySn       string       `gorm:"column:gateway_sn;index;not null;default:''"`
    Alias           string       `gorm:"column:alias;type:varchar(128);default:''"`
    GroupName       string       `gorm:"column:group_name;type:varchar(128);default:''"`
    FirmwareVersion string       `gorm:"column:firmware_version;type:varchar(64);default:''"`
    HardwareVersion string       `gorm:"column:hardware_version;type:varchar(64);default:''"`
    IsOnline        bool         `gorm:"column:is_online;index;not null;default:false"`
    FirstOnlineAt   sql.NullTime `gorm:"column:first_online_at"`
    LastOnlineAt    sql.NullTime `gorm:"column:last_online_at"`
}
```

**GatewaySn 语义**：
- 机巢自身：`GatewaySn = device_sn`
- 飞机/挂载负载（Domain=0/1）：仅由 OSD/State 上行更新（反映当前通信通道），update_topo 不覆盖
- 其他设备：由 update_topo 和 OSD/State 共同更新

**在线状态**：
- DB 默认 `is_online=false`
- 收到有效 OSD 后置为 true
- Cron 每 15s 按 `last_online_at < now-60s` 置为 false

**`TouchOnline(now)`** 方法（`dji_device.go`）：设置 `is_online=true`，首次时设 `first_online_at`，每次更新 `last_online_at`。

### DjiDeviceTopo（dji_device.go）— 拓扑关系表

```go
type DjiDeviceTopo struct {
    gormx.LegacyBaseModel
    GatewaySn        string `gorm:"column:gateway_sn;uniqueIndex:idx_topo_pair;not null"`
    SubDeviceSn      string `gorm:"column:sub_device_sn;uniqueIndex:idx_topo_pair;index:idx_topo_sub;not null"`
    Domain           string `gorm:"column:domain;type:varchar(8);not null;default:''"`
    SubDeviceType    int    `gorm:"column:sub_device_type;not null;default:0"`
    SubDeviceSubType int    `gorm:"column:sub_device_sub_type;not null;default:0"`
    SubDeviceIndex   string `gorm:"column:sub_device_index;type:varchar(32);default:''"`
    ThingVersion     string `gorm:"column:thing_version;type:varchar(64);default:''"`
}
```

**蛙跳支持**：`idx_topo_pair` 是 `gateway_sn + sub_device_sn` 组合唯一，**不是** `sub_device_sn` 唯一。允许同一架飞机出现在多个机巢的 topo 中。

**全量替换策略**（`sys_status_up.go`）：处理 update_topo 时，先删除当前 gateway_sn 下不在新报告中的子设备条目，再 Upsert 剩余条目。

### Snapshot 表（dji_osd_state.go）

`DjiDeviceOsdSnapshot` 和 `DjiDeviceStateSnapshot` 共用同一结构模式：

```go
type DjiDeviceOsdSnapshot struct {
    gormx.LegacyBaseModel
    DeviceSn   string    `gorm:"column:device_sn;uniqueIndex;not null"`
    GatewaySn  string    `gorm:"column:gateway_sn;index;not null;default:''"`
    ReportedAt time.Time `gorm:"column:reported_at;index;not null"`
    RawJSON    string    `gorm:"column:raw_json;type:jsonb;default:'{}'"`
}
```

**策略**：Upsert，每个 device_sn 只保留最近一次快照。不做历史时序。`RawJSON` 存储完整 payload 原始 JSON。

### Insert-only 事件表（dji_event.go）

6 个 Insert-only 表（`DjiHmsAlert`、`DjiFlightTaskReady`、`DjiRemoteLogEvent`、`DjiReturnHomeEvent`、`DjiDrcUpEvent`）：
- 主键为自增 auto increment
- **不修改/删除已写入的记录**（除 `dji_hms_alert` 的 `acked` 标记字段外）
- 用于审计溯源和排查

### Upsert 任务表（dji_event.go）

`DjiDockFlightTask`（唯一键 `gateway_sn + flight_id`）和 `DjiDockDeviceFlightTaskState`（唯一键 `gateway_sn`）：
- 每次新的 `flighttask_progress` 事件覆盖前一次记录
- `DjiDockDeviceFlightTaskState` 只保留每个机巢的最新一条进度

## 写策略选择原则

| 数据特征 | 策略 | 例子 |
|----------|------|------|
| 最新快照 | Upsert | 设备信息、OSD 遥测、State 状态、任务进度 |
| 状态机多实例 | Upsert by composite key | 多机巢 + 多飞行任务 |
| 不可变事件 | Insert-only | HMS 告警、返航事件、日志进度、DRC 上行 |
| 可确认事件 | Insert-only + 标记字段 | HMS 告警的 acked 标记 |

### DjiFlyRegion（dji_fly_region.go）— 飞行区配置主表

```go
type DjiFlyRegion struct {
	gormx.LegacyBaseModel
	GatewaySn    string `gorm:"column:gateway_sn;index;not null"`
	Name         string `gorm:"column:name;type:varchar(128);not null;default:''"`
	FileId       string `gorm:"column:file_id;type:varchar(64);uniqueIndex;not null"`
	BucketName   string `gorm:"column:bucket_name;uniqueIndex:idx_bucket_file;not null;default:''"`
	FileName     string `gorm:"column:file_name;uniqueIndex:idx_bucket_file;not null"`
	FileSize     int64  `gorm:"column:file_size;not null;default:0"`
	Checksum     string `gorm:"column:checksum;type:varchar(128);not null;default:''"`
	GeofenceJSON string `gorm:"column:geofence_json;type:text;default:''"`
}
```

**字段说明**：
- `FileId`：文件唯一标识（UUID），单独 uniqueIndex
- `FileName`：OSS 对象 key，格式为 `{geofence_type}_{fileId}.json`（如 `dfence_550e8400-xxxx.json`）
- `BucketName + FileName`：联合唯一索引 `idx_bucket_file`

**写入策略**：Insert-only。每次 submit 创建新记录；查最新按 `create_time DESC`。软删除用于 Delete 操作，GORM 自动过滤 `del_state=1` 的记录。

**FlightAreasGet 查询**：`WHERE gateway_sn = ? ORDER BY id DESC`（GORM 自动过滤软删除），返回该机巢所有活跃文件列表。

### DjiFlyRegionSyncStatus（dji_fly_region.go）— 同步状态记录表

```go
type DjiFlyRegionSyncStatus struct {
	gormx.LegacyBaseModel
	GatewaySn   string `gorm:"column:gateway_sn;index;not null"`
	FlyRegionID int64  `gorm:"column:fly_region_id;index;not null;default:0"`
	SyncStatus  string `gorm:"column:sync_status;type:varchar(32);not null;default:''"`
	SyncReason  int    `gorm:"column:sync_reason;not null;default:0"`
}
```

**写入策略**：Insert-only。每个 `flight_areas_sync_progress` 事件插入一条新记录，不更新已有记录。此表仅作同步历史追溯，Submit/Delete 不操作此表。

**关联方式**：通过 `fly_region_id` 关联 `dji_fly_region.id`，`flight_areas_sync_progress` 事件按 `gateway_sn + file_name` 匹配对应的 `DjiFlyRegion`。

## 常见陷阱

1. **AirSense/PSDK 等"预留"模块没有对应模型**：当前只 log，没有 DB 写入。添加时注意不要产生孤立写入。
2. **`DjiDevice` 的 `GatewaySn` 默认 `''`**：Domain=0/1 的设备不会在 update_topo 时得到 GatewaySn，查询时需通过 `DjiDeviceTopo` 反查。
3. **软删除恢复**：update_topo 使用 `gormx.Restore`（`sys_status_up.go`）恢复被软删除的 topo 条目，注意不要引入重复唯一键。
4. **Snapshot 的 `RawJSON` 字段类型**：虽然 GORM tag 写的是 `jsonb`，实际在 MySQL/GaussDB 中是 `text` 或 `json`，取决于方言。不要依赖 JSON 查询功能。
5. **`FirstOnlineAt`/`LastOnlineAt` 为 `sql.NullTime`**：未上线过的设备这两个字段为 NULL，代码中需要使用 `.Valid` 判断。

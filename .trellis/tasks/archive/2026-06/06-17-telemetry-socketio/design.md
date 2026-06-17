# 设备遥测数据 SocketIO 推送 - 技术设计

## 架构概览

```
DJI 设备 → MQTT → djicloud → telemetry_up.go → 数据库
                                      ↓
                                 SocketPush gRPC → socketgtw → 前端
```

## 修改范围

### 1. `telemetry_up.go` - 核心推送逻辑

**文件**: `app/djicloud/internal/hooks/telemetry_up.go`

#### 1.1 NewOsdHandler 签名变更

```go
// Before
func NewOsdHandler(db *gormx.DB, onlineCache *collection.Cache, disableSQLTrace bool) func(...)

// After
func NewOsdHandler(db *gormx.DB, onlineCache *collection.Cache, pushCli socketpush.SocketPushClient, disableSQLTrace bool) func(...)
```

#### 1.2 NewStateTelemetryHandler 签名变更

```go
// Before
func NewStateTelemetryHandler(db *gormx.DB, _ *collection.Cache) func(...)

// After
func NewStateTelemetryHandler(db *gormx.DB, _ *collection.Cache, pushCli socketpush.SocketPushClient) func(...)
```

#### 1.3 推送逻辑（在数据库写入后）

参考 `mqtt_drc_up.go:55-69` 的模式：

```go
if pushCli != nil {
    pushCtx := context.WithoutCancel(ctx)
    threading.GoSafe(func() {
        reqId, _ := tool.SimpleUUID()
        room := "thing/product/" + deviceSn + "/osd"
        _, err := pushCli.BroadcastRoom(pushCtx, &socketpush.BroadcastRoomReq{
            ReqId:   reqId,
            Room:    room,
            Event:   "telemetry:osd",
            Payload: toJSONString(data.Data),
        })
        if err != nil {
            logx.WithContext(pushCtx).Errorf("[dji-cloud] socket push osd failed: sn=%s err=%v", deviceSn, err)
        }
    })
}
```

### 2. `register.go` - 参数传递

**文件**: `app/djicloud/internal/hooks/register.go`

修改 `registerTelemetryHandlers` 中的调用：

```go
func registerTelemetryHandlers(c *djisdk.Client, db *gormx.DB, onlineCache *collection.Cache, drcMgr *drc.Manager, pushCli socketpush.SocketPushClient, disableOsdSQLTrace bool) {
    c.OnOsd(NewOsdHandler(db, onlineCache, pushCli, disableOsdSQLTrace))
    c.OnState(NewStateTelemetryHandler(db, onlineCache, pushCli))
    // ...
}
```

### 3. `docs/socketio.md` - 文档更新

在 DRC 章节后添加新章节：

**房间规则**:

| 房间名格式 | 说明 |
|-----------|------|
| `thing/product/{deviceSn}/osd` | 设备 OSD 遥测数据（0.5HZ） |
| `thing/product/{deviceSn}/state` | 设备 State 状态数据（状态变化时） |

**事件列表**:

| 事件名 | 触发时机 | 方向 | 说明 |
|--------|---------|------|------|
| `telemetry:osd` | OSD 数据上报 | 设备 → 云 → 浏览器 | 设备遥测数据（位置、电量、速度等） |
| `telemetry:state` | State 数据上报 | 设备 → 云 → 浏览器 | 设备状态变更（固件版本、硬件版本等） |

**数据结构**:
- OSD payload: DJI OSD 协议原始 JSON
- State payload: DJI State 协议原始 JSON

## 推送时序

```
1. DJI 设备上报 OSD/State 数据
2. djicloud 接收并解析
3. telemetry_up.go 处理：
   a. 更新在线缓存（OSD）
   b. 写入数据库（设备记录 + 快照）
   c. 异步推送 SocketIO（新功能）
4. 前端通过房间订阅接收实时数据
```

## 错误处理

- 推送失败：记录 error 日志，不影响主流程
- pushCli 为 nil：跳过推送（未配置 SocketPush）
- 数据库写入失败：记录 error 日志，推送仍会执行（推送的是原始数据）

## 测试策略

1. 单元测试：验证推送逻辑在 pushCli != nil 时被调用
2. 集成测试：验证前端能通过房间订阅收到数据
3. 容错测试：验证推送失败不影响数据库写入

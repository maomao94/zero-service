# 设备遥测数据 SocketIO 推送

## 背景

当前 `telemetry_up.go` 中的 `NewOsdHandler` 和 `NewStateTelemetryHandler` 只负责将遥测数据写入数据库和更新在线缓存，不支持实时推送到前端。前端无法实时获取设备的 OSD 遥测数据和 State 状态变更。

## 需求

为设备遥测数据（OSD 和 State）添加 SocketIO 推送能力，使前端能够实时订阅特定设备的遥测数据流。

### 功能需求

1. **OSD 数据推送**：设备 OSD 数据上报后（0.5HZ），通过 SocketIO 推送到对应房间
2. **State 数据推送**：设备 State 数据上报后（状态变化时），通过 SocketIO 推送到对应房间
3. **房间命名规则**（与 MQTT topic 一致）：
   - OSD: `thing/product/{deviceSn}/osd`
   - State: `thing/product/{deviceSn}/state`
4. **事件命名规则**：
   - OSD: `telemetry:osd`
   - State: `telemetry:state`

### 非功能需求

1. **异步推送**：推送操作不应阻塞主处理流程
2. **容错处理**：推送失败只记录日志，不影响数据库写入
3. **可选配置**：未配置 SocketPush 时跳过推送（与现有 DRC 推送行为一致）

## 验收标准

1. [ ] `NewOsdHandler` 接收 `pushCli` 参数，数据库写入后异步推送 OSD 数据
2. [ ] `NewStateTelemetryHandler` 接收 `pushCli` 参数，数据库写入后异步推送 State 数据
3. [ ] 推送使用 `threading.GoSafe` + `context.WithoutCancel` 模式
4. [ ] 推送失败仅记录 error 日志，不影响主流程
5. [ ] 更新 `docs/socketio.md`，添加设备遥测数据推送章节
6. [ ] 代码通过 lint 和 typecheck

## 参考

- 现有 DRC 推送模式：`app/djicloud/internal/hooks/mqtt_drc_up.go:55-69`
- SocketIO 文档更新位置：`docs/socketio.md`

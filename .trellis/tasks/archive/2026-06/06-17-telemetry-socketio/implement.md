# 设备遥测数据 SocketIO 推送 - 实现清单

## 实现步骤

### 1. 修改 `telemetry_up.go`

- [ ] 1.1 添加 import：`socketpush`、`threading`、`tool`
- [ ] 1.2 修改 `NewOsdHandler` 签名，添加 `pushCli socketpush.SocketPushClient` 参数
- [ ] 1.3 在 OSD 数据库写入后添加 SocketIO 推送逻辑
- [ ] 1.4 修改 `NewStateTelemetryHandler` 签名，添加 `pushCli socketpush.SocketPushClient` 参数
- [ ] 1.5 在 State 数据库写入后添加 SocketIO 推送逻辑

### 2. 修改 `register.go`

- [ ] 2.1 更新 `NewOsdHandler` 调用，传入 `pushCli`
- [ ] 2.2 更新 `NewStateTelemetryHandler` 调用，传入 `pushCli`

### 3. 更新 `docs/socketio.md`

- [ ] 3.1 在 DRC 章节后添加"设备遥测数据推送"章节
- [ ] 3.2 添加房间规则说明
- [ ] 3.3 添加事件列表
- [ ] 3.4 添加数据结构说明
- [ ] 3.5 添加前端对接示例
- [ ] 3.6 更新版本历史

### 4. 验证

- [ ] 4.1 运行 `go build ./app/djicloud/...` 确认编译通过
- [ ] 4.2 运行 `go vet ./app/djicloud/...` 确认无警告
- [ ] 4.3 检查现有测试是否通过

## 关键代码片段

### 推送逻辑模板（参考 mqtt_drc_up.go:55-69）

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

## 回滚点

- 如果推送逻辑引入问题，可快速移除推送代码块
- 数据库写入逻辑不受影响

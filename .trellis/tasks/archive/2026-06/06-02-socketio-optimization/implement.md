# SocketIO 代码优化 - 执行计划

## P0：server.go 事件处理去重

### 步骤

1. 添加 `sendResponse` 公共函数
2. 添加 `parseRoomPayload` 公共函数（解析 SocketUpRoomReq）
3. 添加 `parseUpPayload` 公共函数（解析 SocketUpReq）
4. 重构 `EventJoinRoom` 使用公共函数
5. 重构 `EventLeaveRoom` 使用公共函数
6. 重构 `EventRoomBroadcast` 使用公共函数
7. 重构 `EventGlobalBroadcast` 使用公共函数
8. 删除 `randomUUID()` 和 `uuid` 导入
9. 修复 564 行日志参数

### 验证

```bash
cd /Users/hehanpeng/GolandProjects/zero-service
go build ./common/socketiox/...
go vet ./common/socketiox/...
```

## P1：container.go 去重与优化

### 步骤

1. 添加 `newSocketClient` 公共函数
2. 重构 `getConn4Etcd` 使用 `newSocketClient`
3. 重构 `getConn4Direct` 使用 `newSocketClient`
4. 重构 `updateClientMap` 使用 `newSocketClient`
5. 优化 `subset()` 为部分洗牌算法
6. 添加 `syncClientMap` 公共方法
7. 重构 `getConn4Etcd` 使用 `syncClientMap`
8. 清理注释掉的代码
9. 修复 `parseURL` 环境变量重复赋值

### 验证

```bash
cd /Users/hehanpeng/GolandProjects/zero-service
go build ./common/socketiox/...
go vet ./common/socketiox/...
```

## P2：最终检查

### 验证

```bash
go build ./socketapp/...
go vet ./socketapp/...
```

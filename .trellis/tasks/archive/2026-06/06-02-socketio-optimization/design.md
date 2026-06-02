# SocketIO 代码优化 - 技术设计

## 优化范围

`common/socketiox/` 包：`server.go`（829 行）、`container.go`（425 行）、`handler.go`（40 行）

## P0：server.go 事件处理去重

### 问题

`EventJoinRoom`、`EventLeaveRoom`、`EventRoomBroadcast`、`EventGlobalBroadcast` 四个处理函数结构相同，约 200 行重复代码。

### 设计

提取两个公共函数：

**1. `parseAndValidatePayload[T any]`** — 泛型解析+校验

```go
func parseAndValidatePayload[T any](ctx context.Context, session *Session, payload *socketio.EventPayload, requiredFields ...string) (*T, string, bool)
```

职责：
- extractPayload → 校验非空
- jsonx.Unmarshal → 校验格式
- 校验必填字段（通过反射或接口）
- 失败时自动 sendErrorResponse
- 返回：解析结果、reqId、是否成功

**2. `sendResponse`** — 统一响应

```go
func sendResponse(session *Session, payload *socketio.EventPayload, code int, msg string, data any, reqId string)
```

职责：
- 优先用 Ack 回复
- 无 Ack 时用 ReplyEventDown
- 消除所有 `if ack != nil { ack(...) } else { session.ReplyEventDown(...) }` 模式

### 约束

- 保持对外 API 不变（Server、Session 公开方法签名不变）
- 保持事件处理行为不变（日志、错误码、响应格式）

## P1：container.go 去重与优化

### 问题 1：gRPC 客户端创建重复（3 处）

**设计**：提取 `newSocketClient`

```go
func newSocketClient(c zrpc.RpcClientConf) socketgtw.SocketGtwClient
```

### 问题 2：subset() 效率低

**设计**：改为 Fisher-Yates 部分洗牌

```go
func subset(set []string, k int) []string {
    n := len(set)
    if k > n {
        k = n
    }
    for i := 0; i < k; i++ {
        j := rand.IntN(n-i) + i
        set[i], set[j] = set[j], set[i]
    }
    return set[:k]
}
```

### 问题 3：updateClientMap 与 Etcd 逻辑重复

**设计**：提取 `syncClientMap`

```go
func (p *SocketContainer) syncClientMap(addrs []string, c zrpc.RpcClientConf)
```

被 `getConn4Etcd` 和 `updateClientMap` 共同调用。

## P2：清理

- 删除 `randomUUID()` 及 `uuid` 包导入
- 修复 564 行日志参数缺失
- 清理 container.go 注释掉的代码
- 修复 parseURL 环境变量重复赋值

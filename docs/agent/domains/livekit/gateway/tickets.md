# LiveKit 入会票据

修改票据生成、Redis 存储、一次性消费、有效期票据或免鉴权票据入会时读取。

## 票据契约

#### 票据类型

| 类型 | 值 | 说明 |
|------|---|------|
| 一次性票据 | 1（默认） | 消费后删除，只能使用一次 |
| 有效期票据 | 2 | 消费后保留至过期，可多次使用（挤掉旧设备） |

#### Redis 存储

- Key：`live:ticket:{ticket}`（单个票据 key）
- Value：JSON 字符串，包含会议号、身份、名称、过期时间、权限、票据类型
- TTL：由 `expire_seconds` 参数决定（秒）

#### 票据消费逻辑

```go
// 根据票据类型处理：一次性票据删除，有效期票据保留
if data.TicketType == 1 {
    // 一次性票据：删除 individual key
    l.svcCtx.Redis.DelCtx(l.ctx, ticketKey)
}
```

#### 过期时间校验

即使 Redis 有 TTL，也需要在代码中校验 `expireTime` 字段：

```go
// 校验票据是否过期（expireTime 格式：2006-01-02 15:04:05）
if data.ExpireTime != "" {
    expireT, err := time.ParseInLocation("2006-01-02 15:04:05", data.ExpireTime, time.Local)
    if err == nil && time.Now().After(expireT) {
        return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST, "票据已过期")
    }
}
```

## 验证

- `fakeLiveRpcCli`（webhook 测试用）必须实现 `live.LiveRpcClient` 的**全部**方法。gRPC 接口一旦新增/删除方法，`helpers_test.go` 里的 fake 需同步补齐/删除对应方法，否则 `go test ./...` 构建失败。
- 未鉴权路由（ticket 组）的 handler 在 `internal/handler/ticket/`，logic 在 `internal/logic/ticket/`，与 meeting 组分开。

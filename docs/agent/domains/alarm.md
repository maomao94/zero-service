# Alarm 飞书告警规范

## 适用范围

修改 `app/alarm` 的告警发送、IM 群管理、卡片渲染，或 `common/alarmx` 的 Lark SDK 封装时读取。

## 服务总览

`app/alarm` 是 go-zero gRPC 服务，负责将业务告警通过飞书 IM 发送。无数据库，Redis 仅用于 chat ID 缓存。

| 属性 | 值 |
| --- | --- |
| 端口 | 21011 |
| RPC 数量 | 2（Ping, Alarm） |
| Nacos | 否 |
| Interceptor | 否 |
| 持久化 | Redis（chat ID cache） |

依据：`app/alarm/alarm.go`、`app/alarm/alarm.proto`。

## 入口与 ServiceContext

```go
type ServiceContext struct {
    Config      config.Config
    Httpc       httpc.Service
    RedisClient *redis.Redis
    AlarmX      *alarmx.AlarmX
}
```

- `Httpc`：使用 go-zero `httpc.Service`，作为 Lark SDK 的底层 HTTP 客户端，赋予断路器能力。
- `AlarmX`：通过 `alarmx.NewAlarmX(larkClient, redisClient)` 创建，其中 `lark.Client` 的 HTTP client 使用 `AlarmxHttpClient` 包装 `httpc.Service`。
- `RedisClient`：用于缓存 chat ID，key 格式 `{appName}:alarm:{chatName}`，TTL 7 天。

依据：`app/alarm/internal/svc/servicecontext.go`。

## Alarm RPC 流程

```
AlarmReq
  → AlarmLogic.Alarm(in)
    → 从 config 合并 UserId（config 中预先配置的报警人）
    → 对 chatName+userId 去重
    → AlarmX.AlarmChat(ctx, alarmInfo)
      → Redis 查找 chatId（key: {appName}:alarm:{chatName}）
        → 命中：跳过创建
        → 未命中：CreateAlertChat（创建飞书群）+ Redis 缓存
      → UpdateAlertChat（确保报警人在群内）
      → SendAlertMessage（发送交互式卡片消息）
```

- 告警卡片使用 JSON 模板文件（`alarm.json`），含占位符 `${title}`、`${project}`、`${dateTime}`、`${alarmId}`、`${content}`、`${error}`、`${ip}`、`${button_name}`。
- 按钮名默认为 "关闭告警"，通过 `/solve` 命令处理，处理后将群名加 `[solved]` 前缀。

依据：`app/alarm/internal/logic/alarmlogic.go`、`app/alarm/alarm.json`。

## common/alarmx — Lark SDK 封装

- **`AlarmX`**：持有 `*lark.Client` + `*redis.Redis`，封装所有飞书 API 调用。
- **`AlarmxHttpClient`**：实现 `larkcore.HttpClient`，将 `httpc.Service.DoRequest` 适配为 Lark SDK HTTP 接口。
- **`AlarmInfo`**：告警字段结构体，调用方通过此结构传参，不在内部拼装字段。

### 核心方法

| 方法 | 职责 |
| --- | --- |
| `AlarmChat(ctx, alarmInfo)` | 编排：查 Redis → 创建/更新群 → 发消息 |
| `CreateAlertChat(ctx, alarmInfo)` | 调用 `ImChatCreate` 创建群，返回 chatId |
| `UpdateAlertChat(ctx, chatId, userIds)` | 调用 `ImChatMembersCreate` 加人 |
| `SendAlertMessage(ctx, chatId, alarmInfo)` | 调用 `buildCard` 渲染模板 → `ImMessageCreate` 发送 |
| `ImChatCreate/ImChatMembersCreate/ImMessageCreate/ImChatUpdate/ImChatGet` | Lark SDK 薄封装 |

### buildCard 卡片渲染

```go
func buildCard(ctx context.Context, alarmInfo AlarmInfo) (string, error) {
    // 1. 从文件路径读取 JSON 模板
    // 2. 替换 ${title} ${project} ${dateTime} ${alarmId} ${content} ${error} ${ip} ${button_name}
    // 3. 对替换值做 EscapeString（JSON 字符转义）
    // 4. 返回完整 JSON
}
```

- 模板文件路径从 Config 注入，不使用硬编码。
- 替换时 `EscapeString` 必须在每个字段值上执行，防止 JSON 注入。
- 新增占位符时：更新模板文件 → 更新 `AlarmInfo` 结构体 → 更新 `buildCard` 替换逻辑，三步同步。

依据：`common/alarmx/alarmx.go`。

## Scenario: 告警去重与群复用

### 1. Scope / Trigger

- 修改 `AlarmLogic.Alarm`、`alarmx.AlarmChat`、或 chat ID 缓存逻辑时适用。

### 2. Signatures

```go
func (l *AlarmLogic) Alarm(in *alarm.AlarmReq) (*alarm.AlarmRes, error)
func (x *AlarmX) AlarmChat(ctx context.Context, alarmInfo AlarmInfo) error
func (x *AlarmX) CreateAlertChat(ctx context.Context, alarmInfo AlarmInfo) (string, error)
```

### 3. Contracts

- `AlarmChat` 中 Redis key 为 `{appName}:alarm:{chatName}`，`appName` 来自 Config，`chatName` 来自 AlarmReq。
- Redis 缓存 TTL 7 天，过期后下次告警重新创建群，不设永久缓存。
- 同一 `chatName` 的多次告警复用同一个 chatId，不在每次 Alarm 时创建新群。
- `UpdateAlertChat` 将 config 的 UserId + AlarmReq 的 UserId 合并后去重，调用 Lark API 确保成员在群；已存在的成员不会报错。
- 告警去重仅针对 `chatName` + `userId` 组合，不改变 AlarmReq 的其他字段。
- `/solve` 命令走 `DoP2ImMessageReceiveV1` 路径，更新群名加 `[solved]` 前缀。该路径当前注释，启用时需确认 Lark Event 订阅配置。

### 4. Validation & Error Matrix

- Redis 不可用 → 缓存查询失败，返回 error，不执行 CreateAlertChat。
- Lark API 创建群失败 → 返回 error，不缓存 chatId，不发送消息。
- Lark API 发送消息失败 → 返回 error，chatId 保留在缓存中（群已创建）。
- `chatName` 为空 → 参数错误，不执行任何操作。
- Config 未配置 UserId → 参数错误，不创建群。

### 5. Good/Base/Bad Cases

- Good: 首次告警 `chatName="patrol-alert"` → Redis miss → 创建群 → 缓存 chatId → 发消息；第二次同 chatName → Redis hit → 跳过创建 → 更新成员 → 发消息。
- Base: Redis 有缓存但成员已变更 → UpdateAlertChat 确保新成员在群内，不影响已发送消息。
- Bad: Redis 查询失败直接 fallthrough 创建群 → 每次告警都创建新群，群数量无限增长。

### 6. Tests Required

- 断言首次告警创建群并缓存。
- 断言同 chatName 二次告警复用缓存，不创建新群。
- 断言 Redis 不可用时返回 error。
- 断言 Lark API 创建群失败时不缓存 chatId。
- 断言 chatName 为空时返回参数错误。

### 7. Wrong vs Correct

#### Wrong

```go
chatId, err := redisClient.Get(key)
if err != nil { // Redis error → fallback to create
    chatId, err = x.CreateAlertChat(ctx, alarmInfo)
}
```

#### Correct

```go
chatId, err := x.RedisClient.Get(key)
if err != nil {
    return err // Redis error → fail fast, no fallback
}
if chatId == "" {
    chatId, err = x.CreateAlertChat(ctx, alarmInfo)
    if err != nil {
        return err
    }
    x.RedisClient.Setex(key, chatId, 7*24*3600)
}
```

依据：`common/alarmx/alarmx.go`、`app/alarm/internal/logic/alarmlogic.go`。

## 反模式

- 在 `buildCard` 外手工拼 JSON 字符串发送卡片。
- Redis 查询失败直接创建新群（导致群爆炸）。
- 修改 Lark 模板文件后不同步更新 `buildCard` 的占位符替换逻辑。
- 在 alarm logic 中直接调用 Lark SDK，绕过 `alarmx.AlarmX` 封装。
- Config 的 `UserId` 只在初始化读取，运行时变更不生效（当前实现如此，若需热更新需重启服务）。
- 把 `AlarmxHttpClient` 替换为原生 `http.Client`，丢失断路器保护。

## 验证

- alarm logic 测试：`go test ./app/alarm/internal/logic`。
- alarmx 测试：`go test ./common/alarmx`。
- 集成验证：确保 `alarm.json` 模板文件存在于 Config 指定的路径。
- 新增 RPC 执行 `app/alarm/gen.sh`。

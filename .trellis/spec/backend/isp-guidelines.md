# ISP 协议接入指南

> ISP（Inspection Substation Protocol）= 区域型变电站远程智能巡视系统技术规范，`common/isp` 包提供协议编解码和常量定义。

## 何时阅读

- 新增 `ispagent` 业务 handler
- 对接上级巡检系统（ISP 协议服务端）
- 扩展 ISP 协议常量或消息类型

## 协议概述

传输层 TCP，帧格式：

```
0xEB90(2B BE) + SendSeq(8B LE) + RecvSeq(8B LE) + SessionSource(1B) + XMLLength(4B LE) + XML(UTF-8) + 0xEB90(2B BE)
```

- 0xEB90 大端，SendSeq/RecvSeq/XMLLength 小端
- `messageId = (Type << 16) | Command`
- Command=0 为上报类，Command≠0 为指令类
- XML 根元素可配置 `PatrolHost` / `PatrolDevice`

## 包结构

```
common/isp/
├── constants.go      # Type/Command/MessageID 常量 + 工具函数
├── message.go        # Message 结构 + gnetx 接口实现
├── xml.go            # XML 编解码（BuildXML/ParseXML）
├── serializer.go     # gnetx Codec 构造 + ISP Serializer
├── serializer_test.go
├── model_types.go    # DevicePointModel / PatrolDeviceModel 结构定义
├── model_writer.go   # WriteDeviceModel / WritePatrolDeviceModel 流式 XML 生成
└── model_writer_test.go
```

## 编解码

使用 `gnetx.LengthPrefixCodec`，带 `leadingBytes=0xEB90`、`trailingBytes=0xEB90`：

```go
codec := isp.NewCodec(rootName, maxFrameLength, debug)
```

- `stripBytes=2`：只剥前导，保留 21B 头给 Serializer
- `lengthOffset=19`、`lengthAdjust=2`：XMLLength 不含尾缀
- `debug=true`：启用 `gnetx.DebugSerializer`（debug 级别输出 hex）

## Message 模型

- `Identifiable` → `MessageID()` 供 Router 路由
- `Correlatable` → `TID()` = SendSeq 供请求-响应匹配
- `Response` → `ResponseTID()` = RecvSeq 供回包解回
- `SendSeq`/`RecvSeq` 对应协议帧 sendSerialNo/receiveSerialNo
- `RecvSeq` 为 ACK（上次收到的对端 SendSeq），出站时回执

## Item 约定

XML 中 `<Item attr="value"/>` 解析为 `map[string]string`，协议定义明确前保持动态。

## 注册响应

251-4 响应 Item 中的间隔属性均按秒解析；缺失或非法值保留默认/上一轮有效值，不应导致注册失败。

| 字段 | 用途 |
|------|------|
| `heart_beat_interval` | 只覆盖系统心跳间隔 |
| `patroldevice_run_interval` | 巡视装置运行数据周期上报间隔（→ `ReportCategoryPatrolDeviceRunData`） |
| `nest_run_interval` | 无人机机巢运行数据间隔（→ `ReportCategoryDroneNestRunData`） |
| `weather_interval` | 微气象数据间隔（→ `ReportCategoryEnvData`） |

心跳、上行周期上报、下游缓存新鲜度是独立概念，禁止用心跳超时直接替代上报间隔或缓存过期判断。

## 应答约定

所有 service→client 消息需回复 251-3 通用应答（Code 区分 `100/200/400/500`），handler 通过 `responseWithCode()` 构造。

## 常见错误

| 错误 | 说明 |
|------|------|
| Command=0 时输出 `<Command>0</Command>` | 应省略（`xmlMessage.Command` 使用 `omitempty`） |
| SendSeq/RecvSeq 字节序用混 | SendSeq/RecvSeq/XMLLength 一律小端 |
| 混淆 Type 共用 | Type=1 既是巡视设备状态上报也是机器人本体指令，由 Command 区分 |

## ispagent 服务约定

### 巡视装置上报缓存

`SendPatrolDeviceRunData`、`SendPatrolDeviceStatusData`、`SendPatrolDeviceCoordinates`、`SendDroneNestRunData`、`SendEnvData` gRPC 调用只写入本地内存缓存并返回本地受理成功，不同步等待上级 ISP 应答。

**当前已注册的 ReportCategory（`app/ispagent/internal/isp/reporting.go`）：**

| 常量 | messageId | 对应 gRPC |
|------|-----------|-----------|
| `ReportCategoryPatrolDeviceRunData` | 2-0 | `SendPatrolDeviceRunData` |
| `ReportCategoryPatrolDeviceStatusData` | 1-0 | `SendPatrolDeviceStatusData` |
| `ReportCategoryPatrolDeviceCoordinates` | 3-0 | `SendPatrolDeviceCoordinates` |
| `ReportCategoryDroneNestRunData` | 10004-0 | `SendDroneNestRunData` |
| `ReportCategoryEnvData` | 21-0 | `SendEnvData` |

- 周期上报由 `app/ispagent/internal/isp.Client` 在注册成功后按各上报类别自己的间隔发送 `CommandReport`；禁止把心跳间隔当作业务上报间隔。
- `patroldevice_run_interval` 只驱动巡视装置运行数据；`nest_run_interval` 驱动机巢运行数据；`weather_interval` 驱动环境数据。未在注册响应中给出的字段保持当前间隔不变。
- 默认间隔：坐标 2 秒（`noFreshCheck=true`），其余类别 1 分钟。可通过 `newReportManager` 选项覆盖，也可运行时通过 `SetInterval` 覆盖。
- 缓存模型必须按 report category 复用：category 映射 Type/Command，缓存 code/items/update time/expired/last sent。
- 缓存 key 由 `keyAttrsByCategory[category]` 定义：运行/状态/环境使用 `patroldevice_code + type`，坐标使用 `patroldevice_code`，机巢使用 `nest_code + type`。
- 上报时同一 XML `Code` 下聚合当前 category 的所有最新 Item。
- 发送前必须 snapshot 缓存，禁止持锁执行 TCP 请求。
- 下游长时间未刷新时按 `freshnessTimeout(report interval)` 判定 expired；非 `noFreshCheck` 类别在 2s tick 扫描时收集过期 key，释放读锁后短写锁删除，删除前必须按 `updatedAt` 二次校验，避免误删并发刷新。
- `noFreshCheck` 类别不做新鲜度清理，继续按类别间隔上报缓存旧值；当前巡视设备坐标属于此类。
- 新 `itemKey` 写入缓存时必须将对应 `category+Code` 的 `lastSent` 置零，使下一次 2s tick 立即上报完整快照；已存在 key 的刷新不能重置 `lastSent`，避免破坏上报间隔控频。
- 过期清理独立于上报间隔：即使 `lastSent` 尚未到期，也要在 tick 扫描时清理非 `noFreshCheck` 过期 item；清理后空的 `category+Code` 缓存槽应删除。
- 新增机巢、环境、微气象等上报类型时复用同一套 category + cache + interval + freshness 生命周期。

#### 巡视上报缓存测试契约

- 新 key 写入后，即使距离上次发送不足 interval，`dueReports` 也应返回该 `category+Code` 的快照。
- 已存在 key 刷新后，`dueReports` 在原 interval 未到期前应继续返回空。
- 非 `noFreshCheck` 过期 item 不应出现在 snapshot 中，并应从 `itemByKey` 清理。
- 同一 `category+Code` 下全部 item 过期后，应删除对应 `cachedReport`。
- `noFreshCheck` 类别即使超过 freshness timeout，也不应删除缓存 item。

#### 构造选项

`newReportManager(opts ...ReportManagerOption)` 支持按类别自定义初始间隔，零值使用默认：

```go
reports: newReportManager(
    ispclient.WithRunDataInterval(10 * time.Second),
    ispclient.WithCoordInterval(5 * time.Second),
    ispclient.WithNestRunInterval(30 * time.Second),
    ispclient.WithEnvDataInterval(30 * time.Second),
),
```

#### 运行时接口

`*Client` 对外暴露的上报控制接口（`app/ispagent/internal/isp/reporting.go`）：

| 方法 | 用途 |
|------|------|
| `CacheReport(ctx, category, code, items)` | gRPC 上报入口，非法 category 返回 error |
| `SetInterval(category, d)` | 运行时覆盖上报间隔，非正值忽略 |
| `SetNoFreshCheck(category, skip)` | 控制是否跳过新鲜度检查 |
| `ReportIntervals()` | 返回所有类别的当前间隔 |
| `CategoryNoFreshCheck(category)` | 查询类别是否跳过新鲜度检查 |

#### 上报链路关键模式

**`dueReports` 两阶段锁**（`reporting.go:233`）：
- RLock 扫描所有 category × code，收集过期 key 并 clone 到期快照
- RUnlock 释放读锁
- 有残留时短暂 Lock 调用 `deleteExpired` 清理（`updatedAt` 二次校验防并发误删）

**`freshItems` 元组返回**（`reporting.go:363`）：
```go
func freshItems(items, code, now, timeout) ([]isp.Item, []expiredReportItem)
```
一次遍历同时返回未过期 item 的 clone 和过期 key 列表。

**`markSent` 快照校验**（`reporting.go:316`）：
```go
func markSent(category, code, sentAt, snapLastSent)
// snapLastSent 是快照时刻的 lastSent，如果被并发 update 重置为零则跳过更新
if !snapLastSent.IsZero() && report.lastSent.IsZero() { return }
```

**新鲜度公式**（`reporting.go:344`）：
```go
freshnessTimeout = max(interval * 2, interval + 10s)
```

#### 新增 proto RPC

| RPC | 表 | 说明 |
|-----|-----|------|
| `SendDroneNestRunData` | O.40 | 机巢运行数据（→ `ReportCategoryDroneNestRunData`，10004-0） |
| `SendEnvData` | J.41 | 环境/微气象数据（→ `ReportCategoryEnvData`，21-0） |
| `ListReportIntervals` | — | 返回 `ReportCategoryInfo` 列表（含 category、name、interval、type、command、key_attrs） |

### 客户端生命周期

`app/ispagent/internal/isp/client.go` 是 TCP 长连接客户端，**不是 go-zero service**：

- `NewClient(cfg, store, db, uploader, provider, opts ...ClientOption)` — 构造即建连、启动轮询 goroutine
- `Close()` 取消 context + 关闭 gnetx client
- 通过 `proc.AddShutdownListener` 注册关闭，**不放入 `serviceGroup`**
- `serviceGroup` 只放 RPC server 和 `crontask.Scheduler`（二者都实现 `Start/Stop`）

**goroutine 拆分**（`client.go:288`）：

```go
func (c *Client) run() {
    go c.reportLoop()  // 独立 2s ticker，TCP 发送不阻塞主循环
    // 主 ticker 只负责注册+心跳
}
```

- `tick()` — 注册检查 + 心跳（2s ticker）
- `reportLoop()` — 上行缓存上报（独立 2s ticker），防 TCP 超时阻塞注册/心跳

**构造选项**（`ClientOption func(*ClientOptions)`，遵循 `coding-standards.md`）：

```go
NewClient(cfg, store, db, uploader, nil,
    WithReportOption(WithNoFreshCheck(categories...)),
    WithReportOption(WithCoordInterval(5*time.Second)),
)
```

**TCP handler 异步**（`connect()`）：

全部入站 handler 使用 `gnetx.HandleTypedAsync` / `FallbackFuncAsync` 注册，由 gnet worker pool offload 执行，不阻塞 eventloop。

源文件：
- `app/ispagent/ispagent.go` — serviceGroup 注册点
- `app/ispagent/internal/svc/servicecontext.go` — shutdown listener

### 汉化映射

所有 ISP 协议指令中文名称统一放在 `handler/names.go`，禁止散落在各 handler 文件中：

```go
// names.go
var taskControlName = map[int32]string{...}    // 任务控制指令名
var robotBodyName = map[int32]string{...}      // 机器人本体指令名
var modelTypeName = map[string]string{...}      // 模型类型名
```

`modelSyncCommandName` 引用 `modelTypeName` 避免重复定义。

### 巡视任务持久化

Cron 触发和 `HandleTaskControl` 都通过 `handler.UpsertPatrolTask` 写入 `GormIspPatrolTask`。使用 `FirstOrCreate` + `Assign` 模式，禁止 `clause.OnConflict`。

状态使用 `gormmodel.PatrolTaskStateXxx` 常量，禁止裸字符串 `"1"`/`"2"`。

源文件：
- `app/ispagent/internal/svc/cron_handler.go` — cron 持久化
- `app/ispagent/internal/handler/task.go` — 任务控制持久化

### carbon 时间格式化

`time.Time.Format("2006-01-02 15:04:05")` → `carbon.CreateFromStdTime(t).ToDateTimeString()`
`time.Time.Format("20060102150405")` → `carbon.CreateFromStdTime(t).Format("YmdHis")`

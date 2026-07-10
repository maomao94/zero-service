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
| `patroldevice_run_interval` | 巡视装置运行数据周期上报间隔 |
| `nest_run_interval` | 无人机机巢运行数据间隔，未实现具体上报时也应保留在本地间隔状态 |
| `weather_interval` | `PatrolDevice` / `PatrolHost` | 微气象数据间隔，未实现具体上报时也应保留在本地间隔状态 |

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

`SendPatrolDeviceRunData`、`SendPatrolDeviceStatusData`、`SendPatrolDeviceCoordinates` gRPC 调用只写入本地内存缓存并返回本地受理成功，不同步等待上级 ISP 应答。

- 周期上报由 `app/ispagent/internal/isp.Client` 在注册成功后按各上报类别自己的间隔发送 `CommandReport`；禁止把心跳间隔当作业务上报间隔。
- `patroldevice_run_interval` 只驱动巡视装置运行数据；状态数据、坐标/经纬度属于独立 ISP 上报类别，注册协议未给出对应频率时使用 report spec 中定义的默认 1 分钟间隔。
- 缓存模型必须按 report category 复用：category 映射 Type/Command，缓存 code/items/update time/expired/last sent。
- 缓存 key 必须全局唯一，至少包含 report category + XML `Code` + 协议 Item 唯一属性。当前巡视设备类：运行/状态使用 `patroldevice_code + type`，坐标使用 `patroldevice_code`；上报时同一 XML `Code` 下聚合当前 category 的所有最新 Item。
- 发送前必须 snapshot 缓存，禁止持锁执行 TCP 请求。
- 下游长时间未刷新时标记 expired；expired 或空 items 不上报，等待下一次 gRPC 更新清除 expired。
- 新增机巢、环境、微气象等上报类型时复用同一套 category + cache + interval + freshness 生命周期。

### 客户端生命周期

`app/ispagent/internal/isp/client.go` 是 TCP 长连接客户端，**不是 go-zero service**：

- `NewClient` 构造即建连、启动轮询 goroutine
- `Close()` 取消 context + 关闭 gnetx client
- 通过 `proc.AddShutdownListener` 注册关闭，**不放入 `serviceGroup`**
- `serviceGroup` 只放 RPC server 和 `crontask.Scheduler`（二者都实现 `Start/Stop`）

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

# Proto 日期时间格式审计

## 范围

- 审计仓库全部 36 个 `.proto` 源文件，其中 24 个项目协议文件、12 个 third-party 文件。
- 统计绝对日期时间字段；duration、timeout、interval、TTL 和 delay 不计为日期时间格式。
- 数量按字段声明统计。Proto 未注明格式时，仅在直接生产/转换代码可证明时列为“实现证据”。

## 已确认格式

| 格式 | Proto 明示 | 实现补充 | 合计 |
| --- | ---: | ---: | ---: |
| 秒级文本 `yyyy-MM-dd HH:mm:ss` | 38 | 46 | 84 |
| 3 位毫秒文本 | 3 | 0 | 3 |
| 目标为 6 位微秒文本 | 8 | 21 | 29 |
| Unix 秒 | 3 | 11 | 14 |
| Unix 毫秒 | 11 | 7 | 18 |
| Unix 微秒 | 0 | 0 | 0 |
| 日期 `yyyy-MM-dd` | 13 | 0 | 13 |
| 时间 `HH:mm:ss` | 0 | 2 | 2 |
| RFC3339 / RFC3339Nano | 0 | 0 | 0 |

另有协议专用文本：

- `yyyyMMddHHmmss`：`app/bridgedump/bridgedump.proto:33-41`。
- `yyyyMMddHHmmssSSS`：`app/bridgedump/bridgedump.proto:58-82`。
- 精确到纳秒但语法未指定：`app/bridgedump/bridgedump.proto:107-108`。

## 三类标准日期时间文本

### 秒级

- Trigger 大量输入、输出和调度字段明确使用 `yyyy-MM-dd HH:mm:ss`，代表证据见 `app/trigger/trigger.proto:493-507,1365-1408,1652-1693`。
- StreamEvent 调度字段见 `facade/streamevent/streamevent.proto:597-601,628-641`。
- Alarm 示例见 `app/alarm/alarm.proto:19`，zerorpc trigger time 示例见 `zerorpc/zerorpc.proto:28`。

### 3 位毫秒

Proto 明示的 3 个字段均为 LAL 上游会话开始时间：

- `app/lalproxy/lalproxy.proto:26-27`：`PubSessionInfo.startTime`，示例 `.586`。
- `app/lalproxy/lalproxy.proto:50-51`：`SubSessionInfo.startTime`，示例 `.724`。
- `app/lalproxy/lalproxy.proto:74-75`：`PullSessionInfo.startTime`，示例 `.123`。

它们是上游 LAL payload 字符串，当前不是本项目 Carbon 格式化生成。DJI 的 `ToDateTimeMilliString` 当前只用于 `common/djisdk/handler.go:24` 的诊断日志，不是 Proto 字段。

### 6 位微秒

- StreamEvent `MsgBody.time` 明确为 `YYYY-MM-DD HH:mm:ss.SSSSSS`、UTC+8：`facade/streamevent/streamevent.proto:113-114`。
- DJI 7 个 `reported_at` 字段明确为同一 6 位格式、UTC+8：`app/djicloud/djicloud.proto:1582-1823`。
- 结合直接生产代码，StreamEvent 的 MQTT/Kafka `send_time`、IEC event time、File `put_time`、Modbus copier 字段等也使用 Carbon micro 输出。

## Unix 与其他格式

- Unix 秒字段包括 Aichat completion `created`、LAL `unixSec`，以及实现可证明的 token/session/health 时间。
- Unix 毫秒字段包括 Aichat progress/filter、GIS fence、DJI execute/online、XFusion event/alarm 时间。
- 未发现项目 Proto 使用 Unix 微秒或 RFC3339/RFC3339Nano 作为确定 wire format。
- Trigger 有 13 个明确 `yyyy-MM-dd` 日期字段；ISP 有 2 个由严格校验代码证明的 `HH:mm:ss` 字段。
- 另有 19 个单位为毫秒的 timeout/interval/duration 字段，它们不是绝对日期时间，不能使用日期格式工具。

## 契约差异

- Proto 的 StreamEvent/DJI 注释要求固定 `SSSSSS`，但 Carbon v2.6.17 的 `ToDateTimeMicroString` 使用 `.999999` layout，会裁剪末尾零，当前实现不保证固定 6 位。
- DJI `QueryDrcStatusRes.started_at_millis` / `last_device_heartbeat_millis` 名称含 `_millis`，实际由 `timeString` 输出微秒日期时间文本；注释只说“时间字符串”，本任务不应据字段名机械改为 Unix 毫秒。
- Aichat `ModelObjectPb.created` 注释未写单位，当前实现为 Unix 毫秒，而同文件 completion `created` 明确为 Unix 秒；应保持各自现状。

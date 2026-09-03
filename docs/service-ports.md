# 服务端口清单

> 端口规则：统一 5 位数字，**1xxxx** = HTTP，**2xxxx** = gRPC。
>
> 以下端口来自各服务 `etc/*.yaml` 的默认配置。部署时如使用环境变量、Nacos 或容器编排覆盖配置，应以运行环境为准。

## HTTP 服务（1xxxx）

| 服务 | 端口 | 目录 | 说明 |
|------|------|------|------|
| gtw | 11001 | `gtw/` | BFF API 网关 |
| livegtw | 11002 | `app/livegtw/` | LiveKit 会议 HTTP 网关 |
| socketgtw | 11003 | `socketapp/socketgtw/` | Socket.IO 网关 HTTP 端 |
| oryxgtw | 11004 | `app/oryxgtw/` | Oryx 流媒体回调网关 |
| aigtw | 13001 | `aiapp/aigtw/` | AI HTTP 网关 |
| ssegtw | 13002 | `aiapp/ssegtw/` | AI SSE 网关 |
| mcpserver | 13003 | `aiapp/mcpserver/` | MCP HTTP 服务 |
| bridgegtw | 15001 | `app/bridgegtw/` | gRPC-Gateway 代理转发 |

## gRPC 服务（2xxxx）

### 核心业务

| 服务 | 端口 | 目录 | 说明 |
|------|------|------|------|
| zero.rpc | 21001 | `zerorpc/` | 核心业务 RPC |
| file.rpc | 21003 | `app/file/` | 文件 / OSS 服务 |
| ieccaller.rpc | 21004 | `app/ieccaller/` | IEC 104 主站采集 |
| iecagent.rpc | 21005 | `app/iecagent/` | IEC 104 代理管理 |
| trigger.rpc | 21006 | `app/trigger/` | 异步任务 / 计划任务 / CronJob |
| xfusionmock.rpc | 21007 | `app/xfusionmock/` | X-Fusion 模拟服务 |
| iecstash.rpc | 21008 | `app/iecstash/` | IEC 104 数据合并 |
| streamevent.rpc | 21009 | `facade/streamevent/` | 流事件处理与 TDengine 落库 |
| podengine.rpc | 21010 | `app/podengine/` | 容器管理引擎 |
| alarm.rpc | 21011 | `app/alarm/` | 告警服务 |
| djicloud.rpc | 21012 | `app/djicloud/` | DJI 云平台服务 |
| bridgekafka.rpc | 21013 | `app/bridgekafka/` | Kafka 桥接 |
| ispagent.rpc | 21014 | `app/ispagent/` | ISP 巡检协议代理 |
| ispserver.rpc | 21015 | `app/ispserver/` | ISP 巡检协议服务端 |
| oryxserver.rpc | 21016 | `app/oryxserver/` | Oryx 流媒体 API 代理 |
| live.rpc | 21017 | `app/live/` | LiveKit 会议管理 |

### AI 服务

| 服务 | 端口 | 目录 | 说明 |
|------|------|------|------|
| aichat.rpc | 23001 | `aiapp/aichat/` | AI 对话服务 |
| aisolo.rpc | 23002 | `aiapp/aisolo/` | AI Agent 服务 |

### 桥接与扩展

| 服务 | 端口 | 目录 | 说明 |
|------|------|------|------|
| socketgtw | 25001 | `socketapp/socketgtw/` | Socket.IO 网关 gRPC 端 |
| socketpush.rpc | 25002 | `socketapp/socketpush/` | Socket 推送服务 |
| bridgedump.rpc | 25003 | `app/bridgedump/` | 南瑞反向隔离装置 |
| bridgemodbus.rpc | 25004 | `app/bridgemodbus/` | Modbus TCP/RTU 桥接 |
| bridgemqtt.rpc | 25005 | `app/bridgemqtt/` | MQTT 桥接 |
| gis.rpc | 25006 | `app/gis/` | 地理信息服务 |
| logdump.rpc | 25007 | `app/logdump/` | 日志导出 |

## 端口段规划

| 端口段 | 用途 |
|--------|------|
| 11001 – 11004 | HTTP 网关 |
| 13001 – 13003 | AI HTTP 服务 |
| 15001 | 桥接 HTTP 网关 |
| 21001 – 21017 | 核心业务 gRPC |
| 23001 – 23002 | AI gRPC 服务 |
| 25001 – 25007 | 桥接与扩展 gRPC |

## 特殊端口

| 服务 | 端口 | 协议 | 说明 |
|------|------|------|------|
| iecagent | 12404 | TCP | IEC 104 设备通信 |
| ispserver | 7100 | TCP | ISP 下级设备接入 |

## 备注

- **socketgtw** 是混合型服务，同时暴露 HTTP（11003）和 gRPC（25001）。
- **lalhook**（原 11002）和 **lalproxy**（原 21002）已不推荐使用，服务保留但不再维护。流媒体能力已迁移至 [Oryx](./oryx/README.md)。
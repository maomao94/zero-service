# 服务端口清单

> 端口规则：统一 5 位数字，**1xxxx** = HTTP，**2xxxx** = gRPC。

## HTTP 服务（1xxxx）

### 11xxx — 通用网关

| 服务 | 目录 | HTTP 端口 | 说明 |
|------|------|-----------|------|
| gtw | `gtw/` | 11001 | BFF API 网关 |
| lalhook | `app/lalhook/` | 11002 | LAL 流媒体回调 |
| socketgtw | `socketapp/socketgtw/` | 11003 | Socket.IO 网关（混合型，同时有 gRPC） |

### 13xxx — AI 应用

| 服务 | 目录 | HTTP 端口 | 说明 |
|------|------|-----------|------|
| aigtw | `aiapp/aigtw/` | 13001 | AI 网关（OpenAI 兼容） |
| ssegtw | `aiapp/ssegtw/` | 13002 | SSE 网关 |
| mcpserver | `aiapp/mcpserver/` | 13003 | MCP 服务器 |

### 15xxx — 桥接网关

| 服务 | 目录 | HTTP 端口 | 说明 |
|------|------|-----------|------|
| bridgegtw | `app/bridgegtw/` | 15001 | gRPC-Gateway 代理转发 |

## gRPC 服务（2xxxx）

### 21xxx — 核心业务

| 服务 | 目录 | gRPC 端口 | 说明 |
|------|------|-----------|------|
| zero.rpc | `zerorpc/` | 21001 | 核心业务 RPC |
| lalproxy.rpc | `app/lalproxy/` | 21002 | LAL 流媒体代理 |
| file.rpc | `app/file/` | 21003 | 文件 / OSS 服务 |
| ieccaller.rpc | `app/ieccaller/` | 21004 | IEC 104 主站 |
| iecagent.rpc | `app/iecagent/` | 21005 | IEC 104 代理管理 |
| trigger.rpc | `app/trigger/` | 21006 | 异步任务 / 计划任务 |
| xfusionmock.rpc | `app/xfusionmock/` | 21007 | X-Fusion 模拟服务 |
| iecstash.rpc | `app/iecstash/` | 21008 | IEC 104 数据合并 |
| streamevent.rpc | `facade/streamevent/` | 21009 | 流事件服务 |
| podengine.rpc | `app/podengine/` | 21010 | 容器管理引擎 |
| alarm.rpc | `app/alarm/` | 21011 | 告警服务 |

### 23xxx — AI 应用

| 服务 | 目录 | gRPC 端口 | 说明 |
|------|------|-----------|------|
| aichat.rpc | `aiapp/aichat/` | 23001 | AI 对话 RPC |

### 25xxx — Socket / 桥接 / 扩展

| 服务 | 目录 | gRPC 端口 | 说明 |
|------|------|-----------|------|
| socketgtw | `socketapp/socketgtw/` | 25001 | Socket.IO 网关 gRPC 端 |
| socketpush.rpc | `socketapp/socketpush/` | 25002 | Socket 推送服务 |
| bridgedump.rpc | `app/bridgedump/` | 25003 | 南瑞反向隔离装置 |
| bridgemodbus.rpc | `app/bridgemodbus/` | 25004 | Modbus TCP/RTU 桥接 |
| bridgemqtt.rpc | `app/bridgemqtt/` | 25005 | MQTT 桥接 |
| gis.rpc | `app/gis/` | 25006 | 地理信息服务 |
| logdump.rpc | `app/logdump/` | 25007 | 日志导出 |

## 端口段规划

| 端口段 | 用途 |
|--------|------|
| 11001 – 11003 | 通用 HTTP 网关 |
| 13001 – 13003 | AI 应用 HTTP 服务 |
| 15001 | 桥接 HTTP 网关 |
| 21001 – 21011 | 核心业务 gRPC 服务 |
| 23001 | AI 应用 gRPC 服务 |
| 25001 – 25007 | Socket / 桥接 / 扩展 gRPC 服务 |

## 备注

- **socketgtw** 是混合型服务，同时暴露 HTTP 11003 和 gRPC 25001
- **iecagent** 额外监听 12404 端口用于 IEC 104 设备通信

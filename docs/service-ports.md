# 服务端口清单

## 网关 / HTTP 服务

| 服务 | 目录 | HTTP 端口 | 说明 |
|------|------|-----------|------|
| gtw | `gtw/` | 11001 | BFF API 网关 |
| lalhook | `app/lalhook/` | 11002 | LAL 流媒体回调 |
| socketgtw | `socketapp/socketgtw/` | 11003 | Socket.IO 网关（混合型，同时有 gRPC） |
| ssegtw | `aiapp/ssegtw/` | 11004 | SSE 网关 |
| bridgeGtw | `app/bridgegtw/` | 15002 | gRPC-Gateway 代理转发 |
| mcpserver | `aiapp/mcpserver/` | 8888 | MCP 服务器 |

## gRPC 服务

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
| bridgedump.rpc | `app/bridgedump/` | 25002 | 南瑞反向隔离装置 |
| bridgemodbus.rpc | `app/bridgemodbus/` | 25003 | Modbus TCP/RTU 桥接 |
| bridgemqtt.rpc | `app/bridgemqtt/` | 25004 | MQTT 桥接 |
| gis.rpc | `app/gis/` | 25005 | 地理信息服务 |
| logdump.rpc | `app/logdump/` | 25006 | 日志导出 |
| socketgtw | `socketapp/socketgtw/` | 25007 | Socket.IO 网关 gRPC 端 |
| socketpush.rpc | `socketapp/socketpush/` | 25008 | Socket 推送服务 |
| alarm.rpc | `app/alarm/` | 8080 | 告警服务 |

## 端口段规划

| 端口段 | 用途 |
|--------|------|
| 8080 | 告警服务（待统一） |
| 8888 | MCP 服务 |
| 11001 – 11004 | HTTP 网关层 |
| 15002 | gRPC-Gateway 代理 |
| 21001 – 21010 | 核心业务 gRPC 服务 |
| 25002 – 25008 | 桥接 / 扩展 gRPC 服务 |

## 备注

- **socketgtw** 是混合型服务，同时暴露 HTTP 11003 和 gRPC 25007
- **iecagent** 额外监听 12404 端口用于 IEC 104 设备通信
- **alarm.rpc** 端口 8080 与其他 gRPC 服务不在同一段，后续可考虑统一

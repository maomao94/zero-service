# Oryx 流媒体服务

基于 [Oryx](https://github.com/ossrs/oryx)（SRS 官方云平台）的流媒体管理服务，提供 HTTP 回调网关和 gRPC API 代理两层能力。

## 服务划分

| 服务 | 职责 | 文档 |
| --- | --- | --- |
| `oryxgtw` | HTTP 回调网关，接收 Oryx 五类回调并分发处理 | [oryxgtw](./oryxgtw.md) |
| `oryxserver` | gRPC 服务端，封装 Oryx/SRS HTTP API 供业务服务调用 | [oryxserver](./oryxserver.md) |

## 架构

```
Oryx/SRS ──HTTP 回调──> oryxgtw ──gRPC──> oryxserver
                              │                  │
                              v                  v
                         鉴权 / 分发        Oryx HTTP API
                                              │
                                              v
                                        SRS / FFmpeg
```

- `oryxgtw` 作为 Oryx 回调的统一入口，按 `action` 字段分发到不同处理逻辑
- `oryxserver` 封装 Oryx 管理 API 和 SRS HTTP API 为 gRPC 接口，供 Java/Go 业务服务通过 Nacos 调用
- 两者协作完成录制生命周期管理、流状态查询和中继拉流等能力

## 典型场景

1. **录制管理**：通过 `oryxserver` 配置录制策略，`oryxgtw` 接收 `on_record_begin`/`on_record_end` 回调落库
2. **流状态监控**：通过 `oryxserver` 查询 SRS 流列表、客户端信息和系统状态
3. **中继拉流**：通过 `oryxserver` 启动 FFmpeg 中继拉流，支持自动录制和时长限制
4. **回调鉴权**：`oryxgtw` 通过 `OryxHookAuth` 中间件验证回调请求合法性

# Oryx gRPC 服务端（oryxserver）

封装 Oryx/SRS HTTP API 为 gRPC 接口，供业务服务（Java/Go）通过 Nacos 调用，提供录制管理、流状态查询、中继拉流等能力。

## 服务信息

| 项目 | 值 |
| --- | --- |
| 服务名 | `oryxserver` |
| 协议 | gRPC |
| 部署模式 | `standalone` / `cluster` |
| 数据库 | PostgreSQL / MySQL / SQLite（录制记录落库） |
| Redis | asynq 任务队列 |

## API 分组

### 录制管理

| RPC | 对应 Oryx HTTP API | 说明 |
| --- | --- | --- |
| `RecordQuery` | `POST /terraform/v1/hooks/record/query` | 查询录制配置 |
| `RecordApply` | `POST /terraform/v1/hooks/record/apply` | 应用录制配置 |
| `RecordEnd` | `POST /terraform/v1/hooks/record/end` | 结束录制任务 |
| `RecordFiles` | `POST /terraform/v1/hooks/record/files` | 列出录制文件 |
| `RecordRemove` | `POST /terraform/v1/hooks/record/remove` | 删除录制文件 |
| `RecordGlobs` | `POST /terraform/v1/hooks/record/globs` | 更新录制 glob 过滤器 |
| `RecordPostProcessing` | `POST /terraform/v1/hooks/record/post-processing` | 更新录制后处理配置 |
| `RecordList` | — | 分页查询录制记录（数据库） |
| `RecordDelete` | `POST /terraform/v1/hooks/record/remove` | 删除录制记录（先删 Oryx 录制文件，再删数据库记录；Oryx 返回 "no record for" 视为幂等安全） |

### DVR 云录制

| RPC | 对应 Oryx HTTP API | 说明 |
| --- | --- | --- |
| `DvrQuery` | `POST /terraform/v1/hooks/dvr/query` | 查询 DVR 配置 |
| `DvrApply` | `POST /terraform/v1/hooks/dvr/apply` | 应用 DVR 配置 |
| `DvrFiles` | `POST /terraform/v1/hooks/dvr/files` | 列出 DVR 文件 |

### 回调配置

| RPC | 对应 Oryx HTTP API | 说明 |
| --- | --- | --- |
| `HooksApply` | `POST /terraform/v1/mgmt/hooks/apply` | 应用回调配置 |
| `HooksQuery` | `POST /terraform/v1/mgmt/hooks/query` | 查询回调配置 |

### SRS HTTP API 代理

| RPC | 对应 SRS HTTP API | 说明 |
| --- | --- | --- |
| `SrsVersions` | `GET /api/v1/versions` | 查询 SRS 版本 |
| `SrsStreams` | `GET /api/v1/streams` | 查询流列表 |
| `SrsClients` | `GET /api/v1/clients` | 查询客户端列表 |
| `SrsVhosts` | `GET /api/v1/vhosts` | 查询虚拟主机列表 |
| `SrsSummaries` | `GET /api/v1/summaries` | 查询系统状态 |
| `SrsRequests` | `GET /api/v1/tests/requests` | 查询最近请求（调试） |

### 中继拉流

| RPC | 说明 |
| --- | --- |
| `StartRelayPull` | 启动 FFmpeg 中继拉流（源流 → copy 推送到 SRS） |
| `StopRelayPull` | 停止中继拉流；请求体 `stop_recording=true` 时同时结束关联录制（查询录制中的记录并逐个结束） |

### 内部 Hook（仅 oryxgtw 调用）

| RPC | 说明 |
| --- | --- |
| `RecordBeginHook` | `on_record_begin` 回调落库 |
| `RecordEndHook` | `on_record_end` 回调落库（含产物信息） |

> 以下 RPC 仅供 `oryxgtw` 内部调用，业务服务请勿直接调用。

## 部署模式

### standalone（单机模式）

- 中继拉流任务存储在本地内存
- 无需配置 MQTT

### cluster（集群模式）

- 跨节点停止转推通过 MQTT 广播
- 需配置 `MqttConfig`

## 配置示例

```yaml
Name: oryxserver.rpc
ListenOn: 0.0.0.0:8081
GracePeriod: 500s
DeployMode: standalone

OryxConfig:
  Ip: 127.0.0.1
  Port: 80
  Timeout: 5000
  Secret: "your-srs-platform-secret"

DB:
  Type: postgres
  DSN: "host=localhost port=5432 user=root dbname=oryx sslmode=disable"

RedisDB: 5
Concurrency: 20

RelayConfig:
  SrsRtmpAddr: "rtmp://127.0.0.1:1935"
  DefaultApp: live
  SecretKey: secret
  SecretValue: ""

NacosConfig:
  IsRegister: true
  Host: nacos
  Port: 8848
  ServiceName: oryxserver
```

## 中继拉流

### 启动拉流

从源地址拉流，copy 推送到 SRS：

```protobuf
rpc StartRelayPull (StartRelayPullReq) returns (StartRelayPullRes);

message StartRelayPullReq {
  string source_url = 1;          // 源流地址（RTMP/RTSP/HTTP-FLV/HLS）
  string app = 2;                 // 目标应用名，默认 live
  string stream = 3;              // 目标流名，不填自动生成 UUID
  string secret_key = 4;          // 鉴权参数名
  string secret_value = 5;        // 鉴权参数值
  uint64 max_duration_seconds = 6; // 最大运行时长（秒），0 不限
}
```

### 播放地址约定

| 协议 | 地址格式 |
| --- | --- |
| RTMP | `rtmp://<host>:<rtmp_port>/{app}/{stream}` |
| HTTP-FLV | `http://<host>:<http_port>/{app}/{stream}.flv` |
| HLS | `http://<host>:<http_port>/{app}/{stream}.m3u8` |
| WebRTC | `webrtc://<host>:<rtc_port>/{app}/{stream}` |

## 数据流

```
业务服务 ──gRPC──> oryxserver ──HTTP──> Oryx/SRS
                      │
                      ├─ 录制管理 → Oryx HTTP API
                      ├─ 流状态查询 → SRS HTTP API
                      ├─ 中继拉流 → FFmpeg → SRS RTMP
                      └─ 录制落库 → PostgreSQL / MySQL
```

## 注意事项

1. SRS HTTP API 需 Bearer 鉴权，配置中的 `OryxConfig.Secret` 即 `SRS_PLATFORM_SECRET`
2. `SrsStreams` 分页查询 SRS v6 下限 10 条（`count=0/1` 都返回 10）
3. 中继拉流任务按节点本地内存管理，`cluster` 模式下跨节点停止需 MQTT 广播
4. 内部 Hook RPC 仅供 `oryxgtw` 调用，业务服务直接调用可能导致数据不一致

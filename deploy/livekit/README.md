# LiveKit 与录制部署

LiveKit Server + Egress 录制的本地编排，用于视频会议联调。SIP 侧（FreeSWITCH、LiveKit SIP Server）已迁移到 [`deploy/sipmediagw`](../sipmediagw/README.md)，本目录的对应服务在 Compose 中保持注释状态。

## 服务

| 服务 | 镜像 | 端口 | 说明 |
| --- | --- | --- | --- |
| `livekit-server` | `livekit/livekit-server:latest` | 7880/tcp，60000-60100/udp | HTTP API + WebSocket 信令与 WebRTC 媒体 |
| `livekit-egress` | `livekit/egress:latest` | 无对外端口 | 会议合成录制，经 Redis 队列与 server 通信 |

两个容器都在 `lk-net` 网络内，并配置了 `host.docker.internal:host-gateway`，便于回调宿主机上的 `livegtw`。

## 依赖

- Redis：独立部署，见 [`deploy/redis`](../redis/README.md)，默认宿主机端口 36379。
- `livegtw`：接收 LiveKit Webhook 并转发给 `live`，见 [`app/livegtw/etc/livegtw.yaml`](../../app/livegtw/etc/livegtw.yaml)。

## 使用

```bash
bash deploy/livekit/start.sh     # 启动并打印端口与密钥说明
bash deploy/livekit/stop.sh      # 停止
```

- 配置：`livekit-server.yaml`（API key/secret、Redis 地址、Webhook）与 `egress.yaml`（录制模板与输出路径）。
- 录制产物写入 `recordings/`（容器内 `/out`），对外播放由 [`deploy/nginx`](../nginx/README.md) 反代。
- 测试 Token：

```bash
bash deploy/livekit/gen-token.sh [user_id] [user_name] [dept_code]
```

脚本内置的签名密钥仅用于本地联调，生产环境必须替换为与 `livegtw` 一致的密钥，不要沿用仓库中的示例值。

## 生产注意事项

- 替换 `livekit-server.yaml`、`sip.yaml` 中的 API 密钥对与 Redis 密码。
- 7880 的信令端口需要 TLS，请在其前面加反向代理（nginx/caddy）终止 TLS；证书可用 [`deploy/tls`](../tls/gen-tls.sh) 生成或改用 CA 签发。
- Egress 的 Chrome 沙箱需要 `SYS_ADMIN`（v1.7.6+ 起必需），单实例建议 4C/4G。
- 录制目录需要可写并预留空间，长期归档建议转存对象存储。

## 相关文档

- [视频会议服务](../../docs/live/README.md)
- [LiveKit 回调与视频会议回调契约](../../docs/live/callbacks-guide.md)

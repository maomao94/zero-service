# Live 会议业务系统 MVP

## Goal

基于 livekitx 机制层构建可用的会议业务系统，交付"三件套"：
- `app/live`：会议 gRPC 业务服务（pgsql 持久化 + redis lock）
- `livegtw`：HTTP 网关（业务 API 转发 + LiveKit webhook 接收 + 测试页静态路由）
- HTML 测试页：livekit-client v2.22.1 全功能测试界面

## Requirements

1. 架构为 `浏览器 → livegtw(HTTP) → app/live(gRPC) → LiveKit Server(127.0.0.1:7880)`，webhook 由 LiveKit 推送到 livegtw，验签后转发 app/live 处理。
2. 会议基础能力（MVP 范围，后续扩展"货站"等业务）：
   - 创建会议（业务会议号 + LiveKit 房间映射）、加入会议（发放 join token）、查询会议列表/详情、结束会议。
   - 会议内管理操作：踢人、音频/视频静音、参与者列表、Data 广播/定向（SendData/InviteParticipant）、服务端 RPC。
   - LiveKit webhook 事件处理（验签、event ID 幂等、会议/参与者状态同步）。
3. 数据持久化用 PostgreSQL（gormx），配置风格参考 `app/oryxserver`（pgsql DataSource + Redis）；并发控制用 Redis lock（go-redis v9）。
4. 服务端不参与实时媒体（不 JoinRoom），实时媒体由浏览器直连 LiveKit；服务端只做管理 API + webhook + token 发放。
5. HTML 测试页覆盖：音视频入会/离会、管理操作、聊天（SDK 原生 ChatMessage + 自定义 topic UserData）、Data 广播/定向、RPC 双向调用、屏幕共享、事件日志。
6. MVP 测试阶段业务 API 免 JWT（JwtAuth 结构保留参考 socketgtw，默认关闭，供 Java 侧服务后续使用）。

## 任务地图

| 子任务 | 交付物 | 依赖 |
|---|---|---|
| 09-01-live-service | app/live gRPC 服务（proto/模型/logic/webhook 处理） | 无 |
| 09-01-live-gtw | livegtw 网关（API 转发 + webhook 接收 + 静态路由）【已完成】 | live-service（RPC 契约） |
| 09-01-live-test-page | HTML 测试页（livekit-client v2.22.1）【已完成】 | live-gtw（API 契约） |
| 09-01-live-e2e-verify | 三端联调验证（含模拟 webhook）【已完成】 | 前三个全部 |

## Acceptance Criteria

- [ ] app/live 编译通过（go build + go vet），单测通过
- [ ] 创建/加入/结束会议真实链路可用：浏览器双标签页可互相看到音视频
- [ ] 踢人/静音/Data 广播定向/RPC 在真实链路生效
- [ ] webhook 链路：模拟签名请求通过验签与幂等，会议状态正确更新；伪造签名被拒绝
- [ ] livegtw 测试页路由 /test/meeting 可访问，六大功能面板齐全
- [ ] 会议与参会记录落 pgsql，重启后数据仍在

## Notes

- 端口：app/live 21017（21xxx 段下一个可用），livegtw 11002（已确认空闲）
- 本地基础设施：LiveKit dev(7880)、pg-goctl(5432)、redis(36379, Pass G62m50oigInC30sf)
- LiveKit dev server 不推送真实 webhook，联调用构造签名请求模拟
- 业务侧命名：`app/live` 模块路径 `zero-service/app/live`；网关 `zero-service/livegtw`
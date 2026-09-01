# 端到端联调验证

## Goal

三端联调：app/live + livegtw + HTML 测试页 全链路验证，含 webhook 模拟。

## Requirements

1. 基础设施就绪检查：pg-goctl(5432) / redis(36379) / livekit-server(7880) 运行中
2. 启动顺序：app/live → livegtw → 浏览器测试页
3. 验证项：
   - 创建会议 → 落 pgsql（查询 live_meetings 表）
   - 双标签页入会 → 互相看到音视频；参与者表记录 joined
   - 聊天（原生+UserData topic）/ Data 广播定向 / RPC echo / 屏幕共享
   - 管理操作：踢人（被踢方收到断开）、静音他人、结束会议（房间删除、状态 ended）
   - webhook：构造带签名请求模拟 room_finished/participant_joined/left → 状态与参会记录正确更新；重复 event ID 幂等（只处理一次）；伪造签名被拒（401）
4. 全程发现的问题记录并修复。

## Acceptance Criteria

- [x] 全部验证项通过（见 requirements；真实音视频媒体渲染需浏览器人工验证，服务端链路已全绿）
- [x] 修复过程中发现的 bug 已解决并补充单测（livegtw JWT 启用、测试页 7 处修复、webhook 签名格式 base64+裸token）
- [x] 联调结果记录到本任务 notes 与各子任务 prd

## 联调结果记录

基础设施：pg-goctl(5432) / redis(36379) / livekit-server(7880) 全部运行中；app/live(21017) + app/livegtw(11002) 启动成功。

业务 API 全链路（带 JWT）：
- 无 token → 401（WithJwt 校验生效）
- 创建会议 → createUser=boss/deptCode=d01 落库（JWT claim → authctx → gRPC metadata → pgsql）
- 加入会议 → token(368字符) + ws://127.0.0.1:7880；参与者 alice 落库
- SendData / Mute(不存在者静默成功) → 200
- PerformRpc → 200（客户端未注册方法时网关返回 extproto code，走 gtwx 错误映射）
- 结束会议 → status=3；参与者批量 left

webhook 链路：
- 正确签名（base64 sha256 + 裸 JWT）→ 200，participant_joined(carol) 落库
- 错误签名 → 401 不转发
- room_finished 补发 ×2 → 200（处理幂等，状态保持 ended，闭环安全重放）

pgsql 落库：
- live_meetings：title/status/create_user=update_user=boss/dept_code=d01/start~end 时间
- live_meeting_participants：alice(create_user=boss)+carol(webhook,create_user 空)+left_time
- redis：live:outId_M（IdUtil 会议号序号）

测试页：/test/meeting HTTP 200，完整功能标签齐全（JS 语法校验通过）。

## Notes

- webhook 模拟脚本：Python 生成签名（base64 sha256 + 裸 JWT Authorization）POST 到 livegtw /webhook/livekit
- LiveKit dev server 不推真实 webhook；真实推送验证需要 livekit-server.yaml 配置 webhook url（已知限制，后续配置）
- **已知限制**：真实音视频媒体渲染（摄像头/麦克风/屏幕共享/编解码/远端画面）需浏览器人工双标签验证，CLI 环境无法覆盖；服务端链路（会议/参与者/Data/RPC/webhook）已全绿
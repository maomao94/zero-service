# app/live 会议 gRPC 服务 — 执行计划

## 前置

- [ ] 确认 goctl 可用（`goctl --help`）
- [ ] 确认 pgsql 可达（`nc -z 127.0.0.1 5432`）、redis 可达（36379）
- [ ] 参考 `app/oryxserver/gen.sh`、`internal/config/config.go`、`model/` 组织

## 步骤

1. **骨架**：拷贝 oryxserver 结构生成 app/live 骨架（gen.sh + live.proto 空服务），`goctl rpc protoc live.proto` 生成 pb
2. **配置**：etc/live.yaml + internal/config/config.go（RpcServerConf + LiveKit{Url,ApiKey,ApiSecret,WebhookKey,TokenValidFor} + DB + Redis）
3. **模型**：internal/model/gormmodel/ 下 LiveMeeting、LiveMeetingParticipant（gorm tags），svc 中 AutoMigrate
4. **svc**：ServiceContext 组装 livekitx.New、gormx 打开 pgsql、redis client、store 接口
5. **logic（核心）**：
   - 会议号生成器（M+日期+随机）
   - CreateMeeting：落库 + CreateRoom（房间名=会议号）
   - JoinMeeting：校验状态 → JoinToken(CanPublish/CanSubscribe/CanPublishData) → 返回 token/wsUrl（wsUrl 由配置 URL 换算 ws 协议）
   - EndMeeting：redis lock → DeleteRoom → 状态流转 → 批量离会
   - KickParticipant/MuteParticipant(±Video)/ListParticipants/SendMeetingData/PerformMeetingRpc：透传 livekitx
   - WebhookNotify：幂等 + 状态同步
6. **错误处理**：业务错误 codes 映射（NotFound/AlreadyExists/InvalidArgument/FailedPrecondition）
7. **单测**：
   - meeting logic 状态流转（store 用测试实现）
   - webhook 幂等（重复 ID 只处理一次）+ 事件处理分支
   - 参数校验错误路径
8. **验证**：`go build ./app/live/...`、`go vet ./app/live/...`、`go test ./app/live/...`

## 验证命令

```bash
go build ./app/live/...
go vet ./app/live/...
go test ./app/live/...
# 手动冒烟（连真实 dev server）：
go run ./app/live -f app/live/etc/live.yaml
```

## 评审门

- [ ] proto 契约完整（10 个 RPC）
- [ ] 模型字段满足会议单据与参会记录要求
- [ ] 幂等与 lock 有单测覆盖
- [ ] 编译/vet/test 全绿

## 回滚点

- 本任务只新增 app/live 目录，无既有代码变更；删除目录即可回滚
- 不修改 common/livekitx 与既有服务
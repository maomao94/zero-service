# LiveKit 会议票据机制 - 实现计划

## 实现顺序

### 阶段 1: gRPC Proto 修改

**目标**: 修改 proto 文件，新增消息类型和 RPC 方法

**步骤**:
1. 修改 `app/live/live.proto`
   - 新增 `GenerateMeetingTicketReq` 消息类型
   - 新增 `GenerateMeetingTicketRes` 消息类型
   - 新增 `JoinMeetingByTicketReq` 消息类型
   - 新增 `JoinMeetingByTicketRes` 消息类型
   - 新增 `GenerateMeetingTicket` RPC 方法
   - 新增 `JoinMeetingByTicket` RPC 方法

2. 重新生成 gRPC 代码
   ```bash
   cd app/live && bash gen.sh
   ```

**验证**:
- [ ] proto 文件语法正确
- [ ] gRPC 代码生成成功
- [ ] 生成的 Go 代码编译通过

### 阶段 2: gRPC 业务逻辑实现

**目标**: 实现票据生成和验证的业务逻辑

**步骤**:
1. 实现 `GenerateMeetingTicket` 逻辑
   - 验证会议是否存在且状态正常
   - 生成票据字符串
   - 存储到 Redis（带 TTL）
   - 返回票据信息和 joinUrl

2. 实现 `JoinMeetingByTicket` 逻辑
   - 从 Redis 获取票据数据
   - 验证票据是否存在且未过期
   - 验证 identity 是否与票据绑定的 identity 匹配
   - **Upsert 参会人记录**（与 JoinMeeting 一致）
   - 删除已使用的票据
   - 生成 LiveKit join token
   - 返回 token 和会议信息

3. 修改 `JoinMeeting` 逻辑
   - 从 authctx 获取 userId 和 userName
   - 使用 authctx 中的信息作为 identity 和 name

**验证**:
- [ ] GenerateMeetingTicket 逻辑正确
- [ ] JoinMeetingByTicket 逻辑正确
- [ ] JoinMeeting 逻辑正确
- [ ] 单元测试通过

### 阶段 3: 网关 API 修改

**目标**: 修改 API 定义，新增路由

**步骤**:
1. 修改 `app/livegtw/livegtw.api`
   - 新增 `GenerateMeetingTicketReq` 类型
   - 新增 `GenerateMeetingTicketRes` 类型
   - 新增 `JoinMeetingByTicketReq` 类型
   - 新增 `JoinMeetingByTicketRes` 类型
   - 修改 `JoinMeetingReq` 类型（移除 Identity 和 Name）
   - 新增 ticket 路由组（不需要鉴权）
   - 在 meeting 路由组中新增 generateTicket 接口

2. 重新生成网关代码
   ```bash
   cd app/livegtw && bash gen.sh
   ```

**验证**:
- [ ] API 定义语法正确
- [ ] 网关代码生成成功
- [ ] 生成的 Go 代码编译通过

### 阶段 4: 网关业务逻辑实现

**目标**: 实现网关层的票据接口

**步骤**:
1. 实现 `generateMeetingTicket` 逻辑
   - 从 authctx 获取用户信息
   - 调用 gRPC GenerateMeetingTicket
   - 返回票据信息

2. 实现 `joinMeetingByTicket` 逻辑
   - 调用 gRPC JoinMeetingByTicket
   - 返回 token 和会议信息

3. 修改 `joinMeeting` 逻辑
   - 从 authctx 获取用户信息
   - 调用 gRPC JoinMeeting

**验证**:
- [ ] generateMeetingTicket 逻辑正确
- [ ] joinMeetingByTicket 逻辑正确
- [ ] joinMeeting 逻辑正确
- [ ] 编译通过

### 阶段 5: 测试验证

**目标**: 验证整个流程

**步骤**:
1. 启动服务
   ```bash
   cd app/live && go run live.go -f etc/live.yaml
   cd app/livegtw && go run livegtw.go -f etc/livegtw.yaml
   ```

2. 测试生成票据
   ```bash
   curl -X POST http://localhost:8888/live/v1/meeting/generateTicket \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{"meetingNo":"M20250902001","identity":"device-001"}'
   ```

3. 测试根据票据加入会议
   ```bash
   curl -X POST http://localhost:8888/live/v1/meeting/joinByTicket \
     -H "Content-Type: application/json" \
     -d '{"ticket":"TICKET-M20250902001-xxx","identity":"device-001"}'
   ```

4. 测试标准加入会议
   ```bash
   curl -X POST http://localhost:8888/live/v1/meeting/join \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{"meetingNo":"M20250902001"}'
   ```

**验证**:
- [ ] 生成票据成功
- [ ] 根据票据加入会议成功
- [ ] 标准加入会议成功
- [ ] 票据一次性使用验证
- [ ] 票据过期验证
- [ ] identity 不匹配验证

## 文件清单

### 需要修改的文件

| 文件 | 修改内容 |
|------|----------|
| `app/live/live.proto` | 新增消息类型和 RPC 方法 |
| `app/live/internal/logic/meeting/` | 新增 GenerateMeetingTicketLogic 和 JoinMeetingByTicketLogic |
| `app/livegtw/livegtw.api` | 新增类型定义和路由 |
| `app/livegtw/internal/logic/meeting/` | 新增 generateMeetingTicketLogic 和 joinMeetingByTicketLogic |
| `app/livegtw/internal/logic/meeting/meetinglogic.go` | 修改 JoinMeeting 逻辑 |

### 需要新增的文件

| 文件 | 内容 |
|------|------|
| `app/live/internal/logic/meeting/generatemeetingticketlogic.go` | 生成票据逻辑 |
| `app/live/internal/logic/meeting/joinmeetingbyticketlogic.go` | 根据票据加入会议逻辑 |
| `app/livegtw/internal/logic/meeting/generatemeetingticketlogic.go` | 网关层生成票据逻辑 |
| `app/livegtw/internal/logic/meeting/joinmeetingbyticketlogic.go` | 网关层根据票据加入会议逻辑 |

## Rollback Points

1. **Proto 修改回滚**: 恢复原始 `live.proto` 文件
2. **gRPC 逻辑回滚**: 删除新增的 logic 文件
3. **API 修改回滚**: 恢复原始 `livegtw.api` 文件
4. **网关逻辑回滚**: 删除新增的 logic 文件

## Validation Commands

```bash
# 编译检查
cd app/live && go build ./...
cd app/livegtw && go build ./...

# 测试检查
cd app/live && go test ./...
cd app/livegtw && go test ./...

# 代码检查
cd app/live && go vet ./...
cd app/livegtw && go vet ./...
```

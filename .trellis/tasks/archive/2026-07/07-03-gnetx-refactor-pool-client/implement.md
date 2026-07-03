# gnetx 重构：实现计划

## 实现顺序

### Step 1: 修改 Session（核心变更）

**文件**: `common/gnetx/session.go`

- [ ] 移除 `pool atomic.Pointer[antsx.ReplyPool[any]]` 字段
- [ ] 移除 `poolOnce sync.Once` 字段
- [ ] 新增 `replyPool *antsx.ReplyPool[any]` 字段（非拥有型引用）
- [ ] 新增 `extraClose func()` 字段（Dialer 使用）
- [ ] 修改 `newSession` 签名：增加 `replyPool *antsx.ReplyPool[any]` 参数
- [ ] 移除 `ensurePool()` 方法
- [ ] 修改 `Request()`：使用 `s.id + "|" + msg.TID()` 作为复合 TID，操作 `s.replyPool`
- [ ] 修改 `resolveResponse()`：内部构造 `s.id + "|" + tid`，操作 `s.replyPool`
- [ ] 修改 `Close()`：移除 `pool.Close()` 调用；新增 `extraClose()` 调用

**风险**: 高（所有调用点需同步更新）
**验证**: `go build ./common/gnetx/` 编译通过（暂时不跑测试，后续步骤补齐）

### Step 2: 修改 Server

**文件**: `common/gnetx/server.go`

- [ ] Server 新增 `replyPool *antsx.ReplyPool[any]` 字段
- [ ] `NewServer()` 中创建 `antsx.NewReplyPool[any]()`
- [ ] `OnOpen()` 中 `newSession()` 调用增加 `s.replyPool` 参数
- [ ] `Shutdown()`/`Stop()` 中关闭 `replyPool`（在所有连接关闭后）
- [ ] OnTraffic 中 `resolveResponse` 调用不变（Session 内部处理复合 TID）

**风险**: 中
**验证**: `go test ./common/gnetx/ -run TestServer -v`

### Step 3: 修改 Client（长连接）

**文件**: `common/gnetx/client.go`

- [ ] Client 新增 `replyPool *antsx.ReplyPool[any]` 字段
- [ ] `NewClient()` 中创建 `antsx.NewReplyPool[any]()`
- [ ] `OnOpen()` 中 `newSession()` 调用增加 `c.replyPool` 参数
- [ ] `Close()` 中关闭 `replyPool`
- [ ] 断连重连：旧 Session 关闭时不关池，新 Session 复用同一池

**风险**: 中
**验证**: `go test ./common/gnetx/ -run TestClient -v`

### Step 4: 新增 Dialer（短连接客户端，基于 Promise）

**新建文件**: `common/gnetx/dialer.go`

- [ ] 定义 `Dialer` 结构体（无 ReplyPool 字段）
- [ ] 定义内部 `dialAdapter`（gnet.EventHandler）
  - 持有 `*antsx.Promise[any]` 和期望的 `tid`
  - OnOpen: 创建 Session（replyPool=nil），编码并发送请求
  - OnTraffic: 解码，若 Response.ResponseTID() == tid → resolve promise
  - OnClose: reject promise（sync.Once 保证只一次）
- [ ] `NewDialer(opts ...ClientOption) *Dialer`
- [ ] `Dial(ctx, network, address) (*Session, error)` — 创建 gnet.Client，拨号，返回 Session（仅支持 Send）
- [ ] `Request(ctx, network, address, msg Correlatable) (any, error)` — Promise 驱动，一步完成
- [ ] `Close() error`

**新建文件**: `common/gnetx/dialer_test.go`

- [ ] `TestDialerDial` — 基本拨号，Send 发送
- [ ] `TestDialerRequest` — 一键请求-响应
- [ ] `TestDialerRequestTimeout` — ctx 超时

**风险**: 低（全新代码，基于 Promise 无池开销）
**验证**: `go test ./common/gnetx/ -run TestDialer -v`

### Step 5: Server 连接统计（statLoop）

**文件**: `common/gnetx/server.go`

- [ ] Server 新增 `totalConnects atomic.Int64`、`totalDisconnects atomic.Int64`
- [ ] Server 新增 `statsCtx context.Context`、`statsCancel context.CancelFunc`、`statsWG sync.WaitGroup`
- [ ] OnOpen 中 `s.totalConnects.Add(1)`，OnClose 中 `s.totalDisconnects.Add(1)`
- [ ] OnBoot 中启动 `go s.statLoop()`（参考 antsx.ReplyPool.statLoop 模式）
- [ ] `statLoop()`：每 1min ticker，atomic.Swap 取 delta，`logx.Statf` 输出 `active/connects/m disconnects/m`
- [ ] `Shutdown()`/`Stop()` 中 `statsCancel()` + `statsWG.Wait()` 停止循环

**风险**: 低
**验证**: `go build ./common/gnetx/` 编译通过

### Step 6: 整体验证

- [ ] `go test ./common/gnetx/ -v` — 所有现有测试通过
- [ ] `go test ./common/gnetx/ -race -v` — 无数据竞争
- [ ] `go vet ./common/gnetx/` — 无 vet 警告

## 回滚点

- Step 1 之前：`git stash` 可回退
- 每 Step 完成后 commit 一个 checkpoint（不上传，仅本地）

## 关键文件变更列表

| 文件 | 变更类型 | 变更内容 |
|------|---------|---------|
| `session.go` | 修改 | 移除 pool/poolOnce，新增 replyPool/extraClose，修改 newSession/Request/resolveResponse/Close |
| `server.go` | 修改 | 新增 replyPool，修改 newSession 调用，修改 Shutdown |
| `client.go` | 修改 | 新增 replyPool，修改 newSession 调用，修改 Close |
| `dialer.go` | 新建 | Dialer + dialAdapter 实现 |
| `dialer_test.go` | 新建 | Dialer 测试 |
| `session_test.go` | 无需修改 | Session 测试不涉及 pool 字段 |

## 预估

- Step 1 (Session): ~30 行改动
- Step 2 (Server): ~20 行改动
- Step 3 (Client): ~25 行改动
- Step 4 (Dialer): ~150 行新增
- Step 5 (验证): 跑测试

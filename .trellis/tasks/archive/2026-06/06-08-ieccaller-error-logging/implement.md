# 执行计划：ieccaller 错误日志与 gRPC 错误转换优化

## 阶段 1: 基础设施 (CommandRejectedError)

### 1.1 创建 CommandRejectedError 类型

**文件**: `common/iec104/client/errors.go`

**改动**:
- 新增 `CommandRejectedError` 结构体
- 实现 `Error()` 方法，输出结构化信息
- 实现 `Is()` 方法以支持 `errors.Is` 匹配

**验证**:
```bash
cd common/iec104/client && go test -run TestCommandRejectedError
```

### 1.2 更新 ClientHandler 使用 CommandRejectedError

**文件**: `app/ieccaller/internal/iec/clienthandler.go`

**改动**:
- `resolveCommandAck()` 第 516-519 行：替换 `fmt.Errorf` 为 `CommandRejectedError`
- `resolveCommandAck()` 第 527-528 行：同上

**验证**:
```bash
cd app/ieccaller && go build ./...
```

## 阶段 2: 错误分类优化

### 2.1 更新 CommandAckHelper

**文件**: `app/ieccaller/internal/logic/command_ack_helper.go`

**改动**:
- 新增 `isCommandRejectedError()` 辅助函数
- 更新 `wrapCommandAckError()` 增加 `iec_rejected` 分支
- 映射到 `extproto.Code__1_05_BIZ_STATE` (105102 / 409)

**验证**:
```bash
cd app/ieccaller && go test ./internal/logic/...
```

### 2.2 更新 BroadcastAck 错误分类

**文件**: `app/ieccaller/mqtt/broadcast.go`

**改动**:
- `publishAckReply()` 第 325-332 行：增加 `iec_rejected` 分类
- 新增 `isCommandRejectedError()` 辅助函数（或复用 logic 包的）

**验证**:
```bash
cd app/ieccaller && go test ./mqtt/...
```

### 2.3 更新 BroadcastReplyPool 错误重建

**文件**: `app/ieccaller/internal/svc/servicecontext.go`

**改动**:
- `PushPbBroadcastWithAck()` 第 313-321 行：增加 `iec_rejected` 分支
- 新增 `reconstructRejectedError()` 函数从错误消息重建语义

**验证**:
```bash
cd app/ieccaller && go build ./...
```

## 阶段 3: 日志增强

### 3.1 增强 LoggerInterceptor

**文件**: `common/Interceptor/rpcserver/loggerInterceptor.go`

**改动**:
- 新增 `time` 导入
- 记录请求开始时间
- 提取 gRPC Status 详情
- 使用 `Errorw` 打印结构化字段

**验证**:
```bash
cd common/Interceptor && go test ./...
```

### 3.2 更新 Logic 层日志

**文件**: 所有 `app/ieccaller/internal/logic/send*logic.go`

**改动**:
- 每个 sendXxxLogic 的错误处理分支：
  - `isCommandRejectedError` → `Warnw` (设备拒绝)
  - 其他错误 → `Errorw` (服务故障)
- 字段包含：method, coa, ioa, error

**涉及文件** (11 个):
1. `sendsinglecommandlogic.go`
2. `senddoublecommandlogic.go`
3. `sendstepcommandlogic.go`
4. `sendsetpointnormalizedlogic.go`
5. `sendsetpointscaledlogic.go`
6. `sendsetpointfloatlogic.go`
7. `sendbitstringcommandlogic.go`
8. `sendcommandlogic.go`
9. `sendinterrogationcmdlogic.go`
10. `sendcounterinterrogationcmdlogic.go`
11. `sendreadcmdlogic.go`
12. `sendtestcmdlogic.go`

**验证**:
```bash
cd app/ieccaller && go test ./internal/logic/...
```

### 3.3 修复无 Context 的日志

**文件**: 所有 `app/ieccaller/internal/logic/send*logic.go`

**改动**:
- 将 `logx.Errorf("cli is empty")` 替换为 `logx.WithContext(l.ctx).Errorw("IEC客户端不存在")`
- 增加 method 字段

**验证**:
```bash
cd app/ieccaller && go build ./...
```

## 阶段 4: 集成验证

### 4.1 编译验证

```bash
cd /Users/hehanpeng/GolandProjects/zero-service
go build ./app/ieccaller/...
```

### 4.2 单元测试

```bash
cd /Users/hehanpeng/GolandProjects/zero-service
go test ./common/iec104/client/... ./app/ieccaller/...
```

### 4.3 日志格式验证

**检查项**:
- [ ] LoggerInterceptor 输出包含 method, duration, grpc_code, reason
- [ ] 业务层设备拒绝使用 Warnw
- [ ] 业务层服务故障使用 Errorw
- [ ] 所有日志使用 logx.WithContext

### 4.4 错误码验证

**检查项**:
- [ ] isNegative=true → 409 / 105102
- [ ] ACK 超时 → 504 / 100997
- [ ] 重复下发 → 409 / 105103
- [ ] 找不到客户端 → 503 / 106101

## 回滚方案

如果出现问题，按以下顺序回滚：

1. 回滚 `command_ack_helper.go` 的 `isCommandRejectedError` 分支
2. 回滚 `clienthandler.go` 的 `CommandRejectedError` 使用
3. 回滚 `loggerInterceptor.go` 的结构化日志

## 依赖关系

```
阶段 1 (CommandRejectedError)
    ↓
阶段 2 (错误分类优化) ← 依赖阶段 1 的类型定义
    ↓
阶段 3 (日志增强) ← 依赖阶段 2 的错误分类
    ↓
阶段 4 (集成验证) ← 依赖所有前序阶段
```

## 预估工时

| 阶段 | 任务 | 预估 |
|------|------|------|
| 1 | CommandRejectedError | 15 min |
| 2 | 错误分类优化 | 30 min |
| 3 | 日志增强 | 45 min |
| 4 | 集成验证 | 15 min |
| **总计** | | **~2 hours** |

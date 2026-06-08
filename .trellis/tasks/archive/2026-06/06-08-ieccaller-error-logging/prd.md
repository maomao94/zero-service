# 优化 ieccaller 错误日志与 gRPC 错误转换

## 背景

ieccaller 服务在集群模式下发送 IEC 104 控制命令时，出现以下错误：

```
IEC发送浮点设点命令失败: command rejected: cot=UnknownIOA isNegative=true
集群推送ACK失败: broadcast command error: command rejected: cot=UnknownTypeID isNegative=true
IEC发送双点命令失败: command rejected: cot=UnknownTypeID isNegative=true
```

当前错误处理存在以下问题：

1. **错误分类不当**：设备明确拒绝命令（isNegative=true）被映射为 `106102 第三方服务异常 / 503`，语义不准确
2. **日志重复且信息不足**：RPC 拦截器和业务层重复打印 ERROR，且缺乏结构化字段
3. **集群 ACK 链路丢失上下文**：broadcast error 退化为泛化的 `broadcast command error: ...`
4. **单机 vs 集群错误不一致**：同一错误在两种模式下返回不同的 message

## 问题分析

### 错误码映射问题

| 场景 | 当前行为 | 期望行为 |
|------|---------|---------|
| `cot=UnknownTypeID` | 503 / 106102 | 409 / 105102 业务状态不允许 |
| `cot=UnknownIOA` | 503 / 106102 | 409 / 105102 业务状态不允许 |
| `isNegative=true` | 503 / 106102 | 409 / 105102 业务状态不允许 |
| ACK 超时 | 504 / 100997 | 保持不变 |
| 重复下发 | 409 / 105103 | 保持不变 |
| 找不到 IEC 客户端 | 503 / 106101 | 保持不变 |

### 关键文件

- [clienthandler.go](app/ieccaller/internal/iec/clienthandler.go:516) - ACK 拒绝时丢失结构化数据
- [command_ack_helper.go](app/ieccaller/internal/logic/command_ack_helper.go:15) - 默认分类为 THIRD_PARTY
- [loggerInterceptor.go](common/Interceptor/rpcserver/loggerInterceptor.go:14) - 只打印泛化错误
- [broadcast.go](app/ieccaller/mqtt/broadcast.go:317) - ErrorKind 只有 timeout/duplicate/unknown

## 验收标准

### AC1: 错误分类准确
- **Given** IEC 从站返回 `isNegative=true` 或 `cot=UnknownTypeID/UnknownIOA`
- **When** 命令被拒绝
- **Then** gRPC 返回 409 / 105102，message 包含 `cot`、`typeId`、`coa`、`ioa`

### AC2: 日志结构化且不重复
- **Given** 任意 RPC 请求
- **When** 发生错误
- **Then** 
  - LoggerInterceptor 打印一次 ERROR，包含 `method、duration、grpc_code、reason、trace、span`
  - 业务层打印 WARN（设备拒绝）或 ERROR（服务故障），包含 `host、port、coa、ioa、typeId、cot、cotCause、isNegative`
  - 日志级别符合约定：设备拒绝=WARN，服务故障=ERROR

### AC3: 集群 ACK 链路完整
- **Given** 集群模式下发送命令
- **When** 远端节点执行失败并返回 ACK
- **Then**
  - BroadcastAckBody.ErrorKind 增加 `iec_rejected`、`cot_error`
  - 日志包含 `broadcastTid、broadcastMethod、nodeId、deployMode`
  - 错误信息从远端节点完整传递到调用方

### AC4: 单机/集群错误一致
- **Given** 相同的 IEC 命令被拒绝
- **When** 分别在单机和集群模式下执行
- **Then** 返回的 gRPC 错误码、reason、message 结构一致

### AC5: 日志文件分层
- **Given** 集群模式部署
- **When** 服务运行
- **Then**
  - 控制台输出简短错误摘要
  - 文件日志输出完整结构化字段
  - 建议按节点实例区分日志文件

## 约束条件

- 不改变现有错误码枚举定义（extproto.proto）
- 向后兼容：现有错误码 106101/106102 仍可用于真正的第三方服务异常
- 遵循项目错误码规范（docs/error-codes.md）
- 日志格式遵循 go-zero logx 规范

## 范围

### 包含
- ieccaller 服务的错误分类优化
- LoggerInterceptor 增强
- 集群广播 ACK 错误分类
- 日志结构化字段补充

### 不包含
- 新增 extproto 错误码枚举（复用现有 105102）
- 其他服务的错误处理改造
- 日志采集/分析系统搭建

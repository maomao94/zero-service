# 恢复全仓构建并修复行为测试

## Goal

恢复当前 `master` 的全仓构建基线，并修复已经确认的 `flowx`、Modbus、文件服务、Eino、DJI Cloud 和 ISP 测试回归。

## Requirements

- `common/flowx` 必须使用 Azure 官方 `github.com/Azure/go-workflow`，禁止恢复本机绝对路径 `replace`。
- 官方库需固定到包含 `WorkflowOption`、`Mutator`、`StepInterceptor` 和 `AttemptInterceptor` 的提交；依赖更新后保持现有 `flowx` 公共行为。
- 保留 `ModbusSlaveConfig` 的字符串主键模型，将 Modbus protobuf 中相关 ID 契约统一为字符串并重新生成代码。
- 保留文件服务 `Oss` 的字符串主键模型，将 OSS 相关 protobuf ID 字段统一为字符串并重新生成代码；分类和状态字段仅做必要的内部数值类型适配。
- 修复 Eino Agent 日志格式参数错误，使 vet 检查通过。
- 恢复 DJI 离线命令错误中的网关上下文，并保持结构化日志行为。
- 保留 ISP 任务控制通知的生产 3 秒延迟，同时让测试可无等待地验证通知内容，并隔离每次测试的 SQLite 数据库。
- 不修改任何证书、私钥或相关配置文件。
- 不处理因沙箱禁止本地监听端口或访问 Docker socket而产生的环境性测试失败。

## Acceptance Criteria

- [ ] `go test ./common/flowx` 通过，现有 options、拦截器和工作流行为测试保持有效。
- [ ] `go test ./app/bridgemodbus/... ./app/file/... ./common/einox/agent` 通过编译和测试。
- [ ] `app/bridgemodbus/gen.sh` 与 `app/file/gen.sh` 执行成功，生成代码分别与对应 proto 契约一致。
- [ ] `go test ./app/djicloud/internal/hooks -run TestRegisterDjiClientRegistersHandlersAndOnlineChecker -count=1` 通过。
- [ ] `go test ./app/ispagent/internal/handler -run TestHandleTaskControlParsesSubstationFromMessageCode -count=3` 通过，且不产生跨轮次唯一键冲突。
- [ ] `go build -mod=readonly ./...` 通过，且不会自动改写 `go.mod` / `go.sum`。
- [ ] 对不依赖受限端口或 Docker 的相关包完成定向测试；全量测试中的环境性失败单独记录。
- [ ] `git diff --check` 通过，生成代码 diff 仅来自 `bridgemodbus.proto` 和 `file.proto`。
- [ ] 证书和私钥目录无 diff。

## Notes

- 用户确认 Modbus 字符串主键是有意设计，必须改 protobuf，不能回退模型。
- 用户确认 OSS 字符串主键同样是有意设计，必须改 protobuf，不能回退模型。
- Modbus ID 字段原地从 `int64` 改为 `string` 会改变 protobuf wire type；调用方必须同步升级。
- OSS ID 字段原地从 `int64` 改为 `string` 同样会改变 protobuf wire type；调用方必须同步升级。

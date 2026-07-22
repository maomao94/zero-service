# 实施计划

## 1. 恢复 flowx

- [x] 将 `github.com/Azure/go-workflow` 更新到官方提交 `fc60b72c83a8199becc949cdf981f28637f02543`。
- [x] 运行 `go mod tidy`，检查依赖 diff 不包含本机路径或无关升级。
- [x] 运行 `go test ./common/flowx`。

## 2. 修复构建错误

- [x] 在文件服务 OSS 配置边界补显式 `int` -> `int64` 转换。
- [x] 将文件服务 OSS 相关 ID protobuf 契约改为字符串，字段号保持不变。
- [x] 执行 `app/file/gen.sh` 并检查生成代码。
- [x] 修复 Eino Agent 日志格式参数。
- [x] 修改 Modbus proto 的三个 ID 字段为字符串。
- [x] 执行 `app/bridgemodbus/gen.sh` 并检查生成代码。
- [x] 更新 Modbus 保存逻辑，直接返回字符串 ID。
- [x] 运行相关包构建与测试。

## 3. 修复行为测试

- [x] 恢复 DJI 离线错误的 `gateway_sn` 上下文并运行定向测试。
- [x] 为 ISP 任务控制内部实现注入通知延迟，生产入口保持 3 秒。
- [x] 隔离 ISP 测试 SQLite DSN，定向测试使用 `-count=3`。

## 4. 集成验证

- [x] 运行受影响包的定向测试。
- [x] 运行 `go build -mod=readonly ./...`。
- [x] 运行 `go test ./...`，将沙箱端口/Docker 限制与代码失败分开记录。
- [x] 运行 `git diff --check`，确认敏感文件无 diff。
- [x] 审查实际 diff，并按 Trellis 完成检查与规范同步。

全量测试残余：`common/gnetx` 的 `TestClientConnectError` 与 `TestClientOnConnectOnReconnect` 因现有实现和测试契约不一致失败；该包无本任务 diff，按用户范围不修改。

## 回滚点

- 依赖更新、Modbus 契约、行为测试修复分三组保持可独立审查。
- 若官方提交引入额外不兼容，停止在依赖步骤，不恢复本机 `replace`。

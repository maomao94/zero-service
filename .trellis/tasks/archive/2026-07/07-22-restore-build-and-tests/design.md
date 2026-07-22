# 技术设计

## 范围

本任务恢复当前构建与指定行为测试，不做证书治理、无关重构或部署改动。

## 1. flowx 官方依赖

`common/flowx` 最初通过本机绝对路径 `replace` 使用 Azure 官方仓库 `main`，代码依赖 2026 年 5 月合入但未进入 `v0.1.13` 标签的 API。后续移除 `replace` 后，模块回落到 `v0.1.13` 并导致编译失败。

使用官方提交 `fc60b72c83a8199becc949cdf981f28637f02543` 作为最低已确认版本，由 Go 解析并写入对应 pseudo-version。该提交包含：

- `WorkflowOption`
- `Mutator`
- `StepInterceptor` / `AttemptInterceptor`
- `ContextKey` 与日志拦截器

不引入本机路径，不复制官方库实现。依赖更新后运行 `go mod tidy` 并审查所有间接依赖变化。

## 2. Modbus 字符串 ID 契约

保留 `gormx.LegacyStringBaseModel`。修改契约源文件：

- `PbModbusConfig.id`: `int64` -> `string`
- `SaveConfigRes.id`: `int64` -> `string`
- `DeleteConfigReq.ids`: `repeated int64` -> `repeated string`

字段编号保持不变，但 wire type 发生变化，因此这是明确的不兼容契约升级。执行 `app/bridgemodbus/gen.sh`，业务逻辑直接传递字符串 ID，不做有损转换。

## 3. 文件服务 OSS 字符串 ID 契约

保留 `gormx.LegacyStringBaseModel`。修改 `file.proto` 中所有直接表示 OSS 主键的字段：

- `Oss.id`: `int64` -> `string`
- `OssDetailReq.id`: `int64` -> `string`
- `CreateOssRes.id`: `int64` -> `string`
- `UpdateOssReq.id`: `int64` -> `string`
- `DeleteOssReq.id`: `int64` -> `string`
- `DeleteOssRes.id`: `int64` -> `string`

字段编号保持不变，执行 `app/file/gen.sh` 并同步生成代码。`Category` 和 `Status` 继续使用 GORM 模型的 `int`，在 protobuf `int64` 和 OSS 配置 `int64` 边界显式转换。

## 4. 其他构建错误

- Eino Agent 的 `Errorf` 同时传入目录和错误参数。

## 5. DJI 错误契约

离线检查失败时返回包含 `gateway_sn` 的稳定错误文本，使调用方获得必要上下文并与现有测试契约一致。结构化日志仍由 `logDjiSDKError` 输出，不改变其他命令错误。

## 6. ISP 延迟与测试隔离

生产入口继续使用 3 秒异步通知延迟。将实现拆为保留现有签名的入口和可注入通知延迟的内部 helper，单测传入零延迟，避免慢测与超时。

测试 SQLite DSN 增加每次测试实例唯一标识，避免 `go test -count=N` 复用 shared in-memory 数据库造成唯一键冲突。

## 兼容与回滚

- `flowx` 依赖可回滚到本任务前版本，但回滚后该包无法编译；不得以本地 `replace` 作为回滚手段。
- Modbus protobuf 需要服务端和调用方同步升级。回滚时契约源与生成代码必须一起回滚。
- 文件服务 OSS protobuf 同样需要服务端和调用方同步升级。回滚时契约源与生成代码必须一起回滚，不得回退字符串主键模型。
- 其余变更均为内部类型、日志参数或测试性改造，不改变公开接口。

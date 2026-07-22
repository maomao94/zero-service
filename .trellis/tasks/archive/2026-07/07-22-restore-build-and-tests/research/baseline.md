# 基线研究

## 已确认故障

- `common/flowx` 引用了 `go-workflow v0.1.13` 不存在的 API。
- `app/file/internal/svc` 存在 `int` 到 `int64` 类型不匹配。
- `app/bridgemodbus/internal/logic` 将字符串模型 ID 转为 `int64`。
- `common/einox/agent` 的日志格式有两个占位符但只传一个参数。
- DJI hook 测试期望带 `gateway_sn` 的离线错误，实现已丢失该上下文。
- ISP 通知生产延迟为 3 秒，测试只等待 1 秒；shared memory SQLite 在 `-count` 下复用数据。

## flowx 根因

引入 `common/flowx` 的提交同时添加了：

`replace github.com/Azure/go-workflow => /Users/.../go-workflow`

后续依赖清理删除了该不可移植 replace，但仍固定 `v0.1.13`。Azure 官方仓库 `main` 的提交 `fc60b72c83a8199becc949cdf981f28637f02543` 已包含 flowx 使用的 API。

## 已确认决策

- 使用 Azure 官方库，不使用本地 fork/path replace。
- Modbus 字符串模型主键是有意变更；修改 protobuf 契约，不回退模型。
- 不处理证书和私钥。

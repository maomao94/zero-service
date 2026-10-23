# facade/openapi

`facade/openapi` 是开放平台的统一转发服务：接收 `OpenApiRpc.Invoke`，按 `method` 路由到下游 gRPC 服务，并用 protojson 转换 `data`。

## 服务信息

| 项 | 值 |
| --- | --- |
| gRPC 端口 | 21018 |
| 契约 | [`openapi.proto`](./openapi.proto) |
| 服务 | `openapi.OpenApiRpc/Invoke` |
| 配置 | [`etc/openapi.yaml`](./etc/openapi.yaml) |

## 请求/响应

```proto
InvokeReq {
  string app_id    = 1;
  string method    = 2;
  bytes  data      = 3; // 该 method 的 protojson
  string tid       = 4;
  int64  timestamp = 5;
}

InvokeRes {
  int32  code = 1;
  string msg  = 2;
  bytes  data = 3; // 结果 protojson
  string tid  = 4;
}
```

`data` 使用 `bytes` 承载 JSON，避免网关层二次反序列化，保留 protojson 的 `int64 → string` 语义。

## 路由

路由表位于 `internal/dispatch`，显式 `map[string]Handler`，不使用反射。

| 下游 | method 前缀 |
| --- | --- |
| `app/podengine` | `pod.*`、`image.list` |
| `app/bridgemodbus` | `modbus.*` |

当前开放 19 个方法：

- Podman 4 个只读方法；
- Modbus 15 个 I/O 方法（读取、写入、设备标识）。

未注册 method 返回 `130103`，非法 protojson 返回 `100101`。

## 配置

```yaml
PodEngineConf:
  Endpoints: [127.0.0.1:21010]
  NonBlock: true
  Timeout: 120000
BridgeModbusConf:
  Endpoints: [127.0.0.1:25004]
  NonBlock: true
  Timeout: 120000
Mock: true
```

- `Mock` 仅允许 `Mode: dev` / `test`，生产启动会失败。
- Mock 模式不建立下游连接，四个 Podman 与十五个 Modbus 方法均返回固定数据。

## 新增 method

1. 在对应业务 proto 中增加请求/响应 message。
2. 在 `internal/dispatch` 增加 relay（真实调用）与 mock（可选）。
3. 注册 method 常量与 `Handler`。
4. 同步更新 `web/openapi/src/lib/methods.ts` 与 `docs/open-api/`。

## 启动与验证

```bash
cd facade/openapi
go run . -f etc/openapi.yaml

go test ./facade/openapi/...
```

网关层说明见 [`openapp/opengtw`](../../openapp/opengtw/README.md)，完整接口文档见 [`docs/open-api`](../../docs/open-api/README.md)。

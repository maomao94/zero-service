# app/openapi

`app/openapi` 是开放平台的统一转发服务：接收 `OpenApiRpc.Invoke`，按 `method` 路由到下游 gRPC 服务，并用 protojson 转换 `data`。

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
  string data      = 3; // 该 method 的 protojson UTF-8 原文
  string tid       = 4;
}

InvokeRes {
  string data = 1; // 结果 protojson UTF-8 原文
}
```

`data` 使用 `string` 承载 protojson 文本，避免网关层二次反序列化，保留 protojson 的 `int64 → string` 语义。
业务错误不放在 `InvokeRes` 字段中，统一通过 gRPC status 返回；六位错误码位于 `ErrorInfo.reason`，文案位于 status message。

## 路由

路由表位于 `internal/dispatch`，显式 `map[string]Handler`，不使用反射。

| 下游 | method 前缀 |
| --- | --- |
| `app/podengine` | `pod.*`、`image.list` |
| `app/bridgemodbus` | `modbus.*` |

当前支持以下转发能力：

- Podman 4 个只读方法；
- Modbus 15 个 I/O 方法（读取、写入、设备标识）。
- Modbus 3 个寄存器换算方法和 5 个链路配置管理方法。
- GIS 24 个方法。仓库配置中的 `GisConf` 指向本机 GIS gRPC 服务。

未注册 method 返回 `130004`，非法 protojson 返回 `100020`。

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
GisConf:
  Endpoints: [127.0.0.1:25006]
  NonBlock: true
  Timeout: 120000
```

请求由已注册的路由直接转发到对应下游 gRPC 服务。

将 `NacosConfig.IsRegister` 设为 `true` 后，服务以 `openapi.rpc` 注册到 Nacos。Podman、Modbus 和 GIS 的客户端配置都支持用 `Target` 替换 `Endpoints`。注册名、命名空间和配置示例见 [Nacos 注册与发现](../../docs/open-api/nacos.md)。

## 新增 method

1. 在对应业务 proto 中增加请求/响应 message。
2. 在 `internal/dispatch` 增加下游 gRPC 转发。
3. 注册 method 常量与 `Handler`。
4. 同步更新 `web/openapi/src/lib/methods.ts` 与 `docs/open-api/`。

## 启动与验证

```bash
cd app/openapi
go run . -f etc/openapi.yaml

cd ../..
go test ./app/openapi/...
```

网关层说明见 [`app/opengtw`](../../app/opengtw/README.md)，完整接口文档见 [`docs/open-api`](../../docs/open-api/README.md)。

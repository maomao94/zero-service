# Nacos 注册与发现

`opengtw` 提供 HTTP 入口，默认注册名为 `opengtw`，端口为 `11005`。`openapi` 提供 `openapi.OpenApiRpc/Invoke`，默认注册名为 `openapi.rpc`，端口为 `21018`。HTTP 调用方从 Nacos 取得 `opengtw` 实例后，仍按 HTTP 协议调用 `/v1/gateway`。

## 注册服务

两个服务的 YAML 都有 `NacosConfig`。本地默认关闭注册，接入时分别设置 `IsRegister: true`，并填写 Nacos 地址和命名空间。

```yaml
NacosConfig:
  IsRegister: true
  Host: 127.0.0.1
  Port: 8848
  NamespaceId: public
  ServiceName: openapi.rpc
```

`opengtw` 对应的 `ServiceName` 为 `opengtw`。启用了 Nacos 鉴权的环境还需配置 `Username` 和 `PassWord`。当服务监听 `0.0.0.0` 时，注册 IP 优先取 `POD_IP`，其次取本机内网 IP。`openapi` 注册时会附带 `gRPC_port`，供仓库的 gRPC resolver 识别。

## 发现下游

在 `app/opengtw/etc/opengtw.yaml` 中，用 `Target` 替换 `OpenApiConf.Endpoints`。

```yaml
OpenApiConf:
  Target: nacos://127.0.0.1:8848/openapi.rpc?namespaceid=public
  NonBlock: true
  Timeout: 120000
```

在 `app/openapi/etc/openapi.yaml` 中，同样可按下表替换各下游的 `Endpoints`。

| 配置项 | `Target` 示例 |
| --- | --- |
| `PodEngineConf` | `nacos://127.0.0.1:8848/pod-engine?namespaceid=public` |
| `BridgeModbusConf` | `nacos://127.0.0.1:8848/bridge-modbus?namespaceid=public` |
| `GisConf` | `nacos://127.0.0.1:8848/gis?namespaceid=public` |

每个客户端只保留一种寻址方式。go-zero 在同时配置 `Endpoints` 和 `Target` 时优先使用 `Endpoints`。服务名和命名空间要与实际注册值一致。鉴权环境的 `Target` 可使用 `nacos://<user>:<password>@<host>:<port>/<service>`，实际凭据由部署环境提供。

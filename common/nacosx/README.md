# nacosx

`common/nacosx` 是 [nacos-sdk-go v2](https://github.com/nacos-group/nacos-sdk-go) 的 go-zero 适配层：

- **服务注册**：把 zRPC / HTTP 服务实例注册到 Nacos（临时实例），进程退出时自动反注册。
- **服务发现**：为 gRPC 注册 `nacos://` resolver，`zrpc` 的 `Target` 直接写 Nacos 地址即可。
- **通用订阅**：`Discover` 暴露地址变更回调，供 socket 网关扇出等非 gRPC 场景复用。

包本身不含业务语义；resolver 与 SDK logger 通过导入即生效的 `init()` 注册。

## 服务注册

推荐直接使用 `ServiceConfig`（配置结构），启动阶段调用 `MustRegisterXxx`：

```go
// 仅 gRPC：实例端口与 gRPC_port 元数据一致
c.NacosConfig.MustRegisterGRPC(c.Name, c.ListenOn)

// 仅 HTTP：只注册实例端口，不写 gRPC_port
c.NacosConfig.MustRegisterHTTP(c.Name, httpListenOn)

// HTTP + gRPC 双端口：实例端口用 HTTP，gRPC 客户端通过 metadata.gRPC_port 找到 gRPC 端口
c.NacosConfig.MustRegister(c.Name, httpListenOn, c.ListenOn)
```

要点：

- **实例端口**（Nacos 中的 `port`）：HTTP 与 gRPC 并存时用 HTTP/主端口；仅 gRPC 服务用 gRPC 端口。
- **gRPC 端口**：写入 `metadata.gRPC_port`；消费方优先使用该元数据，缺失时回退实例端口，与 Java 侧 Feign + gRPC 的约定一致。
- socketgtw 这类同时暴露 HTTP 与 gRPC 的服务，注册为 `MustRegister(name, http_addr, grpc_listen_addr)`。
- 注册 IP 解析顺序：`AdvertiseIP`（显式配置）→ 实例监听地址中的具体 IP → gRPC 监听地址中的具体 IP → `POD_IP` → 本机内网 IP。`0.0.0.0`、`::`、`127.0.0.1`、`::1` 会被跳过；都推导不到时启动报错，不会注册通配/回环地址。
- 未指定 `CacheDir` / `LogDir` 时默认落到 `$TMPDIR/nacos/{cache,log}`，不会污染进程工作目录。
- 底层仍可直接调用 `RegisterService(opts)`；`ServiceConfig.Register` 返回 error，`MustRegister*` 失败时通过 `logx.Must` 退出，适合启动阶段。

`Options` 与选项函数：

| 字段 / 选项 | 说明 | 默认值 |
| --- | --- | --- |
| `ServiceName` | 注册的服务名，与调用方 `Target` 中的路径一致 | 必填 |
| `ListenOn` | 服务监听地址，用于推导注册 IP 与端口 | 必填 |
| `AdvertiseIP` | 注册到 Nacos 的实例 IP，缺省按监听地址/`POD_IP`/内网 IP 推导 | 自动 |
| `Weight` / `WithWeight` | 实例权重 | `1.0` |
| `Cluster` / `WithCluster` | 集群名 | 空 |
| `Group` / `WithGroup` | 分组名 | 空 |
| `Metadata` / `WithMetadata` | 附加元数据；gRPC 端口由注册方法自动写入 `gRPC_port` | 空 map |

## 服务发现

客户端只需要在配置里写 `Target`，无需额外代码：

```yaml
StreamEventConf:
  Target: nacos://nacos:nacos@127.0.0.1:8848/stream-event?namespaceid=public&timeout=5s
```

URL 格式为 `nacos://[user:passwd]@host:port/service?param=value`，支持的查询参数：

| 参数 | 说明 |
| --- | --- |
| `namespaceid` | 命名空间，缺省 `public` |
| `timeout` | 请求超时，Go duration 格式（如 `5s`），内部换算为毫秒；省略时用 SDK 默认 10s |
| `groupName` | 分组名，缺省 `DEFAULT_GROUP` |
| `clusters` | 集群过滤，逗号分隔，如 `clusters=DEFAULT,OTHER` |
| `notLoadCacheAtStart` | 是否忽略本地缓存启动，默认 `true` |
| `updateCacheWhenEmpty` | 实例为空时是否刷新缓存并触发回调，默认 `false` |
| `appName` / `cacheDir` | 客户端标识与本地缓存目录 |
| `logLevel` / `logDir` | 客户端日志配置；受全局 logger 初始化时机限制，通常请改用环境变量或 `SetUpLogger`（见「日志」） |

`cacheDir` 等 URL 参数会被环境变量 `NACOS_CACHE_DIR` 覆盖；`logLevel`、`logDir` 虽然也会解析进 client 配置，但全局 logger 只在包初始化时读取一次，实际调整请使用 `NACOS_LOG_LEVEL`、`NACOS_LOG_DIR` 或 `SetUpLogger`。`notLoadCacheAtStart`、`updateCacheWhenEmpty` 同理对应 `NACOS_NOT_LOAD_CACHE_AT_START`、`NACOS_UPDATE_CACHE_WHEN_EMPTY`。缓存目录缺省为 `$TMPDIR/nacos/cache`。

发现行为：

- 只返回健康且启用的实例；地址去重并按字典序排序，集合未变化时不重复上报。
- 地址端口优先取 `metadata.gRPC_port`，缺失时回退实例的 `port`（兼容 Java 侧仅注册主端口的服务）。
- 以 Nacos 订阅推送为主通道；启动时立即全量拉取一次，之后按 5s → 10s → … → 5min 指数退避兜底。
- 订阅失败会打印错误并继续依靠轮询，不会静默丢地址。

## 非 gRPC 场景：Discover

需要自行维护客户端集合（例如 socket 网关扇出）时，直接复用 `Discover`，不要另写一套订阅逻辑：

```go
discovery, err := nacosx.Discover(context.Background(), c.Target, func(addrs []string) {
	// 已去重、已排序，仅在实例集合变化时串行回调
})
if err != nil {
	logx.Must(err)
}
defer discovery.Close()
```

`Discover` 与 gRPC resolver 共用同一套地址解析、订阅、兜底拉取和日志；`Close` 幂等，会取消订阅并释放 Nacos client。

## 日志

所有日志以 `[nacos]` 为前缀。包初始化时会以 `error` 级别、同时输出 stdout 与文件的方式配置 SDK 全局 logger，默认目录 `$TMPDIR/nacos/log`，可用 `NACOS_LOG_LEVEL`、`NACOS_LOG_DIR` 覆盖。

SDK 的全局 logger 只初始化一次，因此各 client 的 `LogLevel` / `LogDir` 在包初始化后不再生效。需要调整时在启动早期调用：

```go
_ = nacosx.SetUpLogger(nacosx.LoggerConfig{
	Level:          "warn",
	LogDir:         "/var/log/nacos",
	AppendToStdout: true,
})
```

## 相关文档

- [部署指南](../../docs/deployment.md) — 集群部署中的服务发现与负载均衡
- [服务端口清单](../../docs/service-ports.md) — 各服务注册名与端口

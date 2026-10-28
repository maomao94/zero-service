# nacosx

`common/nacosx` 是 [nacos-sdk-go v2](https://github.com/nacos-group/nacos-sdk-go) 的 go-zero 适配层，提供两件事：

- **服务注册**：把 zRPC 服务实例注册到 Nacos（临时实例），进程退出时自动反注册。
- **服务发现**：向 gRPC 注册 `nacos://` resolver，`zrpc` 的 `Target` 直接写 Nacos 地址即可按服务名解析实例，并由 gRPC 负载均衡轮询。

包本身不含业务语义，注册与解析都通过导入即生效的 `init()` 完成。

## 服务注册

```go
import "github.com/maomao94/zero-service/common/nacosx"

sc := []constant.ServerConfig{
	*constant.NewServerConfig("127.0.0.1", 8848),
}
cc := &constant.ClientConfig{
	NamespaceId:         "public",
	TimeoutMs:           5000,
	NotLoadCacheAtStart: true,
	LogDir:              "/tmp/nacos/log",
	CacheDir:            "/tmp/nacos/cache",
	LogLevel:            "error",
}

opts := nacosx.NewNacosConfig("trigger.rpc", c.ListenOn, sc, cc,
	nacosx.WithCluster("DEFAULT"),
	nacosx.WithGroup("DEFAULT_GROUP"),
)
if err := nacosx.RegisterService(opts); err != nil {
	logx.Must(err)
}
```

`RegisterService` 的注册 IP 由 `ListenOn` 推导：监听 `0.0.0.0` 或空主机时，优先取环境变量 `POD_IP`，其次取本机内网 IP；显式配置的主机地址原样使用。

`Options` 与选项函数：

| 字段 / 选项 | 说明 | 默认值 |
| --- | --- | --- |
| `ServiceName` | 注册的服务名，与调用方 `Target` 中的路径一致 | 必填 |
| `ListenOn` | 服务监听地址，用于推导注册 IP 与端口 | 必填 |
| `Weight` / `WithWeight` | 实例权重 | `1.0` |
| `Cluster` / `WithCluster` | 集群名 | 空 |
| `Group` / `WithGroup` | 分组名 | 空 |
| `Metadata` / `WithMetadata` | 附加元数据 | 空 map |

## 服务发现

客户端只需要在配置里写 `Target`，无需额外代码：

```yaml
Target: nacos://127.0.0.1:8848/trigger.rpc?namespaceid=public&timeout=5000s
```

URL 格式为 `nacos://[user:passwd]@host:port/service?param=value`，支持的查询参数：

| 参数 | 说明 |
| --- | --- |
| `namespaceid` | 命名空间，缺省为 `public` |
| `timeout` | 连接与请求超时，如 `5000s` |
| `groupName` | 分组名 |
| `clusters` | 集群过滤 |
| `notLoadCacheAtStart` / `updateCacheWhenEmpty` | 本地缓存行为 |
| `appName` / `logLevel` / `logDir` / `cacheDir` | 客户端标识与日志目录 |

`logLevel`、`logDir`、`cacheDir` 会被环境变量 `NACOS_LOG_LEVEL`、`NACOS_LOG_DIR`、`NACOS_CACHE_DIR` 覆盖，便于容器内统一调整。解析器只返回健康实例，并按地址排序更新状态，避免相同地址列表反复触达负载均衡。

## 日志

包初始化时会以 `error` 级别、追加 stdout 的方式配置 Nacos SDK 全局 logger。需要调整时在启动早期调用：

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

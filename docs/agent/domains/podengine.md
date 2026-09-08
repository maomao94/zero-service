# Pod Engine Docker 编排规范

## 适用范围

修改 `app/podengine`、`common/dockerx`、`common/executorx` 或 Docker 容器编排相关功能时读取。

## 服务定位

- **podengine** — zRPC Docker 容器编排服务，提供类 Kubernetes Pod API（创建/启动/停止/重启/删除/查询容器、镜像列表）。
- **dockerx** — Docker 客户端辅助函数，含 OTel 追踪注入。
- **executorx** — go-zero ChunkExecutor 封装，提供分块消息推送器。

依据：`app/podengine/internal`、`common/dockerx/dockerx.go`、`common/executorx/chunkmessagespusher.go`。

## podengine — 容器编排服务

### 入口与配置

- `podengine.go` 标准 go-zero zRPC 入口，含 Nacos 服务注册和 RPC 拦截器。
- 唯一导入 `_ "zero-service/common/carbonx"` 进行 carbon 时间库副作用初始化。
- `config.Config` 嵌入 `zrpc.RpcServerConf`，额外包含 `NacosConfig`（可选）和 `DockerConfig map[string]string`（节点名 → Docker Host 地址，可选）。
- `ServiceContext` 持有 `DockerClients map[string]*client.Client`（受 `sync.RWMutex` 保护），始终创建名为 `"local"` 的客户端。`GetDockerClient(name)` 默认返回 local。

依据：`app/podengine/podengine.go`、`app/podengine/internal/config/config.go`、`app/podengine/internal/svc/servicecontext.go`。

### RPC 方法

9 个 RPC 方法 (`internal/server/podengineserver.go`)：

| RPC | 功能 |
|-----|------|
| CreatePod | 创建容器：镜像、环境变量、端口、卷、资源限制、重启策略 |
| StartPod | 启动容器 |
| StopPod | 停止容器 |
| RestartPod | 重启容器 |
| GetPod | 容器 inspect |
| ListPods | 容器列表，支持按 ID/名称/标签过滤和 offset/limit 分页 |
| DeletePod | 删除容器 |
| GetPodStats | 容器状态统计 |
| ListImages | Docker 镜像列表 |

### CreatePod 约定

最复杂的 RPC，构建容器的完整配置：
- **镜像**: 从请求获取，必填。
- **环境变量**: 从 `map[string]string` 构建。
- **端口映射**: 格式 `host:container`，解析并绑定。
- **资源限制**: `map[string]string` 格式（cpu/memory/cpuRequest/memoryRequest）。注意 memory 值带单位（K/M/G/T），在 podengine 中解析。
- **卷挂载**: 格式 `host:container[:ro]`，`ro` 表示只读。
- **重启策略**: 支持 `no`、`always`、`on-failure`、`unless-stopped`。
- **容器命名**: `{podName}-{containerName}`（小写）。
- **终止宽限期**: 默认 60 秒。
- **返回**: `PodPb` with phase `PENDING` after creation — 容器阶段模拟 Kubernetes phases: `PENDING` → `RUNNING` → `SUCCEEDED`/`STOPPED`/`UNKNOWN`。
- 使用 protobuf validation (`.Validate()` 方法) 校验请求。

依据：`app/podengine/internal/logic/createpodlogic.go`。

### 多 Docker Host

- `DockerClients` 为 `map[string]*client.Client`，读时使用 `RLock`。
- `GetDockerClient(name)` — name 为空时默认返回 local。
- 新增 remote host 时更新 `DockerConfig` 配置映射。

依据：`app/podengine/internal/svc/servicecontext.go`。

## dockerx — Docker 客户端辅助

所有函数为纯工具函数，无 struct/interface：

| 函数 | 用途 |
|------|------|
| `MustNewClient(ops ...client.Opt)` | 创建 Docker 客户端，自动注入 OTel `TracerProvider`，失败 panic |
| `ParseContainerEnv(env []string)` | `KEY=VALUE` 列表 → `map[string]string` |
| `BuildEnvList(envMap map[string]string)` | `map[string]string` → `[]string`（逆向） |
| `ExtractContainerPorts(networkSettings)` | 端口格式化: `{hostIP}:{hostPort}->{containerPort}/{proto}` |
| `ExtractContainerVolumeMounts(mounts)` | 卷格式化: `{source}:{destination}:{mode}` |
| `ParseContainerResources(resources)` | Docker 资源 → `map[string]string` |

资源解析细节：
- CPU quota ÷ 100000 → cpu 核数
- CPU shares ÷ 1024 → cpuRequest 核数
- Memory 直接使用字节值（单位解析在 podengine 中）
- MemoryReservation → memoryRequest

依据：`common/dockerx/dockerx.go`。

## executorx — 分块消息推送器

- `ChunkMessagesPusher` 封装 go-zero `executors.ChunkExecutor`。
- `NewChunkMessagesPusher(chunkSender, chunkBytes)` — 按字节阈值分块，达到阈值时调用 `chunkSender([]string)` 回调。
- `Write(val string)` — 添加消息，线程安全（`sync.Mutex`）。
- 只支持字符串消息类型，`execute()` 内部类型断言。
- 使用 go-zero 的 `NewChunkExecutor` + `WithChunkBytes` 选项。

依据：`common/executorx/chunkmessagespusher.go`。

## 反模式

- 在多 Docker host 场景中不通过 `GetDockerClient()` 获取客户端，直接硬编码 local。
- 在 podengine 中重复实现 dockerx 已有的辅助函数。
- 资源限制值不处理单位（如 `128M` 直接当数字）。
- 容器生命周期操作不验证容器状态就执行（如对已停止容器调 StopPod）。
- 使用裸 goroutine 管理容器操作，不使用 context 超时。
- `DockerClients` 写入时不持有写锁。

## 验证

- podengine: 验证所有 9 个 RPC，特别是 CreatePod 的参数解析、资源限制、端口和卷映射。
- podengine: 验证多 Docker Host 场景下的客户端选择和故障隔离。
- dockerx: 验证资源解析的边界值（0、极大值、异常值）。
- executorx: 验证分块逻辑、字节阈值边界、并发写入。

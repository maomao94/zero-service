# PodEngine 容器编排服务

`podengine.rpc` 是容器编排服务（默认端口 21010），提供类 Kubernetes Pod API 的容器生命周期管理能力，当前适配 Docker 运行时，可管理本地与远程多节点 Docker。

## 服务职责

- **Pod 生命周期管理**：创建、启动、停止、重启、删除容器，抽象为 Pod 的期望状态（Spec）与观测状态（Phase）。
- **查询与统计**：容器详情/列表（支持按名称、ID、标签过滤与分页）、容器资源统计（CPU/内存/网络/存储）、镜像列表。
- **多节点编排**：请求携带 `node` 参数指定 Docker 运行节点（默认 `local`），通过配置的 Docker Host 地址连接对应 daemon。

## 配置

配置文件：`app/podengine/etc/podengine.yaml`。关键项：

| 配置项 | 说明 | 默认值 |
| --- | --- | --- |
| `ListenOn` | gRPC 监听地址 | `0.0.0.0:21010` |
| `Timeout` | 单次调用上限（毫秒） | `120000` |
| `DockerConfig` | 节点名 → Docker Host 地址映射（如 `tcp://127.0.0.1:2375`），服务始终创建名为 `local` 的客户端 | 空 |
| `NacosConfig.IsRegister` | 是否注册到 Nacos（可选） | `false` |

## 关键接口

完整 RPC 定义见 [`app/podengine/podengine.proto`](../../app/podengine/podengine.proto)（`service PodEngine`），字段与校验以 proto 为权威。

| RPC | 说明 |
| --- | --- |
| `CreatePod` | 创建容器：镜像、环境变量、端口映射（`host:container`）、卷挂载（`host:container[:ro]`）、资源限制、重启策略 |
| `StartPod` | 启动容器 |
| `StopPod` | 停止容器（支持 `force`） |
| `RestartPod` | 重启容器 |
| `GetPod` | 容器 inspect |
| `ListPods` | 容器列表，支持名称/ID/标签过滤与 offset/limit 分页 |
| `DeletePod` | 删除容器（支持 `force`、`removeVolumes`） |
| `GetPodStats` | 容器资源统计（CPU、内存、网络、存储） |
| `ListImages` | Docker 镜像列表，支持分页与引用过滤 |

## 关键约定

- 容器命名格式 `{podName}-{containerName}`（小写）；创建后返回的 Pod 处于 `PENDING` 阶段。
- Pod 阶段模拟 Kubernetes：`PENDING` → `RUNNING` → `SUCCEEDED` / `FAILED` / `STOPPED`。
- 重启策略支持 `no` / `onFailure` / `always`；网络模式支持 `bridge` / `host` / `none`。
- 资源限制以 `map[string]string` 传入，memory 值带单位（K/M/G/T），在 podengine 中解析。
- 终止宽限期默认 60 秒。

## 部署

- 需要可达的 Docker daemon：本地节点使用默认 unix socket，远程节点在 `DockerConfig` 中配置 `tcp://host:port`。
- 标准 go-zero zRPC 服务，启动方式：

```bash
./podengine -f etc/podengine.yaml
```
- 公共库：`common/dockerx`（Docker 客户端与解析辅助）、`common/executorx`（分块消息推送）。

## 权威契约

- RPC 契约：[`app/podengine/podengine.proto`](../../app/podengine/podengine.proto)
- 服务配置：`app/podengine/etc/podengine.yaml`
- 公共库：[`common/dockerx`](../../common/dockerx/dockerx.go)

# Redis 7 Docker 部署

用于本地开发和测试业务系统（缓存、分布式锁、票据、业务序号）。

## 文件说明

| 文件 | 作用 |
| --- | --- |
| `deploy.sh` | 部署与运维脚本 |
| `docker-compose.yaml` | 容器编排配置 |
| `data/` | 持久化目录（AOF，gitignore） |

## 快速开始

```bash
bash deploy/redis/deploy.sh          # 部署
bash deploy/redis/deploy.sh status   # 状态
bash deploy/redis/deploy.sh cli      # 进入 redis-cli
redis-cli -h 127.0.0.1 -p 36379 -a 'G62m50oigInC30sf' ping
```

## 连接信息

| 项 | 值 |
| --- | --- |
| Host / Port | `127.0.0.1:36379`（`REDIS_HOST_PORT` 可改宿主机端口） |
| 密码 | `G62m50oigInC30sf`（`REDIS_PASSWORD` 可改） |
| 容器内端口 | `6379` |

默认使用 `36379:6379` 直映射。端口冲突时可只修改宿主机映射端口：

```bash
REDIS_HOST_PORT=36380 bash deploy/redis/deploy.sh
```

密码通过环境变量覆盖（需与各业务服务配置保持一致）：

```bash
REDIS_PASSWORD='your-strong-password' bash deploy/redis/deploy.sh
```

## 使用方

以下服务连接 `127.0.0.1:36379`（宿主机）或 `host.docker.internal:36379`（容器内）：

- `app/live`、`app/trigger`、`app/oryxserver`：`etc/*.yaml` 的 `Redis`
- `deploy/livekit`：`livekit-server.yaml` / `egress.yaml` / `sip.yaml` 的 `redis.address`

> 注意：修改 `REDIS_PASSWORD` 或 `REDIS_HOST_PORT` 后，需同步更新上述配置。

## 重新初始化

```bash
bash deploy/redis/deploy.sh stop
rm -rf deploy/redis/data
bash deploy/redis/deploy.sh
```

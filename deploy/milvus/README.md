# Milvus Docker 部署

Milvus Standalone 向量数据库部署，包含 etcd 元数据存储、MinIO 对象存储和 Attu 管理界面，用于开发测试及 AI Solo 项目的 RAG 功能。

## 文件说明

| 文件 | 作用 |
| --- | --- |
| `docker-compose.yaml` | Milvus、etcd、MinIO 和 Attu 容器编排配置 |

## 启动与管理

在项目根目录执行：

```bash
docker compose -f deploy/milvus/docker-compose.yaml up -d
docker compose -f deploy/milvus/docker-compose.yaml ps
docker compose -f deploy/milvus/docker-compose.yaml logs -f
docker compose -f deploy/milvus/docker-compose.yaml down
```

Compose 使用命名卷持久化 etcd、MinIO 和 Milvus 数据。执行 `down` 会保留数据；如需彻底删除数据，可显式执行：

```bash
docker compose -f deploy/milvus/docker-compose.yaml down -v
```

`down -v` 会永久删除上述命名卷中的数据。

## 连接信息

| 服务 | 宿主机端口 | 用途 |
| --- | --- | --- |
| Milvus | `19530` | gRPC 客户端连接 |
| Milvus | `9091` | 健康检查 |
| MinIO | `9000` | 对象存储 API |
| MinIO | `9001` | MinIO Console |
| Attu | `3000` | Milvus Web 管理界面 |

Milvus 客户端地址为 `localhost:19530`，Attu 页面为 `http://localhost:3000`。MinIO 默认账号密码为 `minioadmin` / `minioadmin`，仅适用于开发测试；部署到共享或生产环境前应修改凭据，并按需限制宿主机端口的网络访问范围。

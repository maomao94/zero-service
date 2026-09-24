# PostgreSQL 16 Docker 部署

用于本地开发和测试业务系统，也可按需作为数据迁移过程中的临时数据库。

## 文件说明

| 文件 | 作用 |
| --- | --- |
| `deploy.sh` | 部署与运维脚本 |
| `docker-compose.yaml` | 容器编排配置 |
| `roles.sql` | 业务账号体系脚本（app_user/admin_user/query_user） |
| `data/` | 持久化目录（gitignore） |

## 快速开始

```bash
bash deploy/postgres/deploy.sh          # 部署
bash deploy/postgres/deploy.sh status   # 状态
bash deploy/postgres/deploy.sh psql     # 进入 psql
psql "host=127.0.0.1 port=5432 dbname=postgres user=postgres password=postgres"
```

## 连接信息

| 项 | 值 |
| --- | --- |
| Host / Port | `127.0.0.1:5432`（`POSTGRES_HOST_PORT` 可改宿主机端口） |
| 用户 / 密码 | `postgres` / `postgres` |
| 默认库 | `postgres` |

默认使用 PostgreSQL 原生端口 `5432:5432`。本机需要同时运行 openGauss 时，可只修改宿主机映射端口：

```bash
POSTGRES_HOST_PORT=5433 bash deploy/postgres/deploy.sh
```

容器内端口始终为 `5432`。部署脚本会通过 PostgreSQL 配置将数据库默认时区持久化为 `Asia/Shanghai`，即中国东八区。

## 业务账号体系

`roles.sql` 与 kingbase / opengauss / mysql 部署保持同一账号口径，在目标业务库执行
（角色是实例级对象，多个业务库共用同一套角色）：

```bash
# 先建业务库（以 zero 为例；已存在会报错，忽略即可）
docker exec pgsql psql -U postgres -d postgres -c 'create database zero;'
# 再执行账号授权脚本
docker exec -i pgsql psql -U postgres -d zero < deploy/postgres/roles.sql
```

| 账号 | 初始密码（本地示例） | 权限 |
| --- | --- | --- |
| `app_user` | `App_user@123` | public 模式建表 + 业务表查增删改 |
| `admin_user` | `Admin_user@123` | 管理角色和用户 + 建库，业务表与 app_user 互通 |
| `query_user` | `Query_user@123` | 仅 SELECT |

脚本仅在角色不存在时创建，已有账号密码不会被重置。生产环境必须修改以上示例密码。

## 重新初始化

```bash
bash deploy/postgres/deploy.sh stop
rm -rf deploy/postgres/data
bash deploy/postgres/deploy.sh
```

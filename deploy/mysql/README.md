# MySQL 8.4 Docker 部署

用于本地开发和测试业务系统。

## 文件说明

| 文件 | 作用 |
| --- | --- |
| `deploy.sh` | 部署与运维脚本 |
| `docker-compose.yaml` | 容器编排配置 |
| `roles.sql` | 业务库建库 + 三类业务账号授权脚本 |
| `data/` | 持久化目录（gitignore） |

## 快速开始

```bash
bash deploy/mysql/deploy.sh          # 部署
bash deploy/mysql/deploy.sh status   # 状态
bash deploy/mysql/deploy.sh mysql    # 进入 mysql 客户端
mysql -h127.0.0.1 -P3306 -uroot -proot
```

## 连接信息

| 项 | 值 |
| --- | --- |
| Host / Port | `127.0.0.1:3306`（`MYSQL_HOST_PORT` 可改宿主机端口） |
| 用户 / 密码 | `root` / `root` |
| 服务端时区 | `+08:00`（东八区） |
| 字符集 | `utf8mb4` / `utf8mb4_0900_ai_ci` |

默认使用 MySQL 原生端口 `3306:3306`。本机端口冲突时只修改宿主机映射端口，容器内端口始终为 `3306`：

```bash
MYSQL_HOST_PORT=3307 bash deploy/mysql/deploy.sh
```

注意：本机若同时运行 go-zero-looklook 的 MySQL（宿主机端口 `33069`），与本部署互不影响；
两者容器名分别为 `mysql`（looklook）与 `mysql8`（本部署）。

## 业务库与账号体系

`roles.sql` 与 kingbase / opengauss / postgres 部署保持同一账号口径，业务库名统一写作 `zero`，
请按需全局替换为实际业务库名后以 root 执行（幂等，可对多个业务库各跑一遍）：

```bash
docker exec -i mysql8 mysql -uroot -proot < deploy/mysql/roles.sql
```

| 账号 | 初始密码（本地示例） | 权限 |
| --- | --- | --- |
| `app_user` | `App_user@123` | 业务库内建表改表 + 查增删改 |
| `admin_user` | `Admin_user@123` | 建库删库、建账号、业务库全权限（可授权） |
| `query_user` | `Query_user@123` | 业务库仅 SELECT |

说明：

- MySQL 账号为 `用户@主机` 形式，脚本统一使用 `'%'` 允许任意客户端地址连接
- 库级授权自动覆盖库内后续新建的表，无需 PG 的 default privileges
- 脚本仅在账号不存在时创建，已有账号的密码不会被重置
- 生产环境必须修改以上示例密码，并通过受控渠道分发

## 重新初始化

```bash
bash deploy/mysql/deploy.sh stop
rm -rf deploy/mysql/data
bash deploy/mysql/deploy.sh
```

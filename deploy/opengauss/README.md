# openGauss 7.0 Docker 部署（持久化）

高斯数据库本机 Docker 单机部署，用于本地开发和测试业务系统。

> 替代原无挂卷的本地高斯容器（已清理），新部署统一持久化到 `./data`，`down` 不丢数据。

## 文件说明

| 文件 | 作用 |
| --- | --- |
| `deploy.sh` | 部署与运维脚本（单脚本子命令模式） |
| `docker-compose.yaml` | 容器编排配置（含逐行注释） |
| `roles.sql` | 业务账号体系脚本（app_user/admin_user/query_user） |
| `data/` | 数据库持久化目录（gitignore，不入库） |

## 快速开始

```bash
# 部署（自动拉取 opengauss/opengauss-server:latest）
bash deploy/opengauss/deploy.sh

# 查看状态/日志
bash deploy/opengauss/deploy.sh status
bash deploy/opengauss/deploy.sh logs

# 进入数据库
bash deploy/opengauss/deploy.sh psql
# 或宿主机 psql
psql "host=127.0.0.1 port=5432 dbname=postgres user=gaussdb password=Gauss@123"
```

## 连接信息

| 项 | 值 |
| --- | --- |
| Host / Port | `127.0.0.1:5432`（`OPENGAUSS_HOST_PORT` 可改宿主机端口） |
| 用户 / 密码 | `gaussdb` / `Gauss@123`（`omm` 同密码） |
| 默认库 | `postgres` |
| 容器内端口 | `5432` |

密码需满足复杂度（>=8位，含大小写、数字、特殊字符 `#?!@$%^&*-`），修改后需 `stop` 后清空 `data` 重建。

默认使用 openGauss 原生端口 `5432:5432`。端口冲突时可只修改宿主机映射端口：

```bash
OPENGAUSS_HOST_PORT=15432 bash deploy/opengauss/deploy.sh
```

容器内端口始终为 `5432`。部署脚本会通过 `gs_guc` 将数据库默认时区持久化为 `Asia/Shanghai`，即中国东八区。

## 业务账号体系

`roles.sql` 与 kingbase / postgres / mysql 部署保持同一账号口径，在目标业务库执行
（角色是实例级对象，多个业务库共用同一套角色）：

```bash
# 先建业务库（以 zero 为例；已存在会报错，忽略即可）
docker exec opengauss gosu omm bash -c 'export GAUSSHOME=/usr/local/opengauss; export PATH=$GAUSSHOME/bin:$PATH; export LD_LIBRARY_PATH=$GAUSSHOME/lib:/scws/lib:$LD_LIBRARY_PATH; gsql -d postgres -U omm -c "create database zero"'
# 再执行账号授权脚本
docker exec -i opengauss gosu omm bash -c 'export GAUSSHOME=/usr/local/opengauss; export PATH=$GAUSSHOME/bin:$PATH; export LD_LIBRARY_PATH=$GAUSSHOME/lib:/scws/lib:$LD_LIBRARY_PATH; gsql -d zero -U omm' < deploy/opengauss/roles.sql
```

| 账号 | 初始密码（本地示例） | 权限 |
| --- | --- | --- |
| `app_user` | `App_user@123` | public 模式建表 + 业务表查增删改 |
| `admin_user` | `Admin_user@123` | 管理角色和用户 + 建库，业务表与 app_user 互通 |
| `query_user` | `Query_user@123` | 仅 SELECT |

脚本仅在角色不存在时创建，已有账号密码不会被重置。生产环境必须修改以上示例密码。

## 重新初始化

`GS_PASSWORD` 等环境变量仅在 `data` 为空时生效。换密码需 `bash deploy/opengauss/deploy.sh stop` 后 `rm -rf data` 再重建。

## 参考

- 官方容器安装: https://docs.opengauss.org/zh/docs/latest/installation_guide/installing_the_container_image.html
- 镜像: `opengauss/opengauss-server:latest`（`docker pull`）

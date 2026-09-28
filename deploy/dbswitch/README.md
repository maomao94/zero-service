# dbswitch 独立部署

dbswitch 是 dromara 社区开源的异构数据库迁移同步工具（Web 管理端为 dbswitch-admin），
作为 KDTS 的替代方案独立编排，不与任何数据库容器共用 Compose 项目和网络生命周期。

对照官方仓库: https://gitee.com/dromara/dbswitch

## 文件说明

| 文件 | 作用 |
| --- | --- |
| `deploy.sh` | dbswitch 部署与运维脚本 |
| `docker-compose.yaml` | dbswitch-admin 容器编排配置 |
| `data/` | H2 配置库持久化目录（对应容器 /tmp，gitignore） |

## 快速开始

```bash
# 部署或重复启动
bash deploy/dbswitch/deploy.sh

# 查看状态和日志
bash deploy/dbswitch/deploy.sh status
bash deploy/dbswitch/deploy.sh logs

# 停止并移除容器，保留任务数据
bash deploy/dbswitch/deploy.sh stop
```

访问地址：`http://127.0.0.1:9088`，默认账号 `admin / 123456`。

## 数据库连接

容器独立组网，源库/目标库连接在 Web 页面「连接管理」中手动配置。
容器内访问宿主机映射的数据库端口必须使用 `host.docker.internal` 作为主机地址（`127.0.0.1` 指向容器自身）。

Web 端建连接参考（各库账号密码见对应 deploy 目录的 compose 注释）:

| 数据库 | JDBC URL | 类型选择 |
| --- | --- | --- |
| 金仓 (54321, pg 兼容模式) | `jdbc:kingbase8://host.docker.internal:54321/zero` | KINGBASE |
| 金仓（走 PG 方言） | `jdbc:postgresql://host.docker.internal:54321/zero` | POSTGRESQL |
| openGauss (15432) | `jdbc:opengauss://host.docker.internal:15432/postgres` | OPENGAUSS |
| PostgreSQL (5432) | `jdbc:postgresql://host.docker.internal:5432/postgres` | POSTGRESQL |
| MySQL (3306) | `jdbc:mysql://host.docker.internal:3306/<库名>` | MYSQL |

## 目标端为金仓的推荐配置

金仓为 PG 兼容模式时，目标端建议选择 `POSTGRESQL` 类型 + PostgreSQL 官方驱动连接：

- 类型转换按 PG 方言处理，规避 dbswitch KINGBASE 方言已知的目标端字段类型兼容问题
- 可启用 copy 批量写入（`writer-engine-insert: false`），比 insert 快得多
- 注意 `target-schema` 只能填一个，且需与实际业务 schema 匹配（金仓 pg 模式默认 `public`）

## 使用流程

连接管理（源端/目标端）-> 任务配置 -> 发布任务 -> 手动或调度执行 -> 查看调度日志与数据目录结果。

首次全量迁移建议：目标端 `target-drop: true`（存在表先删后建）；结构+数据一次性同步。
变化量同步（`change-data-sync`）要求表有主键，千万级以上数据量慎用。

# KingbaseES V9R1C10 Docker 部署

金仓数据库本机 Docker 单机部署（PG 兼容模式），用于 gormx 金仓适配联调。

## 文件说明

| 文件 | 作用 |
| --- | --- |
| `deploy.sh` | 部署与运维脚本（单脚本子命令模式） |
| `docker-compose.yaml` | 容器编排配置（含逐行注释） |
| `init.sql` | 业务建库脚本（初始化后按需执行） |
| `roles.sql` | 业务程序、管理员、现场查询三类账号及授权脚本 |
| `KingbaseES_*_Docker.tar` | 官网下载的镜像 tar（gitignore，不入库） |
| `data/` | 数据库持久化目录（gitignore，不入库） |

## 快速开始

```bash
# 1. 从官网下载镜像 tar 放到本目录
#    https://www.kingbase.com.cn/download.html

# 2. 部署
bash deploy/kingbase/deploy.sh
# 端口冲突时只修改宿主机映射端口（容器内仍为 54321）
KINGBASE_HOST_PORT=54322 bash deploy/kingbase/deploy.sh

# 3. 建业务库（可选，也可用数据库管理工具在 kingbase 库上执行）
docker exec -i kingbase ksql -Usystem -d kingbase -p 54321 < deploy/kingbase/init.sql

# 4. 在业务库中创建账号并授权；示例使用 zero，请替换成实际库名
docker exec -i kingbase ksql -Usystem -d zero -p 54321 < deploy/kingbase/roles.sql
```

`roles.sql` 需要对每个业务库分别执行一次。它会创建 `app_user`（初始密码 `App_user@123`，业务程序读写）、
`admin_user`（初始密码 `Admin_user@123`，管理员及业务数据读写）和 `query_user`（初始密码 `Query_user@123`，仅查询）；
角色是实例级对象，多个业务库共用相同角色。首次执行请使用 `system` 等有角色管理权限的账号。脚本仅在角色不存在时创建账号，
已有账号的密码及 `CREATEROLE`/`CREATEDB` 属性不会被修改；若账号此前已存在，请先确认 `admin_user` 具备管理所需属性，
并自行修改或核验密码。密码可用 `alter user app_user with password '新密码';`（替换用户名）单独修改。脚本中的初始密码仅用于本机开发，
部署到共享或生产环境前必须修改，并通过受控渠道分发。openGauss、PostgreSQL 和 MySQL 目录下也提供同口径的 `roles.sql`，四种数据库账号与授权口径保持一致。

数据库部署不包含 DTS 迁移工具；需要迁移时使用独立工具，按实际宿主机端口配置源库和目标库连接。

## KDTS 独立部署

KDTS-WEB 已从本目录的数据库 Compose 中拆出，独立配置位于 `deploy/kdts/`，不会随 Kingbase 数据库部署自动启动。
原有 `kdts_data` 已移动为 `deploy/kdts/data`，其中的 H2 任务数据保留。

```bash
bash deploy/kdts/deploy.sh             # 部署或启动 KDTS
bash deploy/kdts/deploy.sh status      # 查看状态
bash deploy/kdts/deploy.sh logs        # 查看日志
bash deploy/kdts/deploy.sh stop        # 停止并移除容器，保留任务数据
```

详细连接方式、端口和宿主机数据库访问配置见 [`deploy/kdts/README.md`](../kdts/README.md)。

## deploy.sh 命令

```bash
bash deploy/kingbase/deploy.sh            # 部署（默认，幂等可重跑）
bash deploy/kingbase/deploy.sh stop       # 移除容器，保留数据
bash deploy/kingbase/deploy.sh restart    # 重启容器（数据库进程异常时恢复）
bash deploy/kingbase/deploy.sh status     # 容器状态 + 授权剩余天数
bash deploy/kingbase/deploy.sh logs       # 最近 100 行容器日志
bash deploy/kingbase/deploy.sh ksql       # 进入 ksql 交互（容器内 trust 免密，system 连默认库）
```

**重新初始化**（危险操作，脚本不提供一键清空命令）：`stop` 后手动 `rm -rf data`，再执行 `deploy.sh`——检测到空目录会自动重新 initdb。

## 连接信息

| 项 | 值 |
| --- | --- |
| Host / Port | `127.0.0.1:54321`（`KINGBASE_HOST_PORT` 可改宿主机端口） |
| 用户 / 密码 | `system` / `12345678ab` |
| 默认库 | `kingbase`（金仓默认库，相当于 PG 的 `postgres`） |
| 认证 | scram-sha-256（PG 兼容） |
| gormx DSN | `kingbase://system:12345678ab@127.0.0.1:54321/kingbase?sslmode=disable` |

数据库管理工具（TablePlus / DBeaver / Navicat 等）可直接以 PostgreSQL 协议连接，默认端口 54321；
若设置 `KINGBASE_HOST_PORT`，客户端连接时使用对应的宿主机端口，容器内端口仍为 54321。
Compose 将端口发布到宿主机网络接口，服务器部署后可由远程客户端连接；请按部署环境配置防火墙访问范围，并修改示例账号密码。

### Java（Spring Boot + MyBatis）

**推荐（PG 兼容模式）**：本部署 `DB_MODE=pg`，直接用 PostgreSQL JDBC 驱动——MyBatis-Plus 对
`DbType.POSTGRE_SQL` 的分页、批量插入、主键回填支持最成熟，社区资料多；与 gormx 复用 postgres
驱动是同一思路：

```xml
<!-- pom.xml -->
<dependency>
    <groupId>org.postgresql</groupId>
    <artifactId>postgresql</artifactId>
</dependency>
```

```yaml
# application.yml（MyBatis / MyBatis-Plus 同样适用，底层共用 spring.datasource）
spring:
  datasource:
    driver-class-name: org.postgresql.Driver
    url: jdbc:postgresql://127.0.0.1:54321/kingbase?reWriteBatchedInserts=true&tcpKeepAlive=true
    username: system
    password: 12345678ab
```

```java
// MyBatis-Plus 分页
MybatisPlusInterceptor interceptor = new MybatisPlusInterceptor();
interceptor.addInnerInterceptor(new PaginationInnerInterceptor(DbType.POSTGRE_SQL));
```

URL 参数说明（[官方 JDBC 连接属性](https://docs.kingbase.com.cn/cn/KES-V9R1C10/application/client_interface/Java/Jdbc/jdbc-2)）：

| 参数 | 说明 |
| --- | --- |
| `reWriteBatchedInserts=true` | 批量插入重写优化（金仓驱动对应 `rewriteBatchedStatements`） |
| `tcpKeepAlive=true` | TCP 保活探测，连接池长连接场景防半开连接 |
| `stringtype=unspecified` | 可选，MP 存 jsonb/枚举字段时的常见坑参数 |
| `currentSchema=xxx` | 可选，指定模式搜索路径（多 schema 时用） |
| `ApplicationName=xxx` | 可选，标识应用，便于服务端排查连接来源 |
| `connectTimeout=5` / `socketTimeout=60` | 可选，连接/读写超时（秒） |

注意：用户名密码走 `username`/`password` 独立配置，不拼进 URL（避免密码出现在日志里）；XML 中写 URL 时 `&` 需转义为 `&amp;`。

**备选（官方 kingbase8 驱动）**：[官方 MyBatis 文档](https://docs.kingbase.com.cn/cn/KES-V9R1C10/quick_start/access_tool/java/Mybatis)
标准方式，`oracle/mysql` 兼容模式或需金仓特性（国密 SSL 等）时必用；PG 模式下非首选（MP 生态
兼容性资料较少，示例工程：`https://kingbase.oss-cn-beijing.aliyuncs.com/KES_INTERFACE/quickstart/mybatis-kingbase.zip`）：

```xml
<dependency>
    <groupId>cn.com.kingbase</groupId>
    <artifactId>kingbase8</artifactId>
    <version>9.0.0</version>
</dependency>
```

```yaml
spring:
  datasource:
    driver-class-name: com.kingbase8.Driver
    url: jdbc:kingbase8://127.0.0.1:54321/kingbase?useServerPrepStmts=true&rewriteBatchedStatements=true&tcpKeepAlive=true
    username: system
    password: 12345678ab
# MyBatis-Plus 分页: DbType.KINGBASE_ES
```

### Go（gormx）

zero-service 的 `common/gormx` 已内置金仓支持，`kingbase://` 前缀 DSN 自动识别并复用 postgres 驱动：

```go
conf := gormx.Config{
    DataSource: "kingbase://system:12345678ab@127.0.0.1:54321/kingbase?sslmode=disable",
}
db, err := gormx.Open(conf) // PG 兼容模式，外部认证 scram-sha-256
```

## 初始化参数（重要）

`DB_USER` / `DB_PASSWORD` / `DB_MODE` / `ENCODING` / `ENABLE_CI` **只在首次初始化（data 为空）时生效**。

**`DB_MODE` 初始化后不可修改**：兼容模式（pg/oracle/mysql/sqlserver）是 initdb 级别的参数，记录在
`data/initdb.conf`，整个数据库集群生效。初始化后修改 compose 里的值无效（entrypoint 检测到数据目录
非空会跳过 initdb）。换兼容模式需 `stop` 后手动清空 `data` 再重新部署——**数据会全部丢失**，
需保留数据请先导出再导入。

换密码不受此限制：初始化后用 `alter user system with password '...'` 修改即可。

`ENABLE_CI`（大小写不敏感）在 pg/mysql 模式下不生效，pg 模式天然大小写敏感（贴近 PG 行为）。

## 时区

镜像 initdb 跟随容器系统时钟，默认 `timezone = 'UTC'`（`now()` 与北京时间差 8 小时，经典坑）。
`deploy.sh` 部署完成时会自动把服务器级默认时区修正为 `Asia/Shanghai`（写入 `kingbase.conf` 并
reload，对新会话生效）：

```sql
show timezone;      -- Asia/Shanghai
select now();       -- 2026-09-23 14:41:05.90246+08
```

说明：
- `timestamptz` 底层按 UTC 存储，显示随会话时区；服务器默认东八区后，Java/Go 客户端无需额外配置
- 连接远端仍为 UTC 的金仓实例时，可会话级覆盖：gormx DSN 加 `TimeZone=Asia/Shanghai`，
  Hikari 用 `connection-init-sql: set time zone 'Asia/Shanghai'`
- 容器系统时钟本身仍是 UTC，仅影响容器内 `date` 显示，不影响数据库时区

## 已知问题（V009R001C010B0004 镜像，官方文档未提及）

- **镜像内无 crond**：官方设计的 `/etc/cron.d` 每分钟自愈任务不会被执行；数据库进程崩溃但容器存活时
  无自动拉起，用 `deploy.sh restart` 恢复。守护进程/主机重启场景由 `restart: unless-stopped` 覆盖。
- **容器重建暂态失败**：`down`/`stop` 后重建，entrypoint 一次性启动可能因数据目录属主/残留 pid 失败，
  `deploy.sh` 内置每 10s 幂等补 `sys_ctl start` 兜底（实测有效）。
- **授权 90 天**：镜像自带授权有限期，`deploy.sh` 部署时实时查询剩余天数；到期替换
  `data/etc/license.dat` 后 `docker exec kingbase /home/kingbase/install/kingbase/bin/sys_ctl reload -D /home/kingbase/userdata/data`。
- **运行中删 data 目录**会导致挂载进入异常状态（Operation not permitted），恢复需 `stop` 后手动清空 `data` 再部署。

## 参考

- 官方 Docker 安装文档: https://docs.kingbase.com.cn/cn/KES-V9R1C10/install/02-docker-install
- gormx 金仓适配: `common/gormx/`（`kingbase://` 前缀 DSN 自动识别，复用 postgres 驱动）

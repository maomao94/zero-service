# KingbaseES V9R1C10 Docker 部署

金仓数据库本机 Docker 单机部署（PG 兼容模式），用于 gormx 金仓适配联调。

## 文件说明

| 文件 | 作用 |
| --- | --- |
| `deploy.sh` | 部署与运维脚本（单脚本子命令模式） |
| `docker-compose.yaml` | 容器编排配置（含逐行注释） |
| `init.sql` | 业务建库脚本（初始化后按需执行） |
| `KingbaseES_*_Docker.tar` | 官网下载的镜像 tar（gitignore，不入库） |
| `data/` | 数据库持久化目录（gitignore，不入库） |

## 快速开始

```bash
# 1. 从官网下载镜像 tar 放到本目录
#    https://www.kingbase.com.cn/download.html

# 2. 部署
bash deploy/kingbase/deploy.sh

# 3. 建业务库（可选，也可用数据库管理工具在 kingbase 库上执行）
docker exec -i kingbase ksql -Usystem -d kingbase -p 54321 < deploy/kingbase/init.sql
```

## deploy.sh 命令

```bash
bash deploy/kingbase/deploy.sh            # 部署（默认，幂等可重跑）
bash deploy/kingbase/deploy.sh stop       # 移除容器，保留数据
bash deploy/kingbase/deploy.sh restart    # 重启容器（数据库进程异常时恢复）
bash deploy/kingbase/deploy.sh status     # 容器状态 + 授权剩余天数
bash deploy/kingbase/deploy.sh logs       # 最近 100 行容器日志
```

**重新初始化**（危险操作，脚本不提供一键清空命令）：`stop` 后手动 `rm -rf data`，再执行 `deploy.sh`——检测到空目录会自动重新 initdb。

## 连接信息

| 项 | 值 |
| --- | --- |
| Host / Port | `127.0.0.1:54321` |
| 用户 / 密码 | `system` / `12345678ab` |
| 默认库 | `kingbase`（金仓默认库，相当于 PG 的 `postgres`） |
| 认证 | scram-sha-256（PG 兼容） |
| gormx DSN | `kingbase://system:12345678ab@127.0.0.1:54321/kingbase?sslmode=disable` |

数据库管理工具（TablePlus / DBeaver / Navicat 等）可直接以 PostgreSQL 协议连接，端口 54321。

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

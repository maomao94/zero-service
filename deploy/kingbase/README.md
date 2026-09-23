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

官方标准接入方式（[官方 MyBatis 文档](https://docs.kingbase.com.cn/cn/KES-V9R1C10/quick_start/access_tool/java/Mybatis)），用金仓官方 JDBC 驱动 kingbase8，Maven 中央仓库可直接拉取：

```xml
<!-- pom.xml -->
<dependency>
    <groupId>cn.com.kingbase</groupId>
    <artifactId>kingbase8</artifactId>
    <version>9.0.0</version>
</dependency>
```

```yaml
# application.yml（MyBatis / MyBatis-Plus 同样适用，底层共用 spring.datasource）
spring:
  datasource:
    driver-class-name: com.kingbase8.Driver
    url: jdbc:kingbase8://127.0.0.1:54321/kingbase
    username: system
    password: 12345678ab
```

```java
// MyBatis-Plus 分页（内置金仓方言，无需按 PG 配置）
MybatisPlusInterceptor interceptor = new MybatisPlusInterceptor();
interceptor.addInnerInterceptor(new PaginationInnerInterceptor(DbType.KINGBASE_ES));
```

原生 MyBatis + PageHelper 的官方示例工程可下载参考：
`https://kingbase.oss-cn-beijing.aliyuncs.com/KES_INTERFACE/quickstart/mybatis-kingbase.zip`

**备选（PG 兼容模式）**：本部署 DB_MODE=pg，也可直接用 postgresql 驱动——
`org.postgresql.Driver` + `jdbc:postgresql://127.0.0.1:54321/kingbase`，分页用 `DbType.POSTGRE_SQL`。
零依赖改造时可选，官方推荐仍为 kingbase8。

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

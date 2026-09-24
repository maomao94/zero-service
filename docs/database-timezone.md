# 数据库时区配置说明

覆盖 `deploy/` 下四种本地数据库部署的时区设置：KingbaseES、openGauss、PostgreSQL、MySQL。
四实例均已实测为东八区，`now()` 返回北京时间。

## 一、设置方式总览

| 数据库 | 设置方式 | 持久化位置 | 生效时机 |
| --- | --- | --- | --- |
| KingbaseES | `deploy.sh` 部署完成时幂等检查 `show timezone`，非东八区则 `sed` 改写 `kingbase.conf` 的 `timezone`/`log_timezone` 后 `sys_ctl reload` | `data/` 卷内 `kingbase.conf` | reload 后对**新会话**生效 |
| openGauss | 不支持 `alter system set timezone`，用官方 `gs_guc set -c "timezone = 'Asia/Shanghai'"` 写入配置后重启容器；已是东八区则跳过（避免无谓重启） | `data/` 卷内 `postgresql.conf` | 容器重启后生效 |
| PostgreSQL | 双保险：compose 启动参数 `postgres -c timezone=Asia/Shanghai` + `TZ`/`PGTZ` 环境变量；`deploy.sh` 再执行 `alter system` 与 `pg_reload_conf()` | 启动参数 + `data/` 卷内 `postgresql.auto.conf` | reload 后对新会话生效 |
| MySQL | 启动参数 `--default-time-zone=+08:00`，随容器启动即生效，无需部署后处理 | compose `command` 启动参数 | 容器启动即生效 |

两个细节差异：

- PG 系用命名时区 `Asia/Shanghai`；MySQL 用偏移量 `+08:00`——命名时区需先执行
  `mysql_tzinfo_to_sql` 装载时区表，偏移量可直接使用
- MySQL compose 中的 `TZ=Asia/Shanghai` 环境变量只影响容器日志时间戳，不影响服务端时区

## 二、全局性与会话行为

时区均为**服务器级全局默认**，对该实例下所有数据库生效（不是单库设置）：

1. **新会话**：连接时继承服务器默认（东八区），Java/Go 客户端无需额外配置
2. **已存在会话**：修改全局默认只对新会话生效，老连接不受影响；改完时区建议重连
   （部署脚本仅在首次部署时设置一次，日常无此场景）
3. **客户端可覆盖**：连接参数优先级高于服务器默认——JDBC URL 的 `serverTimezone`、
   Go DSN 的 `TimeZone`（PG 系）/ `loc`（MySQL）显式设置时以客户端为准

## 三、重启持久性

四种设置在数据库重启、容器重建（`docker compose down` + `up`）后均不丢失：

| 场景 | KingbaseES | openGauss | PostgreSQL | MySQL |
| --- | --- | --- | --- | --- |
| 容器重启/重建 | ✓ 配置在 `data` 挂载卷 | ✓ 同左 | ✓ 卷内 auto.conf + 启动参数 | ✓ 启动参数在 compose 文件 |
| 挂载卷保留 | ✓ `./data` | ✓ `./data` | ✓ `./data` | ✓ `./data` |
| 仅清空 data 重建 | initdb 后由 deploy.sh 重新修正 | 同左（幂等检查） | 启动参数兜底 + alter system | 不受影响（参数在 compose） |

## 四、数据存储语义

时区设置只影响**显示与写入时的换算**，不会因会话时区不同写坏数据：

- PG 系 `timestamptz`：底层按 UTC 存储，显示随会话时区；即使某会话为 UTC，数据本身一致
- MySQL `TIMESTAMP`：底层按 UTC 存储，读写按会话 `time_zone` 换算
- MySQL `DATETIME`：不做时区换算，存字面值（类型语义，与全局设置无关，选用时留意）

## 五、会话级覆盖速查

连接到时区异常（如远端 UTC 实例）时，可按客户端会话级覆盖：

```go
// Go gormx / libpq 系 DSN
dsn := "... TimeZone=Asia/Shanghai"
// MySQL driver DSN
dsn := "... loc=Asia%2FShanghai"
```

```yaml
# Java Hikari 连接池
spring:
  datasource:
    hikari:
      connection-init-sql: set time zone 'Asia/Shanghai'  # PG 系
```

## 参考

- 各库部署与验证细节见对应目录 README：`deploy/kingbase/`、`deploy/opengauss/`、`deploy/postgres/`、`deploy/mysql/`
- KingbaseES 时区章节：`deploy/kingbase/README.md`（含 `timestamptz` 存储与会话覆盖说明）

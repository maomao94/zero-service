# Kingbase KDTS-WEB 独立部署

KDTS 是 KingbaseES 配套的数据迁移与转换工具，不属于数据库容器内部服务。本目录独立编排 KDTS-WEB，避免与 Kingbase 数据库共用 Compose 项目和网络生命周期。

## 文件说明

| 文件 | 作用 |
| --- | --- |
| `deploy.sh` | KDTS 部署与运维脚本 |
| `docker-compose.yaml` | KDTS-WEB 容器编排配置 |
| `data/` | KDTS H2 任务数据持久化目录（gitignore） |

## 快速开始

```bash
# 部署或重复启动
bash deploy/kdts/deploy.sh

# 查看状态和日志
bash deploy/kdts/deploy.sh status
bash deploy/kdts/deploy.sh logs

# 停止并移除容器，保留任务数据
bash deploy/kdts/deploy.sh stop
```

访问地址：`http://127.0.0.1:54523`，HTTPS 地址为 `https://127.0.0.1:54524`。
默认账号通常为 `kingbase / kingbase`，部分 V9R1C10 安装包为 `kingbase / Kb_DI@2019`，以镜像或安装包内说明为准。

## 数据库连接

数据库源库和目标库均由 KDTS Web 页面手动配置。本部署文件不预设数据库主机、端口、网络模式或连接参数，
请根据实际部署环境填写并测试连接。

## 端口调整

KDTS 默认使用 HTTP `54523`、HTTPS `54524`。端口冲突时只修改宿主机端口：

```bash
KDTS_HTTP_HOST_PORT=54533 KDTS_HTTPS_HOST_PORT=54534 bash deploy/kdts/deploy.sh
```

## 数据迁移

KDTS 数据目录从原来的 `deploy/kingbase/kdts_data` 移动到本目录的 `data/`，原 H2 任务数据保留。
数据库部署脚本不会自动启动或停止 KDTS；迁移工具生命周期由本目录的脚本单独管理。

生产环境建议将 KDTS 部署在独立迁移机或跳板机，避免占用数据库宿主机资源。

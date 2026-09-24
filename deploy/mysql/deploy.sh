#!/bin/bash
# MySQL 8.4 Docker 部署（Docker Compose）
# 用法: bash deploy/mysql/deploy.sh [命令]
#   (无参数)  部署: 拉取镜像 -> compose up -> 等待就绪
#   stop      移除容器，保留 ./data
#   restart   重启容器
#   status    查看状态
#   logs      查看日志
#   mysql     进入 mysql 交互

set -e
cd "$(dirname "$0")"

CONTAINER=mysql8
HOST_PORT="${MYSQL_HOST_PORT:-3306}"
DB_USER=root
DB_PASSWORD=root

wait_ready() {
  echo "等待数据库就绪..."
  # 首次初始化阶段 mysqld 仅监听 socket，TCP 探测通过即表示最终启动完成
  for i in $(seq 1 60); do
    if docker exec -e MYSQL_PWD="$DB_PASSWORD" "$CONTAINER" mysql -h127.0.0.1 -P3306 -u"$DB_USER" -e "select 1" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  echo "警告: 超时未就绪；可执行 bash $0 logs 查看日志"
  return 1
}

case "${1:-deploy}" in
  deploy)
    docker info >/dev/null 2>&1 || { echo "错误: Docker 未运行"; exit 1; }
    mkdir -p data
    chmod 755 data
    if docker ps -a --format '{{.Names}}' | grep -qx "$CONTAINER"; then
      PROJECT=$(docker inspect -f '{{index .Config.Labels "com.docker.compose.project"}}' "$CONTAINER" 2>/dev/null || true)
      [ "$PROJECT" = "mysql" ] || { echo "错误: 容器 $CONTAINER 已存在且非本 compose 管理"; exit 1; }
    fi
    echo "拉取镜像 mysql:8.4 ..."
    docker pull mysql:8.4 >/dev/null 2>&1 || true
    echo "启动容器..."
    docker compose up -d
    wait_ready || exit 1
    echo ""
    docker compose ps
    echo ""
    echo "MySQL 部署完成:"
    echo "  端口:   ${HOST_PORT}（容器内 3306）"
    echo "  用户:   $DB_USER / $DB_PASSWORD"
    echo "  时区:   东八区（+08:00），字符集 utf8mb4"
    echo "  数据卷: $(pwd)/data"
    echo "  客户端: mysql -h127.0.0.1 -P $HOST_PORT -u$DB_USER -p"
    echo "  账号:   业务三账号见 roles.sql（app_user/admin_user/query_user）"
    ;;
  stop|down)
    docker compose down
    echo "容器已移除，数据保留在 $(pwd)/data"
    ;;
  restart)
    docker restart "$CONTAINER"
    wait_ready
    echo "已恢复: $(docker ps --filter name=$CONTAINER --format '{{.Status}}')"
    ;;
  status)
    docker compose ps
    ;;
  logs)
    docker compose logs --tail 100
    ;;
  mysql)
    docker exec -e MYSQL_PWD="$DB_PASSWORD" -it "$CONTAINER" mysql -u"$DB_USER"
    ;;
  *)
    echo "未知命令: $1；可用: deploy(默认)/stop/restart/status/logs/mysql"
    exit 1
    ;;
esac

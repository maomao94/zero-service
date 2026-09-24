#!/bin/bash
# PostgreSQL 16 Docker 部署（Docker Compose）
# 用法: bash deploy/postgres/deploy.sh [命令]
#   (无参数)  部署: 拉取镜像 -> compose up -> 等待就绪
#   stop      移除容器，保留 ./data
#   restart   重启容器
#   status    查看状态
#   logs      查看日志
#   psql      进入 psql 交互

set -e
cd "$(dirname "$0")"

CONTAINER=pgsql
HOST_PORT="${POSTGRES_HOST_PORT:-5432}"
export POSTGRES_HOST_PORT="$HOST_PORT"
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=postgres
IMAGE=postgres:16-alpine

ensure_image() {
  echo "从 Docker 仓库拉取镜像 $IMAGE ..."
  if docker pull "$IMAGE"; then
    return 0
  fi
  if docker image inspect "$IMAGE" >/dev/null 2>&1; then
    echo "警告: 无法从仓库拉取 $IMAGE，使用本地已有镜像"
    return 0
  fi
  echo "错误: 无法拉取 $IMAGE，且本地不存在可用镜像"
  return 1
}

wait_ready() {
  echo "等待数据库就绪..."
  for ((i = 1; i <= 60; i++)); do
    if docker exec "$CONTAINER" pg_isready -U "$DB_USER" -h 127.0.0.1 -p 5432 >/dev/null 2>&1; then
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
    ensure_image
    mkdir -p data
    chmod 755 data
    if docker ps -a --format '{{.Names}}' | grep -qx "$CONTAINER"; then
      PROJECT=$(docker inspect -f '{{index .Config.Labels "com.docker.compose.project"}}' "$CONTAINER" 2>/dev/null || true)
      [ "$PROJECT" = "postgres" ] || { echo "错误: 容器 $CONTAINER 已存在且非本 compose 管理"; exit 1; }
      if [ -z "$(ls -A data 2>/dev/null)" ]; then
        echo "data 目录为空，移除旧容器以重新初始化数据库..."
        docker compose rm -sf postgres
      else
        echo "容器已存在（compose 管理），复用..."
      fi
    fi
    echo "启动容器..."
    docker compose up -d
    wait_ready || exit 1
    # 时区: compose command 参数已强制 timezone=Asia/Shanghai，此处检查兜底
    # （无参数启动的旧部署/异常场景），已是东八区则跳过，与 kingbase/opengauss 口径一致
    if [ "$(docker exec "$CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -tA -c 'show timezone' 2>/dev/null | tr -d '[:space:]')" != "Asia/Shanghai" ]; then
      docker exec "$CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -c "alter system set timezone = 'Asia/Shanghai';" >/dev/null
      docker exec "$CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -c "select pg_reload_conf();" >/dev/null
      echo "时区已修正为 Asia/Shanghai（东八区）"
    fi
    echo ""
    docker compose ps
    echo ""
    echo "PostgreSQL 部署完成:"
    echo "  端口:   ${HOST_PORT}（容器内 5432）"
    echo "  用户:   $DB_USER / $DB_PASSWORD"
    echo "  数据库: $DB_NAME"
    echo "  数据卷: $(pwd)/data"
    echo "  psql:   psql \"host=127.0.0.1 port=$HOST_PORT dbname=postgres user=postgres password=postgres\""
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
  psql)
    docker exec -it "$CONTAINER" psql -U "$DB_USER" -d "$DB_NAME"
    ;;
  *)
    echo "未知命令: $1；可用: deploy(默认)/stop/restart/status/logs/psql"
    exit 1
    ;;
esac

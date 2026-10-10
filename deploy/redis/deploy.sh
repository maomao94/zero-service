#!/bin/bash
# Redis 7 Docker 部署（Docker Compose）
# 用法: bash deploy/redis/deploy.sh [命令]
#   (无参数)  部署: 拉取镜像 -> compose up -> 等待就绪
#   stop      移除容器，保留 ./data
#   restart   重启容器
#   status    查看状态
#   logs      查看日志
#   cli       进入 redis-cli 交互

set -e
cd "$(dirname "$0")"

CONTAINER=redis
HOST_PORT="${REDIS_HOST_PORT:-36379}"
export REDIS_HOST_PORT="$HOST_PORT"
export REDIS_PASSWORD="${REDIS_PASSWORD:-G62m50oigInC30sf}"
IMAGE=redis:7-alpine

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
  echo "等待 Redis 就绪..."
  for ((i = 1; i <= 60; i++)); do
    if docker exec "$CONTAINER" redis-cli -a "$REDIS_PASSWORD" --no-auth-warning ping 2>/dev/null | grep -q PONG; then
      return 0
    fi
    sleep 1
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
      [ "$PROJECT" = "redis" ] || { echo "错误: 容器 $CONTAINER 已存在且非本 compose 管理（请先手动移除）"; exit 1; }
      echo "容器已存在（compose 管理），复用..."
    fi
    echo "启动容器..."
    docker compose up -d
    wait_ready || exit 1
    echo ""
    docker compose ps
    echo ""
    echo "Redis 部署完成:"
    echo "  端口:   ${HOST_PORT}（容器内 6379）"
    echo "  密码:   $REDIS_PASSWORD"
    echo "  数据卷: $(pwd)/data"
    echo "  cli:    redis-cli -h 127.0.0.1 -p $HOST_PORT -a '$REDIS_PASSWORD'"
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
  cli)
    docker exec -it "$CONTAINER" redis-cli -a "$REDIS_PASSWORD" --no-auth-warning
    ;;
  *)
    echo "未知命令: $1；可用: deploy(默认)/stop/restart/status/logs/cli"
    exit 1
    ;;
esac

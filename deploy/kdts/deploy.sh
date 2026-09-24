#!/bin/bash
# Kingbase KDTS-WEB 独立部署（Docker Compose）
# 用法: bash deploy/kdts/deploy.sh [命令]
#   (无参数)  部署: 拉取镜像 -> compose up
#   stop      移除容器，保留 ./data
#   restart   重启容器
#   status    查看状态
#   logs      查看日志（最近 100 行）

set -e
cd "$(dirname "$0")"

IMAGE=huzhihui/kingbase-kdts-web:v9r1
CONTAINER=kdts
HTTP_PORT="${KDTS_HTTP_HOST_PORT:-54523}"
HTTPS_PORT="${KDTS_HTTPS_HOST_PORT:-54524}"
export KDTS_HTTP_HOST_PORT="$HTTP_PORT"
export KDTS_HTTPS_HOST_PORT="$HTTPS_PORT"

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
  echo "等待 KDTS 容器就绪..."
  for ((i = 1; i <= 30; i++)); do
    if [ "$(docker inspect -f '{{.State.Running}}' "$CONTAINER" 2>/dev/null || true)" = "true" ]; then
      PUBLISHED_HTTP_PORT=$(docker inspect -f '{{(index (index .NetworkSettings.Ports "54523/tcp") 0).HostPort}}' "$CONTAINER" 2>/dev/null || true)
      PUBLISHED_HTTP_PORT="${PUBLISHED_HTTP_PORT:-$HTTP_PORT}"
      if curl -fsS -o /dev/null "http://127.0.0.1:${PUBLISHED_HTTP_PORT}/"; then
        return 0
      fi
    fi
    sleep 2
  done
  echo "警告: 容器未进入运行状态；可执行 bash $0 logs 查看日志"
  return 1
}

cmd_deploy() {
  docker info >/dev/null 2>&1 || { echo "错误: Docker 未运行"; exit 1; }
  ensure_image
  mkdir -p data
  chmod 755 data

  if docker ps -a --format '{{.Names}}' | grep -qx "$CONTAINER"; then
    PROJECT=$(docker inspect -f '{{index .Config.Labels "com.docker.compose.project"}}' "$CONTAINER" 2>/dev/null || true)
    [ "$PROJECT" = "kdts" ] || {
      echo "错误: 容器 $CONTAINER 已存在且非独立 KDTS Compose 管理，请先迁移或移除旧容器"
      exit 1
    }
  fi

  echo "启动 KDTS..."
  docker compose up -d
  wait_ready || exit 1

  echo ""
  docker compose ps
  echo ""
  echo "KDTS-WEB 部署完成:"
  echo "  HTTP:   http://127.0.0.1:${HTTP_PORT}"
  echo "  HTTPS:  https://127.0.0.1:${HTTPS_PORT}"
  echo "  数据卷: $(pwd)/data"
  echo "  数据库: 源库和目标库连接由 KDTS Web 页面配置"
}

case "${1:-deploy}" in
  deploy)
    cmd_deploy
    ;;
  stop|down)
    docker compose down
    echo "KDTS 容器已移除，数据保留在 $(pwd)/data"
    ;;
  restart)
    docker restart "$CONTAINER"
    wait_ready
    echo "已恢复: $(docker ps --filter name="$CONTAINER" --format '{{.Status}}')"
    ;;
  status)
    docker compose ps
    ;;
  logs)
    docker compose logs --tail 100
    ;;
  *)
    echo "未知命令: $1；可用: deploy(默认)/stop/restart/status/logs"
    exit 1
    ;;
esac

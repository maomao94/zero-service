#!/bin/bash
# dbswitch-admin 独立部署（Docker Compose）
# 用法: bash deploy/dbswitch/deploy.sh [命令]
#   (无参数)  部署: 拉取镜像 -> compose up
#   stop      移除容器，保留 ./data
#   restart   重启容器
#   status    查看状态
#   logs      查看日志（最近 100 行）

set -e
cd "$(dirname "$0")"

IMAGE=registry.cn-hangzhou.aliyuncs.com/inrgihc/dbswitch:latest
CONTAINER=dbswitch
HTTP_PORT="${DBSWITCH_HTTP_HOST_PORT:-9088}"
export DBSWITCH_HTTP_HOST_PORT="$HTTP_PORT"

ensure_image() {
  echo "从 Docker 仓库拉取镜像 $IMAGE ..."
  if docker image inspect "$IMAGE" >/dev/null 2>&1; then
    echo "本地已存在镜像 $IMAGE，跳过拉取"
    return 0
  fi
  if docker pull "$IMAGE"; then
    return 0
  fi
  echo "错误: 无法拉取 $IMAGE"
  return 1
}

wait_ready() {
  echo "等待 dbswitch 容器就绪..."
  for ((i = 1; i <= 30; i++)); do
    if [ "$(docker inspect -f '{{.State.Running}}' "$CONTAINER" 2>/dev/null || true)" = "true" ]; then
      PUBLISHED_HTTP_PORT=$(docker inspect -f '{{(index (index .NetworkSettings.Ports "9088/tcp") 0).HostPort}}' "$CONTAINER" 2>/dev/null || true)
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

  if docker ps -a --format '{{.Names}}' | grep -qx "$CONTAINER"; then
    PROJECT=$(docker inspect -f '{{index .Config.Labels "com.docker.compose.project"}}' "$CONTAINER" 2>/dev/null || true)
    [ "$PROJECT" = "dbswitch" ] || {
      echo "错误: 容器 $CONTAINER 已存在且非本 Compose 项目管理，请先迁移或移除旧容器"
      exit 1
    }
  fi

  echo "启动 dbswitch..."
  docker compose up -d
  wait_ready || exit 1

  echo ""
  docker compose ps
  echo ""
  echo "dbswitch-admin 部署完成:"
  echo "  Web:    http://127.0.0.1:${HTTP_PORT}"
  echo "  账号:   admin / 123456"
  echo "  数据卷: $(pwd)/data（H2 配置库，容器内 /tmp）"
  echo "  数据库: 源库和目标库连接由 Web 页面配置，主机地址填 host.docker.internal"
  echo ""
  echo "Web 端建连接可用参数（连接测试通过后配置任务）:"
  echo "  金仓(pg模式): jdbc:kingbase8://host.docker.internal:54321/zero        system/12345678ab"
  echo "  金仓(PG驱动): jdbc:postgresql://host.docker.internal:54321/zero       system/12345678ab"
  echo "  openGauss:    jdbc:opengauss://host.docker.internal:15432/postgres    gaussdb/Gauss@123"
  echo "  PostgreSQL:   jdbc:postgresql://host.docker.internal:5432/postgres    postgres/postgres"
  echo "  MySQL:        jdbc:mysql://host.docker.internal:3306/<db>              root/root"
}

case "${1:-deploy}" in
  deploy)
    cmd_deploy
    ;;
  stop|down)
    docker compose down
    echo "dbswitch 容器已移除，数据保留在 $(pwd)/data"
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

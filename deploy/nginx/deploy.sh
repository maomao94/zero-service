#!/bin/bash
# Zero Service Web 网关部署（Docker Compose）
# 用法: bash deploy/nginx/deploy.sh [命令]
#   (无参数)  部署: 拉取镜像 -> 校验证书 -> compose up -> 等待就绪
#   build     构建前端产物到 deploy/nginx/dist（live/socketio/workspace）
#   stop      移除容器
#   restart   重启容器
#   status    查看状态
#   logs      查看日志
#   url       打印访问地址

set -e
cd "$(dirname "$0")"

CONTAINER=web-gateway
HOST_PORT="${NGINX_HOST_PORT:-8088}"
SSL_HOST_PORT="${NGINX_SSL_HOST_PORT:-8443}"
SOCKETIO_HOST_PORT="${SOCKETIO_HOST_PORT:-8090}"
WORKSPACE_HOST_PORT="${WORKSPACE_HOST_PORT:-8091}"
export NGINX_HOST_PORT="$HOST_PORT"
export NGINX_SSL_HOST_PORT="$SSL_HOST_PORT"
export SOCKETIO_HOST_PORT="$SOCKETIO_HOST_PORT"
export WORKSPACE_HOST_PORT="$WORKSPACE_HOST_PORT"
IMAGE=nginx:stable-alpine
CERT="${NGINX_TLS_CERT:-../tls/server.crt}"
KEY="${NGINX_TLS_KEY:-../tls/server.key}"
export NGINX_TLS_CERT="$CERT"
export NGINX_TLS_KEY="$KEY"

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

# Nginx 不生成自签证书：证书不存在直接报错，要求先用 deploy/tls 生成或替换为正式证书
require_cert() {
  if [ ! -f "$CERT" ] || [ ! -f "$KEY" ]; then
    echo "错误: 未找到 TLS 证书（$CERT / $KEY）"
    echo "请先执行: bash deploy/tls/gen-tls.sh  生成证书；或提供正式证书后设置"
    echo "  NGINX_TLS_CERT=/path/server.crt NGINX_TLS_KEY=/path/server.key bash deploy/nginx/deploy.sh"
    exit 1
  fi
}

wait_ready() {
  echo "等待网关就绪..."
  for ((i = 1; i <= 30; i++)); do
    if curl -fsS -m 2 "http://127.0.0.1:${HOST_PORT}/healthz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  echo "警告: 超时未就绪；可执行 bash $0 logs 查看日志"
  return 1
}

case "${1:-deploy}" in
  build)
    bash ./build-web.sh
    ;;
  deploy)
    docker info >/dev/null 2>&1 || { echo "错误: Docker 未运行"; exit 1; }
    ensure_image
    require_cert
    # 预留产物目录，避免 docker 以 root 创建；内容由 build-web.sh 或部署时自行上传
    mkdir -p dist/live dist/socketio dist/workspace ../livekit/recordings
    # 迁移：清理旧的 recordings-nginx（早期仅录制静态服务的容器）
    if docker ps -a --format '{{.Names}}' | grep -qx "recordings-nginx"; then
      OLD_PROJECT=$(docker inspect -f '{{index .Config.Labels "com.docker.compose.project"}}' recordings-nginx 2>/dev/null || true)
      if [ "$OLD_PROJECT" = "nginx" ]; then
        echo "清理旧容器 recordings-nginx ..."
        docker rm -f recordings-nginx >/dev/null 2>&1 || true
      fi
    fi
    echo "启动容器..."
    docker compose up -d
    wait_ready || exit 1
    echo ""
    docker compose ps
    echo ""
    echo "Web 网关部署完成:"
    echo "  Live 前端(HTTP):  http://127.0.0.1:${HOST_PORT}/        （推荐）"
    echo "  Live 前端(HTTPS): https://127.0.0.1:${SSL_HOST_PORT}/"
    echo "  SocketIO 工具:    http://127.0.0.1:${SOCKETIO_HOST_PORT}/"
    echo "  Zero 工作台:      http://127.0.0.1:${WORKSPACE_HOST_PORT}/"
    echo "  录制文件:         http://127.0.0.1:${HOST_PORT}/recordings/"
    echo "  前端产物目录:     $(cd dist && pwd)"
    echo "  反代:             /live/ -> livegtw:11002, /socket.io/ -> socketgtw:11003, /livekit/ -> livekit:7880"
    ;;
  stop|down)
    docker compose down
    echo "容器已移除"
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
  url)
    echo "HTTP : http://127.0.0.1:${HOST_PORT}/"
    echo "HTTPS: https://127.0.0.1:${SSL_HOST_PORT}/"
    echo "录制: http://127.0.0.1:${HOST_PORT}/recordings/"
    ;;
  *)
    echo "未知命令: $1；可用: deploy(默认)/build/stop/restart/status/logs/url"
    exit 1
    ;;
esac

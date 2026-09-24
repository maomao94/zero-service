#!/bin/bash
# openGauss 7.0 单机部署（Docker Compose，持久化版）
# 用法: bash deploy/opengauss/deploy.sh [命令]
#   (无参数)  部署: 拉取镜像 -> compose up -> 等待就绪（幂等，可重跑）
#   stop      移除容器，保留 ./data 数据
#   restart   重启容器
#   status    查看容器状态
#   logs      查看容器日志（最近 100 行）
#   psql      进入 gsql 交互（以 omm 免密进入，需容器已就绪）
#
# 重新初始化（危险操作）: stop 后手动 rm -rf data，再执行本脚本会重新 initdb
#
# 说明:
#   - 替代原无挂卷的本地高斯容器，已清理旧容器，新部署持久化到 ./data
#   - 默认端口 5432；端口冲突时设置 OPENGAUSS_HOST_PORT，只修改宿主机映射端口
#   - 密码复杂度: 需含大小写、数字、特殊字符，默认 Gauss@123

set -e
cd "$(dirname "$0")"

IMAGE=opengauss/opengauss-server:latest
CONTAINER=opengauss
HOST_PORT="${OPENGAUSS_HOST_PORT:-5432}"
export OPENGAUSS_HOST_PORT="$HOST_PORT"
DB_USER=gaussdb
DB_PASSWORD='Gauss@123'
DB_NAME=postgres

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
    # 通过 gs_ctl 探测，无需密码；gsql 验证用 omm 用户（local trust，无需密码）
    if docker exec "$CONTAINER" gosu omm bash -c 'export GAUSSHOME=/usr/local/opengauss; export PATH=$GAUSSHOME/bin:$PATH; export LD_LIBRARY_PATH=$GAUSSHOME/lib:/scws/lib:$LD_LIBRARY_PATH; gs_ctl status -D /var/lib/opengauss/data >/dev/null 2>&1'; then
      if docker exec "$CONTAINER" gosu omm bash -c 'export GAUSSHOME=/usr/local/opengauss; export PATH=$GAUSSHOME/bin:$PATH; export LD_LIBRARY_PATH=$GAUSSHOME/lib:/scws/lib:$LD_LIBRARY_PATH; gsql -d postgres -U omm -c "select 1" >/dev/null 2>&1'; then
        return 0
      fi
    fi
    sleep 2
  done
  echo "警告: 超时未就绪；可执行 bash $0 logs 查看日志"
  return 1
}

cmd_deploy() {
  docker info >/dev/null 2>&1 || { echo "错误: Docker 未运行"; exit 1; }
  ensure_image

  # 数据目录准备（持久化）
  mkdir -p data
  chmod 755 data

  # 容器状态检查
  if docker ps -a --format '{{.Names}}' | grep -qx "$CONTAINER"; then
    PROJECT=$(docker inspect -f '{{index .Config.Labels "com.docker.compose.project"}}' "$CONTAINER" 2>/dev/null || true)
    [ "$PROJECT" = "opengauss" ] || { echo "错误: 容器 $CONTAINER 已存在且非本 compose 管理，请先移除"; exit 1; }
    if [ -z "$(ls -A data 2>/dev/null)" ]; then
      echo "data 目录为空，移除旧容器以重新初始化数据库..."
      docker compose rm -sf "$CONTAINER"
    else
      echo "容器已存在（compose 管理），复用..."
    fi
  fi

  echo "启动容器..."
  docker compose up -d
  wait_ready || exit 1

  # openGauss 不支持 alter system 修改时区，通过 gs_guc 写入持久化 postgresql.conf 后重启生效
  # 已是 Asia/Shanghai 时跳过，避免重复部署时无谓重启
  CURRENT_TZ=$(docker exec "$CONTAINER" gosu omm bash -c 'export GAUSSHOME=/usr/local/opengauss; export PATH=$GAUSSHOME/bin:$PATH; export LD_LIBRARY_PATH=$GAUSSHOME/lib:/scws/lib:$LD_LIBRARY_PATH; gsql -d postgres -U omm -tA -c "show timezone"' 2>/dev/null | tr -d '[:space:]')
  if [ "$CURRENT_TZ" != "Asia/Shanghai" ]; then
    docker exec "$CONTAINER" gosu omm bash -c 'export GAUSSHOME=/usr/local/opengauss; export PATH=$GAUSSHOME/bin:$PATH; export LD_LIBRARY_PATH=$GAUSSHOME/lib:/scws/lib:$LD_LIBRARY_PATH; gs_guc set -D /var/lib/opengauss/data -c "timezone = '\''Asia/Shanghai'\''"' >/dev/null
    docker restart "$CONTAINER" >/dev/null
    wait_ready || exit 1
    echo "时区已修正为 Asia/Shanghai（东八区）"
  fi

  echo ""
  docker compose ps
  echo ""
  echo "openGauss 部署完成:"
  echo "  端口:   ${HOST_PORT}（容器内 5432）"
  echo "  用户:   $DB_USER / ${DB_PASSWORD}（omm 同密码）"
  echo "  数据库: ${DB_NAME}（默认库）"
  echo "  数据卷: $(pwd)/data"
  echo "  gsql:   gsql -d postgres -U gaussdb -W '${DB_PASSWORD}' -h 127.0.0.1 -p$HOST_PORT"
  echo "  psql:   psql \"host=127.0.0.1 port=$HOST_PORT dbname=postgres user=gaussdb password=${DB_PASSWORD}\""
}

case "${1:-deploy}" in
  deploy)
    cmd_deploy
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
  psql|gsql)
    # gaussdb 本地连接也强制密码（openGauss 对非初始用户不适用 trust），统一以 omm 免密进入
    docker exec -it "$CONTAINER" gosu omm bash -c 'export GAUSSHOME=/usr/local/opengauss; export PATH=$GAUSSHOME/bin:$PATH; export LD_LIBRARY_PATH=$GAUSSHOME/lib:/scws/lib:$LD_LIBRARY_PATH; gsql -d postgres -U omm'
    ;;
  *)
    echo "未知命令: $1；可用: deploy(默认)/stop/restart/status/logs/psql"
    exit 1
    ;;
esac

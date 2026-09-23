#!/bin/bash
# KingbaseES V9R1C10 单机部署（Docker Compose，PG 兼容模式）
# 用法: bash deploy/kingbase/deploy.sh [命令]
#   (无参数)  部署: 自动 load 镜像 -> retag kingbase:kes -> compose up -> 等待就绪（幂等，可重跑）
#   stop      移除容器，保留 ./data 数据
#   restart   重启容器（数据库进程异常时的恢复手段）
#   status    查看容器状态与授权剩余天数
#   logs      查看容器日志（最近 100 行）
#
# 重新初始化（危险操作，脚本不代做）: stop 后手动 rm -rf data，再执行本脚本检测到空目录会自动重新 initdb
#
# 说明:
#   - 环境变量仅在首次初始化（data 为空）时生效，见 docker-compose.yaml
#   - gormx 连接串: kingbase://system:12345678ab@127.0.0.1:54321/kingbase?sslmode=disable

set -e
cd "$(dirname "$0")"

IMAGE_TAG=kingbase:kes
DB_USER=system
DB_PASSWORD=12345678ab
DB_NAME=kingbase    # 管理操作连接库（金仓默认库，相当于 PG 的 postgres），业务库用 init.sql 或管理工具创建

license_days() {
  docker exec -e PGPASSWORD="$DB_PASSWORD" kingbase \
    ksql -U"$DB_USER" -d "$DB_NAME" -p 54321 -tA -c 'select GET_LICENSE_VALIDDAYS();' 2>/dev/null
}

wait_ready() {
  echo "等待数据库就绪..."
  for i in $(seq 1 60); do
    if docker exec -e PGPASSWORD="$DB_PASSWORD" kingbase \
        ksql -U"$DB_USER" -d "$DB_NAME" -p 54321 -c 'select version();' >/dev/null 2>&1; then
      return 0
    fi
    # 每 10s 幂等补一次 sys_ctl start: 容器重建瞬间的属主/残留 pid 问题会导致
    # entrypoint 一次性启动失败，镜像内无 crond 不会自动重试
    if [ $((i % 5)) -eq 0 ]; then
      docker exec kingbase /home/kingbase/install/kingbase/bin/sys_ctl \
        -D /home/kingbase/userdata/data -l /home/kingbase/userdata/data/logfile start >/dev/null 2>&1 || true
    fi
    sleep 2
  done
  echo "警告: 超时未就绪；可执行 bash $0 logs 查看日志，数据异常时 stop 后手动清空 data 目录重新部署"
  return 1
}

cmd_deploy() {
  # 1. 检查 Docker（官方要求 >= 20.10.0）
  docker info >/dev/null 2>&1 || { echo "错误: Docker 未运行"; exit 1; }
  VER=$(docker version --format '{{.Server.Version}}')
  [ "$(printf '%s\n20.10.0' "$VER" | sort -V | head -1)" = "20.10.0" ] \
    || { echo "错误: Docker 版本 $VER 低于官方要求的 20.10.0"; exit 1; }

  # 2. 镜像准备（已有 kingbase:kes 则跳过，否则从同目录 tar 加载）
  if docker image inspect "$IMAGE_TAG" >/dev/null 2>&1; then
    echo "镜像已存在: $IMAGE_TAG"
  else
    TAR=$(ls KingbaseES_*_Docker.tar 2>/dev/null | head -1 || true)
    [ -n "$TAR" ] || { echo "错误: 未找到镜像 tar，请将 KingbaseES_*_Docker.tar 放到本目录"; exit 1; }
    echo "加载镜像 $TAR ..."
    # 从 docker load 输出解析本次加载的镜像名，避免误 retag 本地已有旧版镜像
    # 若 docker load 报错，可按官方提示改用 docker import 导入
    LOADED=$(docker load -i "$TAR" | sed -n 's/^Loaded image: //p' | head -1)
    [ -n "$LOADED" ] || { echo "错误: docker load 未识别到镜像名"; exit 1; }
    docker tag "$LOADED" "$IMAGE_TAG"
    echo "镜像已 retag: $LOADED -> $IMAGE_TAG"
  fi

  # 3. 数据目录准备（官方要求宿主机挂载目录 755 权限，否则 Permission denied；目录被删时自动重建）
  mkdir -p data
  chmod 755 data

  # 4. 容器状态检查（compose 管理的容器: data 为空则先 down 重建以重新初始化，否则复用）
  if docker ps -a --format '{{.Names}}' | grep -qx "kingbase"; then
    PROJECT=$(docker inspect -f '{{index .Config.Labels "com.docker.compose.project"}}' kingbase 2>/dev/null || true)
    [ "$PROJECT" = "kingbase" ] || { echo "错误: 容器 kingbase 已存在且非本 compose 管理，请先移除"; exit 1; }
    if [ -z "$(ls -A data 2>/dev/null)" ]; then
      echo "data 目录为空，移除旧容器以重新初始化数据库..."
      docker compose down
    else
      echo "容器已存在（compose 管理），复用..."
    fi
  fi

  # 5. 启动
  echo "启动容器..."
  docker compose up -d
  wait_ready || exit 1

  echo ""
  docker compose ps
  echo ""
  echo "KingbaseES 部署完成:"
  echo "  端口:   54321（容器内 54321）"
  echo "  用户:   $DB_USER / $DB_PASSWORD"
  echo "  数据库: kingbase（金仓默认库；业务库用 init.sql 或管理工具创建）"
  echo "  数据卷: $(pwd)/data"
  echo "  gormx:  kingbase://$DB_USER:$DB_PASSWORD@127.0.0.1:54321/kingbase?sslmode=disable"
  echo "  授权:   剩余 $(license_days) 天，到期需替换 userdata/etc/license.dat"
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
    docker restart kingbase
    wait_ready
    echo "已恢复: $(docker ps --filter name=kingbase --format '{{.Status}}')"
    ;;
  status)
    docker compose ps
    echo "授权剩余: $(license_days) 天"
    ;;
  logs)
    docker compose logs --tail 100
    ;;
  *)
    echo "未知命令: $1；可用: deploy(默认)/stop/restart/status/logs"
    exit 1
    ;;
esac

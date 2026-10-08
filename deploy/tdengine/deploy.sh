#!/bin/bash
# TDengine TSDB 3.4 单机部署（Docker Compose，社区版）
# 用法: bash deploy/tdengine/deploy.sh [命令] [init 的追加 SQL 文件...]
#   (无参数)  部署: compose up -> 等待就绪（幂等，可重跑）
#   init      业务库初始化（幂等，可重跑）:
#             按文件名顺序执行 init.d/*.sql，再执行命令行追加的 SQL 文件
#             bash deploy/tdengine/deploy.sh init                            # 仅 init.d/ 下的建库脚本
#             bash deploy/tdengine/deploy.sh init model/sql/tdengine.sql     # 追加业务超表 DDL
#             bash deploy/tdengine/deploy.sh init a.sql b.sql                # 追加多个 SQL 文件
#   stop      移除容器，保留 ./data 数据
#   restart   重启容器
#   status    查看容器状态
#   logs      查看容器日志（最近 100 行）
#   taos      进入 taos CLI 交互
#
# 重新初始化（危险操作，脚本不代做）: stop 后手动 rm -rf data log，再执行本脚本
#
# 说明:
#   - 镜像 tdengine/tsdb:latest（社区版，3.3.7.0 起 tdengine/tdengine 更名为 tdengine/tsdb）
#   - 默认宿主机端口 6030/6041；端口冲突时设置 TDENGINE_HOST_PORT / TDENGINE_REST_PORT
#   - 默认账号 root/taosdata，生产环境务必通过 TDENGINE_ROOT_PASSWORD 修改
#   - REST 接口: curl -u root:taosdata -d "show databases" localhost:6041/rest/sql

set -e
# 记录执行时的工作目录：脚本内部 cd 到部署目录后，
# init 追加参数的相对路径仍需基于用户执行时的工作目录解析
ORIG_PWD="$(pwd)"
cd "$(dirname "$0")"

export TDENGINE_HOST_PORT="${TDENGINE_HOST_PORT:-6030}"
export TDENGINE_REST_PORT="${TDENGINE_REST_PORT:-6041}"
export TDENGINE_ROOT_PASSWORD="${TDENGINE_ROOT_PASSWORD:-taosdata}"
export COMPOSE_IGNORE_ORPHANS=1

wait_ready() {
  echo "等待 TDengine 就绪..."
  for ((i = 1; i <= 60; i++)); do
    if docker exec tdengine taos -s "show databases" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  echo "警告: 超时未就绪；可执行 bash $0 logs 查看日志，数据异常时 stop 后手动清空 data 目录重新部署"
  return 1
}

cmd_deploy() {
  # 1. 检查 Docker
  docker info >/dev/null 2>&1 || { echo "错误: Docker 未运行"; exit 1; }

  # 2. 数据目录准备
  mkdir -p data log

  # 3. 容器状态检查
  if docker ps -a --format '{{.Names}}' | grep -qx "tdengine"; then
    PROJECT=$(docker inspect -f '{{index .Config.Labels "com.docker.compose.project"}}' tdengine 2>/dev/null || true)
    [ "$PROJECT" = "tdengine" ] || { echo "错误: 容器 tdengine 已存在且非本 compose 管理，请先移除"; exit 1; }
    if [ -z "$(ls -A data 2>/dev/null)" ]; then
      echo "data 目录为空，移除旧容器以重新初始化..."
      docker compose rm -sf tdengine
    else
      echo "容器已存在（compose 管理），复用..."
    fi
  fi

  # 4. 启动
  echo "启动容器..."
  docker compose up -d
  wait_ready || exit 1

  echo ""
  docker compose ps tdengine
  echo ""
  echo "TDengine 部署完成:"
  echo "  端口:   ${TDENGINE_HOST_PORT}（native）/ ${TDENGINE_REST_PORT}（REST）"
  echo "  用户:   root / ${TDENGINE_ROOT_PASSWORD}"
  echo "  数据卷: $(pwd)/data"
  echo "  CLI:    docker exec -it tdengine taos"
  echo "  REST:   curl -u root:${TDENGINE_ROOT_PASSWORD} -d \"show databases\" localhost:${TDENGINE_REST_PORT}/rest/sql"
}

# 执行一个 SQL 文件（taos -s 整文件传参）
# 说明:
#   - taos 管道 stdin（taos < file）会被按交互终端逐字符解析导致
#     "Incomplete SQL statement"（实测），须用官方非交互 -s 传整文件内容
#     （-s 支持分号分隔多语句，-- 注释与多行 DDL 均正常解析，实测验证）
#   - taos -s 对失败 SQL 的退出码仍为 0（实测），须检查输出中的 DB error
run_sql_file() {
  local sql="$1" out
  out=$(docker exec tdengine taos -s "$(cat "$sql")" 2>&1)
  if echo "$out" | grep -q "DB error"; then
    echo "错误: $sql 执行失败:"
    echo "$out" | grep "DB error"
    exit 1
  fi
}

cmd_init() {
  # 容器需在运行状态
  docker ps --format '{{.Names}}' | grep -qx "tdengine" \
    || { echo "错误: 容器 tdengine 未运行，先执行 bash $0 部署"; exit 1; }

  # 1. 按文件名顺序执行 init.d/ 下的建库脚本（每个业务一个文件，幂等）
  shopt -s nullglob
  INIT_SQLS=(init.d/*.sql)
  shopt -u nullglob
  if [ ${#INIT_SQLS[@]} -gt 0 ]; then
    for sql in "${INIT_SQLS[@]}"; do
      echo "执行 $sql ..."
      run_sql_file "$sql"
    done
  else
    echo "提示: init.d/ 下无 SQL 文件，跳过"
  fi

  # 2. 执行命令行追加的 SQL 文件（业务超表 DDL 等，如 model/sql/tdengine.sql）
  shift   # 跳过命令名 init，剩余参数为追加的 SQL 文件
  if [ $# -gt 0 ]; then
    for sql in "$@"; do
      # 相对路径基于用户执行时的工作目录解析（脚本已 cd 到部署目录）
      if [ ! -f "$sql" ] && [ -f "$ORIG_PWD/$sql" ]; then
        sql="$ORIG_PWD/$sql"
      fi
      [ -f "$sql" ] || { echo "错误: 未找到 SQL 文件 $sql"; exit 1; }
      echo "执行 $sql ..."
      run_sql_file "$sql"
    done
  fi

  # 3. 验证
  echo ""
  docker exec tdengine taos -s "show databases;"
  echo ""
  echo "初始化完成（幂等可重跑）；服务配置对齐: 各服务 TaosDB.DBName 必须与对应库名一致"
  echo "IEC 104 采集示例（facade/streamevent/etc/streamevent.yaml）:"
  echo "  TaosDB:"
  echo "    DataSource: root:taosdata@http(localhost:6041)/?timezone=Asia%2FShanghai"
  echo "    DBName: iec104   # 必须与库名一致，默认值 default 不会命中本库"
}

case "${1:-deploy}" in
  deploy)
    cmd_deploy
    ;;
  init)
    cmd_init "$@"
    ;;
  stop|down)
    docker compose down
    echo "容器已移除，数据保留在 $(pwd)/data"
    ;;
  restart)
    docker restart tdengine
    wait_ready
    echo "已恢复: $(docker ps --filter name=tdengine --format '{{.Status}}')"
    ;;
  status)
    docker compose ps tdengine
    ;;
  logs)
    docker compose logs --tail 100 tdengine
    ;;
  taos)
    docker exec -it tdengine taos
    ;;
  *)
    echo "未知命令: $1；可用: deploy(默认)/init/stop/restart/status/logs/taos"
    exit 1
    ;;
esac

#!/bin/bash

# === logdump 服务单架构快速部署脚本 ===
#
# 适用场景：
#   - 日常开发迭代：代码改动后快速部署到测试/生产服务器
#   - 热修复部署：紧急 bug 修复后立即上线
#   - 单服务器环境：明确知道目标服务器架构（amd64 或 arm64）
#
# 不适用场景：
#   - 多平台镜像交付：需要同时提供 x86 和 ARM 镜像包 → 使用 build.sh
#   - 离线分发：需要预先制作镜像 tar 包给多个客户 → 使用 build.sh
#
# 用法: ./deploy.sh [环境] [平台]
# 示例:
#   ./deploy.sh test                 # 部署到测试环境（默认 amd64）
#   ./deploy.sh prod linux/arm64     # 部署到 ARM 架构生产服务器

set -e

ENVIRONMENT=${1:-dev}
PLATFORM=${2:-}
ENV_FILE="env/${ENVIRONMENT}.env"

log()    { echo -e "\033[1;34m[INFO]\033[0m $1"; }
warn()   { echo -e "\033[1;33m[WARN]\033[0m $1"; }
error()  { echo -e "\033[1;31m[ERROR]\033[0m $1"; }

# 检查 .env 文件
if [ ! -f "$ENV_FILE" ]; then
  error "环境文件不存在: $ENV_FILE"
  exit 1
fi

log "加载环境变量文件: $ENV_FILE"
export $(grep -v '^#' $ENV_FILE | xargs)

# 校验必要变量
REQUIRED_VARS=(REMOTE_USER REMOTE_PASSWD REMOTE_HOST REMOTE_PORT REMOTE_PATH REMOTE_COMPOSE_PATH IMAGE_NAME SERVICE_NAME)
for var in "${REQUIRED_VARS[@]}"; do
  if [ -z "${!var}" ]; then
    error "缺少必要环境变量: $var"
    exit 1
  fi
done

# 共享部署工具
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
source "${SCRIPT_DIR}/../../deploy/common.sh"

TARGET_ARCH=$(target_arch "$PLATFORM")

# 校验远端架构：构建前确认目标服务器架构，避免把错架构镜像推上去后容器启动失败
log "校验远端架构: ${REMOTE_USER}@${REMOTE_HOST} (${TARGET_ARCH})"
assert_remote_arch "$REMOTE_HOST" "$REMOTE_PORT" "$REMOTE_USER" "$REMOTE_PASSWD" "$TARGET_ARCH"

# 备份保留数量，默认3
BACKUP_KEEP=${BACKUP_KEEP:-3}

# 远程标签，默认 latest
REMOTE_IMAGE_TAG=${REMOTE_IMAGE_TAG:-latest}

# === 调用 build.sh 构建单架构镜像 ===
log "调用 build.sh 构建 ${TARGET_ARCH} 架构镜像..."

BUILD_TIMESTAMP=$(date +%Y%m%d%H%M%S)

# 调用 build.sh，传递时间戳和目标架构
if ! TAR_NAME=$(BUILD_TIMESTAMP="$BUILD_TIMESTAMP" BUILD_OUTPUT_ONLY=true bash "${SCRIPT_DIR}/build.sh" "$ENVIRONMENT" "linux/${TARGET_ARCH}"); then
  error "镜像构建失败"
  exit 1
fi

if [ -z "$TAR_NAME" ]; then
  error "镜像构建未返回 tar 文件名"
  exit 1
fi

log "镜像构建完成: $TAR_NAME"

# === 上传镜像到远程服务器 ===
log "上传镜像到远程服务器..."
RETRY_COUNT=0
MAX_RETRIES=3

sshpass -p "$REMOTE_PASSWD" ssh -p ${REMOTE_PORT} -o StrictHostKeyChecking=no ${REMOTE_USER}@${REMOTE_HOST} "mkdir -p ${REMOTE_PATH}"

RETRY_COUNT=0
until sshpass -p "$REMOTE_PASSWD" scp -P ${REMOTE_PORT} -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null ${TAR_NAME} ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_PATH}/
do
  RETRY_COUNT=$((RETRY_COUNT + 1))
  if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
    error "SCP 上传失败超过 ${MAX_RETRIES} 次，退出"
    exit 1
  fi
  warn "SCP 上传失败，重试中（$RETRY_COUNT/${MAX_RETRIES}）..."
  sleep 2
done

# === 远程执行部署 ===
log "连接远程服务器部署..."
sshpass -p "$REMOTE_PASSWD" ssh -p ${REMOTE_PORT} -o StrictHostKeyChecking=no ${REMOTE_USER}@${REMOTE_HOST} bash -s <<EOFREMOTE
  set -e
  cd ${REMOTE_PATH}

  echo "[远程] 加载新镜像: ${TAR_NAME}"
  docker load -i ${TAR_NAME}

  # 提取镜像标签（格式：<timestamp>-<arch>）
  TEMP_TAG="${BUILD_TIMESTAMP}-${TARGET_ARCH}"

  NEW_IMAGE_ID=\$(docker images --format '{{.Repository}} {{.Tag}} {{.ID}}' | grep "^${IMAGE_NAME} \${TEMP_TAG} " | awk '{print \$3}')
  echo "[远程] 新镜像ID: \$NEW_IMAGE_ID"

  if docker image inspect ${IMAGE_NAME}:${REMOTE_IMAGE_TAG} > /dev/null 2>&1; then
    OLD_IMAGE_ID=\$(docker images --format '{{.Repository}} {{.Tag}} {{.ID}}' | grep "^${IMAGE_NAME} ${REMOTE_IMAGE_TAG} " | awk '{print \$3}')
    echo "[远程] 旧 ${IMAGE_NAME}:${REMOTE_IMAGE_TAG} 镜像ID: \$OLD_IMAGE_ID"
  else
    OLD_IMAGE_ID=""
    echo "[远程] 旧 ${IMAGE_NAME}:${REMOTE_IMAGE_TAG} 镜像不存在"
  fi

  if [ -n "\$OLD_IMAGE_ID" ] && [ "\$OLD_IMAGE_ID" != "\$NEW_IMAGE_ID" ]; then
    BACKUP_TAG="backup_\$(date +%Y%m%d%H%M%S)"
    docker tag ${IMAGE_NAME}:${REMOTE_IMAGE_TAG} ${IMAGE_NAME}:\$BACKUP_TAG
    echo "[远程] 备份成功: ${IMAGE_NAME}:\$BACKUP_TAG"
  else
    echo "[远程] 不需要备份旧镜像"
  fi

  echo "[远程] 打标签为: ${REMOTE_IMAGE_TAG}"
  docker tag ${IMAGE_NAME}:\${TEMP_TAG} ${IMAGE_NAME}:${REMOTE_IMAGE_TAG}

  if docker image inspect ${IMAGE_NAME}:\${TEMP_TAG} > /dev/null 2>&1; then
    echo "[远程] 删除临时 tag: ${IMAGE_NAME}:\${TEMP_TAG}"
    docker rmi ${IMAGE_NAME}:\${TEMP_TAG} || true
  fi

  CURRENT_IMAGE_ID=\$(docker images --format '{{.Repository}} {{.Tag}} {{.ID}}' | grep "^${IMAGE_NAME} ${REMOTE_IMAGE_TAG} " | awk '{print \$3}')
  echo "[远程] 当前 ${REMOTE_IMAGE_TAG} 镜像ID: \$CURRENT_IMAGE_ID"

  echo "[远程] 清理旧备份镜像，只保留最近 ${BACKUP_KEEP} 个"
  BACKUP_TAGS=\$(docker images ${IMAGE_NAME} --format "{{.Tag}}" | grep '^backup_' | sort -r)
  TAGS_TO_REMOVE=\$(echo "\$BACKUP_TAGS" | tail -n +$((BACKUP_KEEP+1)))

  IMAGES_TO_REMOVE=""
  for tag in \$TAGS_TO_REMOVE; do
    IMAGE_ID=\$(docker images --format '{{.Repository}} {{.Tag}} {{.ID}}' | grep "^${IMAGE_NAME} \$tag " | awk '{print \$3}')
    if [ "\$IMAGE_ID" != "\$CURRENT_IMAGE_ID" ]; then
      IMAGES_TO_REMOVE="\$IMAGES_TO_REMOVE \$tag"
    else
      echo "[远程] 跳过删除备份镜像 ${IMAGE_NAME}:\$tag（与当前镜像ID相同）"
    fi
  done

  if [ ! -z "\$IMAGES_TO_REMOVE" ]; then
    for tag in \$IMAGES_TO_REMOVE; do
      echo "[远程] 删除旧备份镜像: ${IMAGE_NAME}:\$tag"
      docker rmi -f ${IMAGE_NAME}:\$tag || true
    done
  else
    echo "[远程] 无需删除备份镜像"
  fi

  echo "[远程] 启动服务: ${SERVICE_NAME}"
  cd ${REMOTE_COMPOSE_PATH}
  docker-compose up -d ${SERVICE_NAME}

  echo "[远程] 清理临时文件"
  rm -f ${REMOTE_PATH}/${TAR_NAME}
EOFREMOTE

# === 清理本地 tar 文件 ===
rm -f ${TAR_NAME}
log "${ENVIRONMENT} 环境部署完成 ✅"

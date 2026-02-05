#!/bin/bash

set -e

# === 参数 ===
ENVIRONMENT=${1:-dev}
ENV_FILE="env/${ENVIRONMENT}.env"

# === 日志函数 ===
log()    { echo -e "\033[1;34m[INFO]\033[0m $1"; }
warn()   { echo -e "\033[1;33m[WARN]\033[0m $1"; }
error()  { echo -e "\033[1;31m[ERROR]\033[0m $1"; }

# === 检查 .env 文件 ===
if [ ! -f "$ENV_FILE" ]; then
  error "环境文件不存在: $ENV_FILE"
  exit 1
fi

# === 加载环境变量 ===
log "加载环境变量文件: $ENV_FILE"
export $(grep -v '^#' $ENV_FILE | xargs)

# === 校验必要变量 ===
REQUIRED_VARS=(REMOTE_USER REMOTE_PASSWD REMOTE_HOST REMOTE_PORT REMOTE_PATH REMOTE_COMPOSE_PATH IMAGE_NAME SERVICE_NAME)
for var in "${REQUIRED_VARS[@]}"; do
  if [ -z "${!var}" ]; then
    error "缺少必要环境变量: $var"
    exit 1
  fi
done

# === 备份保留数量，默认3，.env可覆盖 ===
BACKUP_KEEP=${BACKUP_KEEP:-3}

# 本地构建镜像标签（默认时间戳）
LOCAL_IMAGE_TAG=$(date +%Y%m%d%H%M%S)
SANITIZED_IMAGE_NAME=$(echo "$IMAGE_NAME" | tr '/' '-')
TAR_NAME="${SANITIZED_IMAGE_NAME}_${LOCAL_IMAGE_TAG}.tar"

# 远程标签，默认 latest，可通过 env 配置覆盖
REMOTE_IMAGE_TAG=${REMOTE_IMAGE_TAG:-latest}

# === go build 编译 ===
log "开始编译..."
GOARCH=amd64 GOOS=linux go build -o app/xfusionmock xfusionmock.go

# === 本地构建镜像 ===
log "本地构建镜像: ${IMAGE_NAME}:${LOCAL_IMAGE_TAG}"
docker build -t ${IMAGE_NAME}:${LOCAL_IMAGE_TAG} .

# 获取本地新镜像ID
LOCAL_IMAGE_ID=$(docker image inspect -f '{{.Id}}' ${IMAGE_NAME}:${LOCAL_IMAGE_TAG})
log "本地新镜像ID: ${LOCAL_IMAGE_ID}"

# 远程获取当前镜像ID
#REMOTE_IMAGE_ID=$(sshpass -p "$REMOTE_PASSWD" ssh -p ${REMOTE_PORT} -o StrictHostKeyChecking=no ${REMOTE_USER}@${REMOTE_HOST} \
#  "docker image inspect -f '{{.Id}}' ${IMAGE_NAME}:${REMOTE_IMAGE_TAG} 2>/dev/null || echo ''")
#
#log "远程当前镜像ID: ${REMOTE_IMAGE_ID}"

# === 保存镜像为 tar 文件 ===
log "保存镜像为 tar 文件: ${TAR_NAME}"
docker save -o ${TAR_NAME} ${IMAGE_NAME}:${LOCAL_IMAGE_TAG}

# === 清理本地镜像 ===
log "清理本地镜像..."
docker image rm ${IMAGE_NAME}:${LOCAL_IMAGE_TAG}

# === 上传镜像到远程服务器（scp + sshpass） ===
log "上传镜像到远程服务器..."
RETRY_COUNT=0
MAX_RETRIES=3

# 先用 sshpass + ssh 自动创建远程目录（有密码）
sshpass -p "$REMOTE_PASSWD" ssh -p ${REMOTE_PORT} -o StrictHostKeyChecking=no ${REMOTE_USER}@${REMOTE_HOST} "mkdir -p ${REMOTE_PATH}"

# 再用 sshpass + scp 上传文件
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
sshpass -p "$REMOTE_PASSWD" ssh -o StrictHostKeyChecking=no ${REMOTE_USER}@${REMOTE_HOST} bash -s <<EOF
  set -e
  cd ${REMOTE_PATH}

  # 加载新镜像（含时间戳标签）
  echo "[远程] 加载新镜像: ${TAR_NAME}"
  docker load -i ${TAR_NAME}

  # 获取新镜像 ID
  NEW_IMAGE_ID=\$(docker images --format '{{.Repository}} {{.Tag}} {{.ID}}' | grep "^${IMAGE_NAME} ${LOCAL_IMAGE_TAG} " | awk '{print \$3}')
  echo "[远程] 新镜像ID: \$NEW_IMAGE_ID"

  # 获取当前旧镜像ID（如果有）
  if docker image inspect ${IMAGE_NAME}:${REMOTE_IMAGE_TAG} > /dev/null 2>&1; then
    OLD_IMAGE_ID=\$(docker images --format '{{.Repository}} {{.Tag}} {{.ID}}' | grep "^${IMAGE_NAME} ${REMOTE_IMAGE_TAG} " | awk '{print \$3}')
    echo "[远程] 旧 ${IMAGE_NAME}:${REMOTE_IMAGE_TAG} 镜像ID: \$OLD_IMAGE_ID"
  else
    OLD_IMAGE_ID=""
    echo "[远程] 旧 ${IMAGE_NAME}:${REMOTE_IMAGE_TAG} 镜像不存在"
  fi

  # 如果旧镜像存在且与新镜像ID不同，则备份旧镜像
  if [ -n "\$OLD_IMAGE_ID" ] && [ "\$OLD_IMAGE_ID" != "\$NEW_IMAGE_ID" ]; then
    BACKUP_TAG="backup_\$(date +%Y%m%d%H%M%S)"
    docker tag ${IMAGE_NAME}:${REMOTE_IMAGE_TAG} ${IMAGE_NAME}:\$BACKUP_TAG
    echo "[远程] 备份成功: ${IMAGE_NAME}:\$BACKUP_TAG"
  else
    echo "[远程] 不需要备份旧镜像"
  fi

  # 给新镜像打上 latest 标签，docker-compose 使用
  echo "[远程] 打标签为: ${REMOTE_IMAGE_TAG}"
  docker tag ${IMAGE_NAME}:${LOCAL_IMAGE_TAG} ${IMAGE_NAME}:${REMOTE_IMAGE_TAG}

  # 删除打 tag 前的时间戳镜像 tag（仅删除 tag，不删除镜像本体）
  if docker image inspect ${IMAGE_NAME}:${LOCAL_IMAGE_TAG} > /dev/null 2>&1; then
    echo "[远程] 删除临时 tag: ${IMAGE_NAME}:${LOCAL_IMAGE_TAG}"
    docker rmi ${IMAGE_NAME}:${LOCAL_IMAGE_TAG} || true
  fi

  # 当前 latest 镜像ID（更新标签后）
  CURRENT_IMAGE_ID=\$(docker images --format '{{.Repository}} {{.Tag}} {{.ID}}' | grep "^${IMAGE_NAME} ${REMOTE_IMAGE_TAG} " | awk '{print \$3}')
  echo "[远程] 当前 ${REMOTE_IMAGE_TAG} 镜像ID: \$CURRENT_IMAGE_ID"

  # 清理旧备份，只保留最近 ${BACKUP_KEEP} 个，且不删除与当前 latest 镜像ID相同的备份
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

  # 启动指定服务
  echo "[远程] 启动服务: ${SERVICE_NAME}"
  cd ${REMOTE_COMPOSE_PATH}
  docker-compose up -d ${SERVICE_NAME}

  # 清理上传的 tar 文件
  echo "[远程] 清理临时文件"
  rm -f ${REMOTE_PATH}/${TAR_NAME}
EOF

# === 清理本地 app 目录 ===
rm -rf app/
# === 清理本地 tar 文件 ===
rm -f ${TAR_NAME}
log "${ENVIRONMENT} 环境部署完成 ✅"
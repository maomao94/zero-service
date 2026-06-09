#!/bin/bash

set -e

ENVIRONMENT=${1:-dev}
ENV_FILE="env/${ENVIRONMENT}.env"

log()    { echo -e "\033[1;34m[INFO]\033[0m $1"; }
warn()   { echo -e "\033[1;33m[WARN]\033[0m $1"; }
error()  { echo -e "\033[1;31m[ERROR]\033[0m $1"; }

if [ ! -f "$ENV_FILE" ]; then
  error "环境文件不存在: $ENV_FILE"
  exit 1
fi

log "加载环境变量文件: $ENV_FILE"
export $(grep -v '^#' $ENV_FILE | xargs)

REQUIRED_VARS=(REMOTE_USER REMOTE_PASSWD REMOTE_HOST REMOTE_PORT REMOTE_PATH REMOTE_COMPOSE_PATH IMAGE_NAME SERVICE_NAME)
for var in "${REQUIRED_VARS[@]}"; do
  if [ -z "${!var}" ]; then
    error "缺少必要环境变量: $var"
    exit 1
  fi
done

BACKUP_KEEP=${BACKUP_KEEP:-3}

LOCAL_IMAGE_TAG=$(date +%Y%m%d%H%M%S)
SANITIZED_IMAGE_NAME=$(echo "$IMAGE_NAME" | tr '/' '-')
TAR_NAME="${SANITIZED_IMAGE_NAME}_${LOCAL_IMAGE_TAG}.tar"

REMOTE_IMAGE_TAG=${REMOTE_IMAGE_TAG:-latest}

log "开始编译..."
GOARCH=amd64 GOOS=linux go build -o app/bridgekafka bridgekafka.go

log "本地构建镜像: ${IMAGE_NAME}:${LOCAL_IMAGE_TAG}"
docker build -t ${IMAGE_NAME}:${LOCAL_IMAGE_TAG} .

LOCAL_IMAGE_ID=$(docker image inspect -f '{{.Id}}' ${IMAGE_NAME}:${LOCAL_IMAGE_TAG})
log "本地新镜像ID: ${LOCAL_IMAGE_ID}"

log "保存镜像为 tar 文件: ${TAR_NAME}"
docker save -o ${TAR_NAME} ${IMAGE_NAME}:${LOCAL_IMAGE_TAG}

log "清理本地镜像..."
docker image rm ${IMAGE_NAME}:${LOCAL_IMAGE_TAG}

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

log "连接远程服务器部署..."
sshpass -p "$REMOTE_PASSWD" ssh -o StrictHostKeyChecking=no ${REMOTE_USER}@${REMOTE_HOST} bash -s <<EOF
  set -e
  cd ${REMOTE_PATH}

  echo "[远程] 加载新镜像: ${TAR_NAME}"
  docker load -i ${TAR_NAME}

  NEW_IMAGE_ID=\$(docker images --format '{{.Repository}} {{.Tag}} {{.ID}}' | grep "^${IMAGE_NAME} ${LOCAL_IMAGE_TAG} " | awk '{print \$3}')
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
  docker tag ${IMAGE_NAME}:${LOCAL_IMAGE_TAG} ${IMAGE_NAME}:${REMOTE_IMAGE_TAG}

  if docker image inspect ${IMAGE_NAME}:${LOCAL_IMAGE_TAG} > /dev/null 2>&1; then
    echo "[远程] 删除临时 tag: ${IMAGE_NAME}:${LOCAL_IMAGE_TAG}"
    docker rmi ${IMAGE_NAME}:${LOCAL_IMAGE_TAG} || true
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
EOF

rm -rf app/
rm -f ${TAR_NAME}
log "${ENVIRONMENT} 环境部署完成 ✅"

#!/bin/bash

COMMAND="$1"
SERVICE_NAMES="$2"

# 检查 COMMAND 是否为空
if [ -z "$COMMAND" ]; then
  echo "Error: COMMAND must not be empty! Use 'restart', 'up', 'stop', or 'start'."
  exit 1
fi

TASK_NAME="${COMMAND}-docker"  # 生成任务名称，如 start-docker

# 如果 SERVICE_NAMES 为空，则将其设置为 service-name=""
if [ -z "$SERVICE_NAMES" ]; then
  SERVICE_NAMES=""  # 显式设置为空值
  echo "SERVICE_NAMES is empty, will execute all services."
else
  # 将逗号替换为空格
  SERVICE_NAMES=$(echo "$SERVICE_NAMES" | tr ',' ' ')
  echo "Executing for services: $SERVICE_NAMES"
fi

case "$COMMAND" in
  restart|up|stop|start)
    echo "Calling task to ${COMMAND} for services '${SERVICE_NAMES}' with task name $TASK_NAME..."
    task "$TASK_NAME" SERVICE_NAME="$SERVICE_NAMES"  # 调用对应的任务并传入服务名称
    ;;
  
  *)
    echo "Error: Unknown COMMAND '$COMMAND'. Use 'restart', 'up', 'stop', or 'start'."
    exit 1
    ;;
esac

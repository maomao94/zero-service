#!/bin/bash
# 停止 SIP 测试环境（Docker Compose）
# 用法: bash deploy/livekit/stop.sh

set -e
cd "$(dirname "$0")"

echo "停止 SIP 测试环境..."
docker compose down

echo "已停止"

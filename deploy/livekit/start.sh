#!/bin/bash
# 启动 SIP 测试环境（Docker Compose）
# 用法: bash deploy/livekit/start.sh

set -e
cd "$(dirname "$0")"

echo "启动 SIP 测试环境..."
docker compose up -d

echo ""
echo "服务状态:"
docker compose ps

echo ""
echo "端口说明:"
echo "  7880  - LiveKit Server API"
echo "  5060  - FreeSWITCH SIP 信令"
echo "  5070  - LiveKit SIP Server SIP 信令"
echo "  8021  - FreeSWITCH ESL 管理"
echo ""
echo "默认分机: 1000-1019，密码: 1234"
echo "API Key: devkeydevkeydevkeydevkeydevkeydevkey"

#!/bin/bash
# 启动 LiveKit + Egress 录制环境（Docker Compose）
# 用法: bash deploy/livekit/start.sh
# 说明: FreeSWITCH / LiveKit SIP Server 已迁移到 SIPMediaGW（本 compose 中已注释）

set -e
cd "$(dirname "$0")"

echo "启动 LiveKit + Egress ..."
docker compose up -d

echo ""
echo "服务状态:"
docker compose ps

echo ""
echo "端口说明:"
echo "  7880            - LiveKit Server API + WS"
echo "  60000-60100/udp - LiveKit WebRTC RTP 媒体流"
echo "  livekit-egress  - 录制 worker（内部，无对外端口）"
echo ""
echo "依赖: Redis（部署见 deploy/redis，宿主机 36379）"
echo "录制输出: deploy/livekit/recordings/（对外播放见 deploy/nginx）"
echo "API Key: devkeydevkeydevkeydevkeydevkeydevkey"

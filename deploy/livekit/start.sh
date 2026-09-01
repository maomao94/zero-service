#!/bin/bash
# Start livekit-server with webhook config
# Usage: bash deploy/start-livekit.sh

cd "$(dirname "$0")"

# Kill existing livekit-server
pkill -f livekit-server 2>/dev/null
sleep 1

# Start livekit-server
livekit-server --config livekit-server.yaml > /tmp/livekit-server.log 2>&1 &
LIVEKIT_PID=$!
sleep 2

# Check if running
if nc -z 127.0.0.1 7880 2>/dev/null; then
    echo "livekit-server started (PID: $LIVEKIT_PID) on :7880"
    echo "webhook -> http://127.0.0.1:11002/webhook/livekit"
else
    echo "livekit-server failed to start"
    cat /tmp/livekit-server.log
    exit 1
fi

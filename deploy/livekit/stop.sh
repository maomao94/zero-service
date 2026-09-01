#!/bin/bash
# Stop livekit-server
pkill -f "livekit-server --config" 2>/dev/null
echo "livekit-server stopped"

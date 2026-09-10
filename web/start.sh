#!/bin/bash

# Zero Service 前端项目启动脚本

echo "==================================="
echo "  Zero Service 前端项目启动脚本"
echo "==================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 获取脚本所在目录
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# 检查是否已安装依赖
check_dependencies() {
    local dir=$1
    local name=$2
    
    if [ ! -d "$dir/node_modules" ]; then
        echo -e "${YELLOW}正在安装 $name 的依赖...${NC}"
        cd "$dir" && npm install
        echo ""
    fi
}

# 启动项目
start_project() {
    local dir=$1
    local name=$2
    local port=$3
    
    echo -e "${GREEN}启动 $name (端口: $port)${NC}"
    cd "$dir" && npm run dev &
    echo ""
}

# 检查并安装依赖
echo "检查并安装依赖..."
echo ""

check_dependencies "$SCRIPT_DIR/live" "Live 视频会议"
check_dependencies "$SCRIPT_DIR/socketio" "SocketIO 网关测试"
check_dependencies "$SCRIPT_DIR/workspace" "Zero 工作台"

echo "==================================="
echo "  启动所有项目"
echo "==================================="
echo ""

# 启动所有项目
start_project "$SCRIPT_DIR/live" "Live 视频会议" "5178"
start_project "$SCRIPT_DIR/socketio" "SocketIO 网关测试" "5179"
start_project "$SCRIPT_DIR/workspace" "Zero 工作台" "5180"

echo "==================================="
echo "  所有项目已启动"
echo "==================================="
echo ""
echo -e "${BLUE}访问地址:${NC}"
echo -e "  工作台:          ${GREEN}http://localhost:5180${NC}"
echo -e "  Live 视频会议:   ${GREEN}http://localhost:5178${NC}"
echo -e "  SocketIO 测试:   ${GREEN}http://localhost:5179${NC}"
echo ""
echo -e "${YELLOW}按 Ctrl+C 停止所有项目${NC}"
echo ""

# 等待所有后台进程
wait

#!/bin/bash
# MQTT并发发布脚本（相对路径临时文件版）
# 临时文件存储在当前目录的./tmp子目录，自动创建和清理

# 默认配置
CONCURRENT=5        # 默认并发数
TOTAL=100           # 默认总执行次数
HOST="127.0.0.1"    # 默认服务器IP
PORT=1883           # 默认端口
USERNAME=""         # 用户名（空为匿名）
PASSWORD=""         # 密码
TOPIC="testgo"      # 默认主题
MESSAGE="hello"     # 默认消息
TMP_DIR="./tmp"     # 相对路径临时目录
FIFO_FILE="$TMP_DIR/mqtt_counter.fifo"  # FIFO管道路径

# 显示帮助
show_help() {
    echo "用法: $0 [选项]"
    echo "选项:"
    echo "  -c  并发数 (默认: $CONCURRENT)"
    echo "  -n  总执行次数 (默认: $TOTAL)"
    echo "  -H  服务器IP (默认: $HOST)"
    echo "  -p  端口 (默认: $PORT)"
    echo "  -u  用户名 (支持特殊字符，用单引号包裹)"
    echo "  -P  密码 (支持特殊字符，用单引号包裹)"
    echo "  -t  主题 (默认: $TOPIC)"
    echo "  -m  消息内容 (默认: $MESSAGE)"
    echo "  --help  显示帮助"
    exit 1
}

# 清理临时文件（脚本退出时执行）
cleanup() {
    if [ -p "$FIFO_FILE" ]; then
        rm -f "$FIFO_FILE"
    fi
    # 仅在临时目录为空时删除（避免误删用户文件）
    if [ -d "$TMP_DIR" ] && [ -z "$(ls -A "$TMP_DIR")" ]; then
        rmdir "$TMP_DIR"
    fi
}

# 解析参数
while [[ $# -gt 0 ]]; do
    case "$1" in
        -c) CONCURRENT="$2"; shift 2 ;;
        -n) TOTAL="$2"; shift 2 ;;
        -H) HOST="$2"; shift 2 ;;
        -p) PORT="$2"; shift 2 ;;
        -u) USERNAME="$2"; shift 2 ;;
        -P) PASSWORD="$2"; shift 2 ;;
        -t) TOPIC="$2"; shift 2 ;;
        -m) MESSAGE="$2"; shift 2 ;;
        --help) show_help ;;
        *) echo "未知选项: $1"; show_help ;;
    esac
done

# 参数验证
if ! [[ "$CONCURRENT" =~ ^[0-9]+$ ]] || [ "$CONCURRENT" -le 0 ]; then
    echo "错误: 并发数必须是正整数"
    exit 1
fi
if ! [[ "$TOTAL" =~ ^[0-9]+$ ]] || [ "$TOTAL" -le 0 ]; then
    echo "错误: 总执行次数必须是正整数"
    exit 1
fi
if ! [[ "$PORT" =~ ^[0-9]+$ ]] || [ "$PORT" -lt 1 ] || [ "$PORT" -gt 65535 ]; then
    echo "错误: 端口必须是1-65535之间的整数"
    exit 1
fi

# 创建临时目录（如果不存在）
if [ ! -d "$TMP_DIR" ]; then
    mkdir -p "$TMP_DIR" || {
        echo "错误: 无法创建临时目录 $TMP_DIR"
        exit 1
    }
fi

# 构建命令（处理特殊字符）
BASE_CMD="mosquitto_pub -h '$HOST' -p '$PORT' -t '$TOPIC' -m '$MESSAGE'"
AUTH=""

if [ -n "$USERNAME" ] && [ -n "$PASSWORD" ]; then
    ESCAPED_USER=$(printf "%q" "$USERNAME")
    ESCAPED_PASS=$(printf "%q" "$PASSWORD")
    AUTH="-u $ESCAPED_USER -P $ESCAPED_PASS"
elif [ -n "$USERNAME" ] || [ -n "$PASSWORD" ]; then
    echo "警告: 用户名和密码需同时提供，将使用匿名连接"
fi

MQTT_CMD="$BASE_CMD $AUTH"

# 显示配置
echo "======================================"
echo "MQTT并发发布配置"
echo "--------------------------------------"
echo "服务器: $HOST:$PORT"
echo "主题: $TOPIC"
echo "消息: $MESSAGE"
echo "认证: $(if [ -n "$AUTH" ]; then echo "启用($USERNAME)"; else echo "禁用"; fi)"
echo "并发数: $CONCURRENT"
echo "总次数: $TOTAL"
echo "临时目录: $(realpath "$TMP_DIR")"  # 显示绝对路径方便查看
echo "======================================"
echo "开始发送..."

# 记录开始时间
START=$(date +%s%N)

# 初始化FIFO管道（相对路径）
rm -f "$FIFO_FILE"  # 清理可能残留的管道
mkfifo "$FIFO_FILE" || {
    echo "错误: 无法创建FIFO管道 $FIFO_FILE"
    cleanup
    exit 1
}

# 统计成功数
SUCCESS=0
cat "$FIFO_FILE" | while read; do
    SUCCESS=$((SUCCESS + 1))
done &
COUNTER_PID=$!

# 核心并发逻辑
seq 1 $TOTAL | xargs -I {} -P $CONCURRENT bash -c "$(printf "%q" "$MQTT_CMD") > /dev/null 2>&1; if [ \$? -eq 0 ]; then echo 1 > '$FIFO_FILE'; fi"

# 等待所有进程完成，清理资源
wait
cleanup  # 调用清理函数
kill $COUNTER_PID 2>/dev/null

# 计算结果
FAILED=$((TOTAL - SUCCESS))
END=$(date +%s%N)
DURATION=$(( (END - START) / 1000000000 ))
[ $DURATION -eq 0 ] && DURATION=1
THROUGHPUT=$(( SUCCESS / DURATION ))

# 显示结果
echo
echo "======================================"
echo "执行结果"
echo "--------------------------------------"
echo "总次数: $TOTAL"
echo "成功: $SUCCESS"
echo "失败: $FAILED"
echo "耗时: $DURATION 秒"
echo "吞吐量: $THROUGHPUT 条/秒"
echo "======================================"
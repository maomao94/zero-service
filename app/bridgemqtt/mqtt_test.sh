#!/bin/bash
# MQTT并发发布脚本（修复引号语法错误版）
# 支持含特殊字符的用户名/密码，解决引号不匹配问题

# 默认配置
CONCURRENT=5        # 默认并发数
TOTAL=100           # 默认总执行次数
HOST="127.0.0.1"    # 默认服务器IP
PORT=1883           # 默认端口
USERNAME=""         # 用户名（空为匿名）
PASSWORD=""         # 密码
TOPIC="testgo"      # 默认主题
MESSAGE="hello"     # 默认消息

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

# 构建命令（关键修复：用双引号+printf转义处理特殊字符）
BASE_CMD="mosquitto_pub -h '$HOST' -p '$PORT' -t '$TOPIC' -m '$MESSAGE'"
AUTH=""

if [ -n "$USERNAME" ] && [ -n "$PASSWORD" ]; then
    # 用printf转义单引号和特殊字符，确保命令安全拼接
    ESCAPED_USER=$(printf "%q" "$USERNAME")
    ESCAPED_PASS=$(printf "%q" "$PASSWORD")
    AUTH="-u $ESCAPED_USER -P $ESCAPED_PASS"
elif [ -n "$USERNAME" ] || [ -n "$PASSWORD" ]; then
    echo "警告: 用户名和密码需同时提供，将使用匿名连接"
fi

# 组合最终命令
if [ -n "$AUTH" ]; then
    MQTT_CMD="$BASE_CMD $AUTH"
else
    MQTT_CMD="$BASE_CMD"
fi

# 显示配置（隐藏密码，避免泄露）
echo "======================================"
echo "MQTT并发发布配置"
echo "--------------------------------------"
echo "服务器: $HOST:$PORT"
echo "主题: $TOPIC"
echo "消息: $MESSAGE"
echo "认证: $(if [ -n "$AUTH" ]; then echo "启用($USERNAME)"; else echo "禁用"; fi)"
echo "并发数: $CONCURRENT"
echo "总次数: $TOTAL"
echo "======================================"
echo "开始发送..."

# 记录开始时间
START=$(date +%s%N)

# 统计成功数（FIFO管道计数）
SUCCESS=0
mkfifo /tmp/mqtt_counter.fifo
cat /tmp/mqtt_counter.fifo | while read; do
    SUCCESS=$((SUCCESS + 1))
done &
COUNTER_PID=$!

# 核心并发逻辑（用printf传递命令，避免引号解析错误）
seq 1 $TOTAL | xargs -I {} -P $CONCURRENT bash -c "$(printf "%q" "$MQTT_CMD") > /dev/null 2>&1; if [ \$? -eq 0 ]; then echo 1 > /tmp/mqtt_counter.fifo; fi"

# 等待所有进程完成，清理资源
wait
rm -f /tmp/mqtt_counter.fifo
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
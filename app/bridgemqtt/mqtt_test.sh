#!/bin/bash
# MQTT并发发布脚本（无文件操作版）
# 功能：纯参数配置，并发发送MQTT消息

# 默认配置
CONCURRENT=5        # 默认并发数
TOTAL=100           # 默认总执行次数
HOST="10.10.1.191"    # 默认服务器IP
PORT=1883           # 默认端口
USERNAME="admin"         # 用户名（空为匿名）
PASSWORD="Znyjzx098!"         # 密码
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
    echo "  -u  用户名 (可选)"
    echo "  -P  密码 (可选)"
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

# 构建命令（处理认证）
AUTH=""
if [ -n "$USERNAME" ] && [ -n "$PASSWORD" ]; then
    AUTH="-u '$USERNAME' -P '$PASSWORD'"
elif [ -n "$USERNAME" ] || [ -n "$PASSWORD" ]; then
    echo "警告: 用户名和密码需同时提供，将使用匿名连接"
fi

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
echo "======================================"
echo "开始发送..."

# 记录开始时间
START=$(date +%s%N)

# 统计成功数（用变量+管道避免文件，适合简单场景）
SUCCESS=0
# 用FIFO实现多进程计数（比临时文件更轻量）
mkfifo /tmp/mqtt_counter.fifo
cat /tmp/mqtt_counter.fifo | while read; do
    SUCCESS=$((SUCCESS + 1))
done &
COUNTER_PID=$!

# 核心并发逻辑
seq 1 $TOTAL | xargs -I {} -P $CONCURRENT bash -c '
    # 执行发布命令
    eval "mosquitto_pub -h '$HOST' -p '$PORT' -t '$TOPIC' -m '$MESSAGE' '$AUTH' > /dev/null 2>&1"
    # 成功则写入计数器
    if [ $? -eq 0 ]; then
        echo 1 > /tmp/mqtt_counter.fifo
    fi
'

# 等待所有进程完成，关闭计数器
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

#!/bin/bash
# MQTT并发性能测试脚本
# 功能：向指定MQTT主题并发发送指定数量的消息，用于测试消费端性能
# 使用方法：./mqtt_perf_test.sh -c 并发数 -n 总消息数 -h broker地址 -t 主题

# 默认配置
CONCURRENT=5          # 默认并发数
TOTAL_MESSAGES=100    # 默认总消息数
BROKER_HOST="localhost" # 默认Broker地址
BROKER_PORT=1883      # 默认端口
TOPIC="test/performance" # 默认测试主题
QOS=1                 # 默认QoS等级
USERNAME=""           # 用户名（默认空）
PASSWORD=""           # 密码（默认空）
MESSAGE_SIZE=1024     # 消息大小（字节）
VERBOSE=0             # 是否显示详细日志

# 显示帮助信息
show_help() {
    echo "MQTT并发性能测试脚本"
    echo "用法: $0 [选项]"
    echo "选项:"
    echo "  -c  并发数 (默认: $CONCURRENT)"
    echo "  -n  总消息数 (默认: $TOTAL_MESSAGES)"
    echo "  -h  MQTT Broker地址 (默认: $BROKER_HOST)"
    echo "  -p  MQTT Broker端口 (默认: $BROKER_PORT)"
    echo "  -t  目标主题 (默认: $TOPIC)"
    echo "  -q  QoS等级 (0/1/2, 默认: $QOS)"
    echo "  -u  用户名 (可选)"
    echo "  -P  密码 (可选)"
    echo "  -s  消息大小(字节) (默认: $MESSAGE_SIZE)"
    echo "  -v  显示详细日志"
    echo "  --help  显示帮助信息"
    exit 1
}

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case "$1" in
        -c) CONCURRENT="$2"; shift 2 ;;
        -n) TOTAL_MESSAGES="$2"; shift 2 ;;
        -h) BROKER_HOST="$2"; shift 2 ;;
        -p) BROKER_PORT="$2"; shift 2 ;;
        -t) TOPIC="$2"; shift 2 ;;
        -q) QOS="$2"; shift 2 ;;
        -u) USERNAME="$2"; shift 2 ;;
        -P) PASSWORD="$2"; shift 2 ;;
        -s) MESSAGE_SIZE="$2"; shift 2 ;;
        -v) VERBOSE=1; shift ;;
        --help) show_help ;;
        *) echo "未知选项: $1"; show_help ;;
    esac
done

# 验证参数
if ! [[ "$CONCURRENT" =~ ^[0-9]+$ ]] || [ "$CONCURRENT" -le 0 ]; then
    echo "错误: 并发数必须是正整数"
    exit 1
fi

if ! [[ "$TOTAL_MESSAGES" =~ ^[0-9]+$ ]] || [ "$TOTAL_MESSAGES" -le 0 ]; then
    echo "错误: 总消息数必须是正整数"
    exit 1
fi

if ! [[ "$QOS" =~ ^[0-2]$ ]]; then
    echo "错误: QoS等级必须是0、1或2"
    exit 1
fi

# 生成测试消息内容
generate_message() {
    # 生成指定大小的随机字符串
    head -c $MESSAGE_SIZE /dev/urandom | base64 | head -c $MESSAGE_SIZE
}

# 预生成消息内容（避免重复生成）
TEST_MESSAGE=$(generate_message)

# 显示测试配置
echo "======================================"
echo "MQTT性能测试配置"
echo "======================================"
echo "Broker地址:      $BROKER_HOST:$BROKER_PORT"
echo "目标主题:        $TOPIC"
echo "QoS等级:         $QOS"
echo "并发数:          $CONCURRENT"
echo "总消息数:        $TOTAL_MESSAGES"
echo "单条消息大小:    $MESSAGE_SIZE 字节"
echo "认证信息:        $(if [ -n "$USERNAME" ]; then echo "已设置"; else echo "未设置"; fi)"
echo "======================================"
echo "开始测试..."
echo

# 记录开始时间
START_TIME=$(date +%s%N)

# 生成消息ID列表并并发发送
seq 1 $TOTAL_MESSAGES | xargs -I {} -P $CONCURRENT bash -c '
    # 导入外部变量
    BROKER_HOST='"$BROKER_HOST"'
    BROKER_PORT='"$BROKER_PORT"'
    TOPIC='"$TOPIC"'
    QOS='"$QOS"'
    USERNAME='"$USERNAME"'
    PASSWORD='"$PASSWORD"'
    TEST_MESSAGE='"$TEST_MESSAGE"'
    VERBOSE='"$VERBOSE"'

    # 构建mosquitto_pub命令
    CMD="mosquitto_pub -h $BROKER_HOST -p $BROKER_PORT -t \"$TOPIC\" -q $QOS -m \"Message {}\: $TEST_MESSAGE\""

    # 添加认证信息
    if [ -n "$USERNAME" ]; then
        CMD="$CMD -u \"$USERNAME\" -P \"$PASSWORD\""
    fi

    # 执行命令
    if [ $VERBOSE -eq 1 ]; then
        echo "发送消息 {}: $CMD"
        eval $CMD
    else
        eval $CMD > /dev/null 2>&1
    fi

    # 检查命令执行结果
    if [ $? -ne 0 ]; then
        echo "错误: 消息 {} 发送失败" >&2
    fi
'

# 记录结束时间
END_TIME=$(date +%s%N)

# 计算耗时（秒）
DURATION=$(( (END_TIME - START_TIME) / 1000000000 ))
if [ $DURATION -eq 0 ]; then
    DURATION=1  # 避免除以零
fi

# 计算性能指标
THROUGHPUT=$(( TOTAL_MESSAGES / DURATION ))

# 显示测试结果
echo
echo "======================================"
echo "测试完成!"
echo "======================================"
echo "总消息数:        $TOTAL_MESSAGES"
echo "并发数:          $CONCURRENT"
echo "总耗时:          $DURATION 秒"
echo "吞吐量:          $THROUGHPUT 条/秒"
echo "======================================"
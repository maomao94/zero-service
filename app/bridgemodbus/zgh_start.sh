#!/usr/bin/env bash
# bridgemodbus 本地压测脚本（ghz）。
#
# 依赖：ghz（brew install ghz）。
# 启动服务：
#   cd app/bridgemodbus && go run . -f etc/bridgemodbus.yaml
#
# 用法：bash zgh_start.sh [target] [concurrency] [duration]
#   target      服务地址，默认 127.0.0.1:25004
#   concurrency 并发调用数，默认 1（同一 code 的 Modbus 收发串行）
#   duration    单轮压测时长，默认 1m
#
# 注意：
#   - 同一 code 复用一个 Client，增加 -c 只会增加排队，不会增加设备 TCP 连接。
#   - 请求里的 modbusCode=local 需要先在 modbus_slave_config 配好；使用默认配置可改成空字符串。
#   - 不限制 rps（ghz 默认 0 表示不限速），实际吞吐由设备或模拟器的响应速度决定。

set -euo pipefail

TARGET="${1:-127.0.0.1:25004}"
CONCURRENCY="${2:-1}"
DURATION="${3:-1m}"

ghz --insecure \
  --proto ./bridgemodbus.proto \
  --call bridgemodbus.BridgeModbus/ReadCoils \
  -d '{"modbusCode": "local", "address": 0, "quantity": 1000}' \
  -c "${CONCURRENCY}" \
  --duration "${DURATION}" \
  --timeout 5s \
  --output read_coils_report.html \
  --format html \
  "${TARGET}"

ghz --insecure \
  --proto ./bridgemodbus.proto \
  --call bridgemodbus.BridgeModbus/ReadHoldingRegisters \
  -d '{"modbusCode": "local", "address": 0, "quantity": 125}' \
  -c "${CONCURRENCY}" \
  --duration "${DURATION}" \
  --timeout 5s \
  --output read_holding_registers_report.html \
  --format html \
  "${TARGET}"

###
# 历史压测记录（2025-10-11，旧脚本：端口 25003、-c 1000）
# 当时连接池实现与当前不同，且请求参数未校验，可能包含连接失败后的快速返回，
# 不能作为当前容量基线，仅保留作对比参考。
#2025-10-11T10:54:27.216+08:00    stat   (bridgemodbus.rpc) - qps: 32410.3/s, drops: 0, avg time: 15.9ms, med: 0.0ms, 90th: 0.0ms, 99th: 3.6ms, 99.9th: 2000.7ms app=bridgemodbus.rpc    caller=stat/metrics.go:210
#2025-10-11T10:55:27.300+08:00    stat   (bridgemodbus.rpc) - qps: 32498.8/s, drops: 0, avg time: 14.4ms, med: 0.0ms, 90th: 0.0ms, 99th: 3.4ms, 99.9th: 2000.7ms app=bridgemodbus.rpc    caller=stat/metrics.go:210
#2025-10-11T10:56:27.266+08:00    stat   (bridgemodbus.rpc) - qps: 30494.0/s, drops: 0, avg time: 15.2ms, med: 0.0ms, 90th: 0.0ms, 99th: 4.9ms, 99.9th: 2000.7ms app=bridgemodbus.rpc    caller=stat/metrics.go:210
#2025-10-11T10:57:27.405+08:00    stat   (bridgemodbus.rpc) - qps: 34456.7/s, drops: 0, avg time: 13.4ms, med: 0.0ms, 90th: 0.0ms, 99th: 3.3ms, 99.9th: 2000.7ms app=bridgemodbus.rpc    caller=stat/metrics.go:210
#2025-10-11T10:58:27.687+08:00    stat   (bridgemodbus.rpc) - qps: 38777.7/s, drops: 0, avg time: 11.9ms, med: 0.0ms, 90th: 0.0ms, 99th: 2.5ms, 99.9th: 2000.5ms caller=stat/metrics.go:210      app=bridgemodbus.rpc
#2025-10-11T10:59:27.515+08:00    stat   (bridgemodbus.rpc) - qps: 37004.3/s, drops: 0, avg time: 12.4ms, med: 0.0ms, 90th: 0.0ms, 99th: 2.6ms, 99.9th: 2000.6ms app=bridgemodbus.rpc    caller=stat/metrics.go:210
#2025-10-11T11:00:27.508+08:00    stat   (bridgemodbus.rpc) - qps: 37434.9/s, drops: 0, avg time: 12.2ms, med: 0.0ms, 90th: 0.0ms, 99th: 2.6ms, 99.9th: 2000.6ms app=bridgemodbus.rpc    caller=stat/metrics.go:210
#2025-10-11T11:01:27.515+08:00    stat   (bridgemodbus.rpc) - qps: 37646.5/s, drops: 0, avg time: 12.1ms, med: 0.0ms, 90th: 0.0ms, 99th: 2.8ms, 99.9th: 2000.5ms app=bridgemodbus.rpc    caller=stat/metrics.go:210
#2025-10-11T11:02:27.403+08:00    stat   (bridgemodbus.rpc) - qps: 35495.3/s, drops: 0, avg time: 13.2ms, med: 0.0ms, 90th: 0.0ms, 99th: 2.9ms, 99.9th: 2000.6ms app=bridgemodbus.rpc    caller=stat/metrics.go:210

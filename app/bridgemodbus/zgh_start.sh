ghz --insecure \
  --proto ./bridgemodbus.proto \
  --call bridgemodbus.BridgeModbus/ReadCoils \
  -d '{"modbusCode": "local", "address": 0, "quantity": 1000}' \
  -c 1000 \
  --rps 100000 \
  --duration 10m \
  --timeout 5s \
  --output read_coils_report.html \
  --format html \
  127.0.0.1:25003

ghz --insecure \
  --proto ./bridgemodbus.proto \
  --call bridgemodbus.BridgeModbus/ReadHoldingRegisters \
  -d '{"modbusCode": "local", "address": 0, "quantity": 125}' \
  -c 1000 \
  --rps 100000 \
  --duration 10m \
  --timeout 5s \
  --output read_holding_registers_report.html \
  --format html \
  127.0.0.1:25003


###
#压测结果
#2025-10-11T10:54:27.216+08:00    stat   (bridgemodbus.rpc) - qps: 32410.3/s, drops: 0, avg time: 15.9ms, med: 0.0ms, 90th: 0.0ms, 99th: 3.6ms, 99.9th: 2000.7ms app=bridgemodbus.rpc    caller=stat/metrics.go:210
#2025-10-11T10:55:27.300+08:00    stat   (bridgemodbus.rpc) - qps: 32498.8/s, drops: 0, avg time: 14.4ms, med: 0.0ms, 90th: 0.0ms, 99th: 3.4ms, 99.9th: 2000.7ms app=bridgemodbus.rpc    caller=stat/metrics.go:210
#2025-10-11T10:56:27.266+08:00    stat   (bridgemodbus.rpc) - qps: 30494.0/s, drops: 0, avg time: 15.2ms, med: 0.0ms, 90th: 0.0ms, 99th: 4.9ms, 99.9th: 2000.7ms app=bridgemodbus.rpc    caller=stat/metrics.go:210
#2025-10-11T10:57:27.405+08:00    stat   (bridgemodbus.rpc) - qps: 34456.7/s, drops: 0, avg time: 13.4ms, med: 0.0ms, 90th: 0.0ms, 99th: 3.3ms, 99.9th: 2000.7ms app=bridgemodbus.rpc    caller=stat/metrics.go:210
#2025-10-11T10:58:27.687+08:00    stat   (bridgemodbus.rpc) - qps: 38777.7/s, drops: 0, avg time: 11.9ms, med: 0.0ms, 90th: 0.0ms, 99th: 2.5ms, 99.9th: 2000.5ms caller=stat/metrics.go:210      app=bridgemodbus.rpc
#2025-10-11T10:59:27.515+08:00    stat   (bridgemodbus.rpc) - qps: 37004.3/s, drops: 0, avg time: 12.4ms, med: 0.0ms, 90th: 0.0ms, 99th: 2.6ms, 99.9th: 2000.6ms app=bridgemodbus.rpc    caller=stat/metrics.go:210
#2025-10-11T11:00:27.508+08:00    stat   (bridgemodbus.rpc) - qps: 37434.9/s, drops: 0, avg time: 12.2ms, med: 0.0ms, 90th: 0.0ms, 99th: 2.6ms, 99.9th: 2000.6ms app=bridgemodbus.rpc    caller=stat/metrics.go:210
#2025-10-11T11:01:27.515+08:00    stat   (bridgemodbus.rpc) - qps: 37646.5/s, drops: 0, avg time: 12.1ms, med: 0.0ms, 90th: 0.0ms, 99th: 2.8ms, 99.9th: 2000.5ms app=bridgemodbus.rpc    caller=stat/metrics.go:210
#2025-10-11T11:02:27.403+08:00    stat   (bridgemodbus.rpc) - qps: 35495.3/s, drops: 0, avg time: 13.2ms, med: 0.0ms, 90th: 0.0ms, 99th: 2.9ms, 99.9th: 2000.6ms app=bridgemodbus.rpc    caller=stat/metrics.go:210
ghz --insecure \
  --proto ./bridgemodbus.proto \
  --call bridgemodbus.BridgeModbus/ReadCoils \
  -d '{"modbusCode": "local", "address": 0, "quantity": 1000}' \
  -c 80 \
  --rps 2000 \
  --duration 10m \
  --timeout 5s \
  --output read_coils_report.html \
  --format html \
  10.10.1.213:25003

ghz --insecure \
  --proto ./bridgemodbus.proto \
  --call bridgemodbus.BridgeModbus/ReadHoldingRegisters \
  -d '{"modbusCode": "local", "address": 0, "quantity": 125}' \
  -c 80 \
  --rps 2000 \
  --duration 10m \
  --timeout 5s \
  --output read_holding_registers_report.html \
  --format html \
  10.10.1.213:25003
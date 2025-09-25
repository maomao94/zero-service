#!/usr/bin/env bash

echo "开始生成"
goctl rpc protoc bridgemodbus.proto --go_out=. --go-grpc_out=. --zrpc_out=. --client=false

protoc \
  --proto_path=. \
  --descriptor_set_out=./bridgemodbus.pb \
  bridgemodbus.proto
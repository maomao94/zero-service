#!/usr/bin/env bash

echo "开始生成"
goctl rpc protoc bridgemqtt.proto --go_out=. --go-grpc_out=. --zrpc_out=. --client=false

protoc \
  --proto_path=. \
  --descriptor_set_out=./bridgemqtt.pb \
  bridgemqtt.proto
#!/usr/bin/env bash

echo "开始生成"
goctl rpc protoc openapi.proto \
  --go_out=. \
  --go-grpc_out=. \
  --zrpc_out=. \
  --client=false \
  --proto_path=. \
  --proto_path=../../third_party
protoc \
  --proto_path=. \
  --proto_path=../../third_party \
  --include_imports \
  --descriptor_set_out=./openapi.pb \
  --openapiv2_out ../../swagger \
  openapi.proto

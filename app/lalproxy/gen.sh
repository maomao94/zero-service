#!/usr/bin/env bash

echo "开始生成"
goctl rpc protoc lalproxy.proto \
  --go_out=. \
  --go-grpc_out=. \
  --zrpc_out=. \
  --client=false \
  --proto_path=. \
  --proto_path=../../third_party
protoc \
  --proto_path=. \
  --go_opt=json_tag \
  --proto_path=../../third_party \
  --validate_out=lang=go:. \
  --include_imports \
  --descriptor_set_out=./lalproxy.pb \
  --openapiv2_out ../../swagger \
  lalproxy.proto

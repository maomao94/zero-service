#!/usr/bin/env bash

echo "开始生成"
goctl rpc protoc xfusionmock.proto \
  --go_out=. \
  --go-grpc_out=. \
  --zrpc_out=. \
  --client=false \
  --proto_path=. \
  --proto_path=../../third_party
protoc \
  --proto_path=. \
  --proto_path=../../third_party \
  --validate_out=lang=go:. \
  --include_imports \
  --descriptor_set_out=./xfusionmock.pb \
  --openapiv2_out ../../swagger \
  xfusionmock.proto
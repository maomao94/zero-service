#!/usr/bin/env bash

echo "开始生成"
goctl rpc protoc iecstream.proto \
  --go_out=. \
  --go-grpc_out=. \
  --zrpc_out=. \
  --client=false \
  --proto_path=. \
  --proto_path=../../third/googleapis \
  --proto_path=../../third
protoc \
  --proto_path=. \
  --proto_path=../../third/googleapis \
  --proto_path=../../third \
  --validate_out=lang=go:. \
  --include_imports \
  --descriptor_set_out=./iecstream.pb \
  --openapiv2_out ../../swagger \
  iecstream.proto
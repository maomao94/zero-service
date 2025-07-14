#!/usr/bin/env bash

echo "开始生成"
protoc \
  --proto_path=. \
  --proto_path=../third_party \
  --go_out=../third_party \
  --go-grpc_out=../third_party \
  extproto.proto
#!/usr/bin/env bash

echo "开始生成"
goctl rpc protoc --go_out=. --go-grpc_out=. --zrpc_out=. --client=false xfusionmock.proto
protoc --descriptor_set_out=./xfusionmock.pb xfusionmock.proto

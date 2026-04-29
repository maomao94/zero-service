#!/usr/bin/env bash

echo "开始生成"
goctl rpc protoc djicloud.proto --go_out=. --go-grpc_out=. --zrpc_out=. --client=false

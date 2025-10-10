#!/usr/bin/env bash

echo "开始生成"
goctl rpc protoc alarm.proto --go_out=. --go-grpc_out=. --zrpc_out=. --client=false
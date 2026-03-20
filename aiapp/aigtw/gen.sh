#!/usr/bin/env bash

echo "开始生成"
goctl api format --dir=./
goctl api go --api=./aigtw.api --dir=./

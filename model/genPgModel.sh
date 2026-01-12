#!/usr/bin/env bash

dbname=$1
tables=$2  # 单表传表名，多表传逗号分隔的表名（如"plan,plan_exec_item"）
modeldir=./genModel

# PostgreSQL 数据库配置（根据你的实际环境修改）
host=127.0.0.1
port=5432
username=postgres
passwd=postgres
schema=public  # PostgreSQL的schema，默认public

echo "开始创建 PostgreSQL 库：$dbname 的表：$tables"

# 先创建生成目录（避免不存在）
mkdir -p ${modeldir}

# 核心：goctl生成PostgreSQL模型的正确命令
goctl model pg datasource \
  -url="postgres://${username}:${passwd}@${host}:${port}/${dbname}?sslmode=disable&TimeZone=Asia/Shanghai" \
  -table="${tables}" \
  -dir="${modeldir}" \
  -cache=false \
  -schema=${schema} \
  --style=gozero
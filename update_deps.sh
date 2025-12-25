#!/bin/bash

# 自动更新 Go 项目依赖脚本
# 功能：使用 depu 命令检查可更新依赖，并自动更新到最新版本

# 1. 运行 depu 命令检查可更新的依赖包
echo "正在运行 depu 检查可更新依赖..."
deps_output=$(depu)

# 2. 解析 depu 输出结果，提取需要更新的包信息
#    - sed -n '3,/^$/p': 从第3行开始，直到遇到空行结束，跳过表头
#    - sed '/^[[:space:]]*$/d': 过滤掉空行，处理 macOS 兼容性
echo "正在解析更新信息..."
pkg_lines=$(echo "$deps_output" | sed -n '3,/^$/p' | sed '/^[[:space:]]*$/d')

# 3. 检查是否有可更新的依赖
if [ -z "$pkg_lines" ]; then
    echo "没有可更新的依赖包。"
    exit 0
fi

echo "发现可更新依赖，开始更新..."

# 4. 遍历每一行依赖信息，执行更新操作
while IFS= read -r line; do
    # 使用 awk 提取包名和最新版本
    # -F'|': 使用 | 作为字段分隔符
    # gsub(/^[[:space:]]+|[[:space:]]+$/, "", $1): 去除字段前后的空格
    pkg=$(echo "$line" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $1); print $1}')
    latest=$(echo "$line" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $3); print $3}')
    
    # 确保包名和版本都不为空
    if [ -n "$pkg" ] && [ -n "$latest" ]; then
        echo "正在更新 $pkg 到最新版本 $latest"
        # 使用 go get -u 命令更新依赖到指定版本
        go get -u "$pkg@$latest"
    fi
done <<< "$pkg_lines"

# 5. 更新完成后，运行 go mod tidy 整理依赖关系
echo "所有依赖更新完成。正在运行 go mod tidy 整理依赖..."
go mod tidy
echo "依赖更新脚本执行完成！"
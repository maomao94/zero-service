#!/bin/bash
# 生成前端发布测试数据
# 用法: cd cli/dtui/testdata/deploy && ./build-testdata.sh

set -e
cd "$(dirname "$0")"

echo "=== 生成前端发布测试数据 ==="

# 1. 从 sample-folder 创建 zip
rm -f sample.zip
cd sample-folder && zip -r ../sample.zip . && cd ..
echo "✓ sample.zip (从 sample-folder 打包)"

# 2. 创建带嵌套目录的 zip
rm -f nested.zip
mkdir -p _tmp-nested/assets/css _tmp-nested/assets/js
cp sample-folder/index.html _tmp-nested/
echo "body { color: red; }" > _tmp-nested/assets/css/style.css
echo "console.log('hello');" > _tmp-nested/assets/js/app.js
cd _tmp-nested && zip -r ../nested.zip . && cd ..
rm -rf _tmp-nested
echo "✓ nested.zip (带 assets/css + assets/js 子目录)"

# 3. 创建空文件夹测试目录
rm -rf empty-folder
mkdir -p empty-folder
echo "✓ empty-folder/ (空目录，测试空部署)"

echo ""
echo "=== 测试数据已就绪 ==="
echo "在 dtui 前端发布视图中按 d，输入以下路径测试："
echo "  文件夹: $(pwd)/sample-folder"
echo "  zip:    $(pwd)/sample.zip"
echo "  嵌套:   $(pwd)/nested.zip"
echo "  空目录: $(pwd)/empty-folder"

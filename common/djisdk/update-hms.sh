#!/usr/bin/env bash
# 下载 DJI 官方 HMS 告警文案字典并覆盖本地 hms.json
# 文档地址: https://developer.dji.com/doc/cloud-api-tutorial/cn/feature-set/dock-feature-set/hms.html
set -euo pipefail

cd "$(dirname "$0")"

URL="https://terra-1-g.djicdn.com/fee90c2e03e04e8da67ea6f56365fc76/SDK%20%E6%96%87%E6%A1%A3/CloudAPI/hms.json"

echo "开始下载"
curl -fsSL "$URL" -o hms.json.new

if ! python3 -m json.tool hms.json.new > /dev/null; then
  echo "下载内容不是合法 JSON，已放弃覆盖" >&2
  rm -f hms.json.new
  exit 1
fi

mv hms.json.new hms.json
echo "已覆盖 hms.json，共 $(wc -l < hms.json | tr -d ' ') 行"
go test ./ -run TestHms

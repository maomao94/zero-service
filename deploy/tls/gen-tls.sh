#!/usr/bin/env bash
set -euo pipefail

# 自签 TLS 证书生成脚本
# 职责单一：只生成证书/私钥，不改动任何部署配置、不修改系统信任链
#
# 用途：为 deploy/nginx 反向代理提供 TLS 证书（LiveKit 经反向代理访问，不单独配 TLS）
#
# 用法：
#   # 开发环境（默认 localhost + loopback）
#   ./gen-tls.sh
#
#   # 测试/生产环境（注入域名 + 额外 IP/域名）
#   DOMAIN=meet.example.com \
#   EXTRA_DNS="api.meet.example.com,*.meet.example.com" \
#   EXTRA_IPS="10.0.0.1,172.16.0.1" \
#   ./gen-tls.sh
#
# 生成物：
#   $CERT_DIR/server.crt   证书
#   $CERT_DIR/server.key   私钥
#   $CERT_DIR/san.cnf      证书签名请求用的 SAN 配置
#
# 信任问题：自签证书需由使用方自行决定是否信任（本脚本不做任何系统级操作）。

CERT_DIR="${CERT_DIR:-$(cd "$(dirname "$0")" && pwd)}"
DAYS="${CERT_DAYS:-3650}"
CN="${DOMAIN:-localhost}"

# 解析额外 SAN 条目（逗号分隔）
IFS=',' read -ra EXTRA_DNS_ARR <<< "${EXTRA_DNS:-}"
IFS=',' read -ra EXTRA_IPS_ARR <<< "${EXTRA_IPS:-}"

echo "=== 生成自签 TLS 证书 ==="
echo "证书目录: $CERT_DIR"
echo "CN / 主域名: $CN"
echo "有效期: ${DAYS} 天"
echo ""

mkdir -p "$CERT_DIR"

openssl genrsa -out "$CERT_DIR/server.key" 2048 2>/dev/null

# 动态生成 SAN 配置
{
cat <<EOF
[req]
default_bits = 2048
prompt = no
default_md = sha256
distinguished_name = dn
x509_extensions = v3_req

[dn]
C = CN
ST = Beijing
L = Beijing
O = Zero Service
OU = Dev
CN = $CN

[v3_req]
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = *.local
DNS.3 = $CN
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

dns_idx=4
for d in ${EXTRA_DNS_ARR[@]+"${EXTRA_DNS_ARR[@]}"}; do
  d=$(echo "$d" | xargs)
  [ -z "$d" ] && continue
  echo "DNS.$dns_idx = $d"
  ((dns_idx++))
done

ip_idx=3
for ip in ${EXTRA_IPS_ARR[@]+"${EXTRA_IPS_ARR[@]}"}; do
  ip=$(echo "$ip" | xargs)
  [ -z "$ip" ] && continue
  echo "IP.$ip_idx = $ip"
  ((ip_idx++))
done
} > "$CERT_DIR/san.cnf"

openssl req -x509 -nodes -days "$DAYS" \
  -key "$CERT_DIR/server.key" \
  -out "$CERT_DIR/server.crt" \
  -config "$CERT_DIR/san.cnf" 2>/dev/null

echo "✅ 证书已生成:"
echo "   $CERT_DIR/server.crt"
echo "   $CERT_DIR/server.key"
echo ""

echo "=== 证书详情 ==="
openssl x509 -in "$CERT_DIR/server.crt" -noout -subject -dates -ext subjectAltName 2>/dev/null
echo ""

echo "=== 配置完成（未改动任何部署文件、未修改系统信任链）==="
echo ""
echo "下一步："
echo "  - 由 deploy/nginx 反向代理使用该证书（HTTPS 终止），LiveKit 保持内部 ws，不单独配 TLS"
echo "  - 部署网关: bash deploy/nginx/deploy.sh"
echo "  - 自签证书的信任由你自行决定（浏览器手动信任或导入系统，本脚本不代劳）"
echo ""
echo "生产环境用法:"
echo "  DOMAIN=meet.yourcompany.com \\"
echo "  EXTRA_DNS=\"api.meet.yourcompany.com\" \\"
echo "  EXTRA_IPS=\"\$(curl -s ifconfig.me)\" \\"
echo "  ./gen-tls.sh"

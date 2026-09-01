#!/bin/bash
# Generate JWT token for testing
# Usage: bash deploy/livekit/gen-token.sh [user_id] [user_name] [dept_code]
# sh gen-token.sh 0001 张三 dept001
USER_ID="${1:-boss}"
USER_NAME="${2:-Boss}"
DEPT_CODE="${3:-d01}"
SECRET="629c6233-1a76-471b-bd25-b87208762219"
EXP=$(($(date +%s) + 86400))
IAT=$(date +%s)

python3 -c "
import base64, hashlib, hmac, json, time

def b64url(data):
    return base64.urlsafe_b64encode(data).rstrip(b'=').decode()

claims = {
    'user_id': '$USER_ID',
    'user_name': '$USER_NAME',
    'dept_code': '$DEPT_CODE',
    'exp': $EXP,
    'iat': $IAT
}

header = b64url(json.dumps({'alg':'HS256','typ':'JWT'}).encode())
payload = b64url(json.dumps(claims).encode())
sig = b64url(hmac.new('$SECRET'.encode(), f'{header}.{payload}'.encode(), hashlib.sha256).digest())

print(f'{header}.{payload}.{sig}')
"
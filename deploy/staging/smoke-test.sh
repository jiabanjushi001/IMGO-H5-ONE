#!/bin/sh
set -eu

API_URL=${1:?用法: smoke-test.sh https://api-staging.example.com tcp-staging.example.com 19090}
TCP_HOST=${2:?缺少 TCP 域名}
TCP_PORT=${3:-19090}

case "$API_URL" in
  https://*) ;;
  *) echo 'API_URL 必须使用 https://' >&2; exit 1 ;;
esac

health=$(curl --fail --silent --show-error --proto '=https' --tlsv1.2 --max-time 15 "$API_URL/healthz")
case "$health" in
  *'"status":"ok"'*) echo 'HTTP healthz: PASS' ;;
  *) echo "HTTP healthz: FAIL ($health)" >&2; exit 1 ;;
esac

request='{"version":1,"id":"staging-smoke","action":"system.ping","data":{}}'
response=$(
  printf '%s\n' "$request" \
    | openssl s_client -connect "$TCP_HOST:$TCP_PORT" -servername "$TCP_HOST" -verify_return_error -quiet 2>/dev/null \
    | head -n 1
)

case "$response" in
  *'"id":"staging-smoke"'*'"code":0'*'"type":"pong"'*) echo 'TLS TCP system.ping: PASS' ;;
  *) echo "TLS TCP system.ping: FAIL ($response)" >&2; exit 1 ;;
esac

echo 'IMGo 预发布 HTTP/TCP 冒烟测试通过。'

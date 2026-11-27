#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ENV_FILE=${1:-"$SCRIPT_DIR/.env"}
MODE=${2:-actual}

if [ ! -f "$ENV_FILE" ]; then
  echo "预发布环境文件不存在：$ENV_FILE" >&2
  exit 1
fi

set -a
# 文件由部署人员维护并被 systemd 读取，因此采用与 start.sh 相同的 shell 格式。
. "$ENV_FILE"
set +a

errors=0
required_keys='IMGO_ADDR IMGO_TCP_ADDR IMGO_TCP_TLS_CERT IMGO_TCP_TLS_KEY MYSQL_DSN TABLE_PREFIX JWT_KEY CHAT_KEY PUBLIC_DIR BASE_URL QR_BASE_URL H5_URL INVITE_URL'

for key in $required_keys; do
  eval "value=\${$key-}"
  if [ -z "$value" ]; then
    echo "缺少配置：$key" >&2
    errors=$((errors + 1))
  fi
done

if [ "$MODE" = "template" ]; then
  if [ "$errors" -ne 0 ]; then exit 1; fi
  echo '预发布环境模板字段完整。'
  exit 0
fi

for key in MYSQL_DSN JWT_KEY CHAT_KEY BASE_URL QR_BASE_URL H5_URL INVITE_URL IMGO_TCP_TLS_CERT IMGO_TCP_TLS_KEY; do
  eval "value=\${$key-}"
  case "$value" in
    *REPLACE_*|*CHANGE_ME*|*example.com*)
      echo "配置仍含占位值：$key" >&2
      errors=$((errors + 1))
      ;;
  esac
done

case "$IMGO_ADDR" in
  127.0.0.1:18080) ;;
  *) echo 'IMGO_ADDR 必须为 127.0.0.1:18080，不能直接暴露公网或占用正式端口。' >&2; errors=$((errors + 1)) ;;
esac

case "$IMGO_TCP_ADDR" in
  *:19090) ;;
  *) echo 'IMGO_TCP_ADDR 必须使用预发布专用端口 19090。' >&2; errors=$((errors + 1)) ;;
esac

case "$MYSQL_DSN" in
  *staging*) ;;
  *) echo 'MYSQL_DSN 必须明确指向名称包含 staging 的独立数据库/账号。' >&2; errors=$((errors + 1)) ;;
esac

case "$PUBLIC_DIR" in
  *imgo-staging*) ;;
  *) echo 'PUBLIC_DIR 必须使用独立的 imgo-staging 目录。' >&2; errors=$((errors + 1)) ;;
esac

for key in BASE_URL QR_BASE_URL H5_URL INVITE_URL; do
  eval "value=\${$key-}"
  case "$value" in
    https://*) ;;
    *) echo "$key 必须使用 https://" >&2; errors=$((errors + 1)) ;;
  esac
done

if [ "${#JWT_KEY}" -lt 32 ]; then
  echo 'JWT_KEY 至少需要 32 字节随机值。' >&2
  errors=$((errors + 1))
fi

if [ ! -r "$IMGO_TCP_TLS_CERT" ]; then
  echo "TCP TLS 证书不可读：$IMGO_TCP_TLS_CERT" >&2
  errors=$((errors + 1))
fi
if [ ! -r "$IMGO_TCP_TLS_KEY" ]; then
  echo "TCP TLS 私钥不可读：$IMGO_TCP_TLS_KEY" >&2
  errors=$((errors + 1))
fi

if [ "$errors" -ne 0 ]; then
  echo "预发布环境检查失败，共 $errors 项。" >&2
  exit 1
fi

echo '预发布环境检查通过：独立端口、数据库、目录、HTTPS 与 TCP TLS 均已配置。'

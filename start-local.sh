#!/bin/sh
set -eu
cd -- "$(dirname -- "$0")"
if [ ! -f .env ]; then
  echo '请先复制 .env.example 为 .env，并配置数据库和随机 JWT_KEY。' >&2
  exit 1
fi
set -a
# Normalize Windows CRLF in memory; leave the original .env unchanged.
eval "$(sed "s/$(printf '\r')$//" ./.env)"
set +a
if [ ! -x bin/imgo ]; then
  mkdir -p bin
  go build -trimpath -o bin/imgo ./cmd/imgo
fi
exec ./bin/imgo

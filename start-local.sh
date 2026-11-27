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
mkdir -p bin
# Local development must always rebuild: otherwise an existing binary can
# silently run stale handlers after Go sources or schema migrations change.
go build -trimpath -o bin/imgo ./cmd/imgo
exec ./bin/imgo

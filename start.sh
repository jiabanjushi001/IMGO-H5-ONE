#!/bin/sh
set -eu
cd -- "$(dirname -- "$0")"
if [ ! -f .env ]; then
 echo '请先复制 .env.example 为 .env 并配置数据库、密钥和域名。' >&2
 exit 1
fi
if [ ! -f bin/imgo ]; then
 echo '缺少 bin/imgo，请重新解压完整的 Linux 服务器包。' >&2
 exit 1
fi
if [ ! -x bin/imgo ]; then
 chmod 755 bin/imgo 2>/dev/null || true
fi
if [ ! -x bin/imgo ]; then
 echo 'bin/imgo 没有执行权限，请运行：chmod 755 bin/imgo' >&2
 exit 1
fi
set -a
# Normalize Windows CRLF in memory; leave the original .env unchanged.
eval "$(sed "s/$(printf '\r')$//" ./.env)"
set +a
exec ./bin/imgo "$@"

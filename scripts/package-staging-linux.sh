#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
ARCHIVE=$("$ROOT/scripts/package-linux.sh")

for required in \
  'deploy/staging/.env.example' \
  'deploy/staging/imgo-staging.service' \
  'deploy/staging/api-staging.nginx.conf' \
  'deploy/staging/check-env.sh' \
  'deploy/staging/smoke-test.sh' \
  'deploy/staging/README.md'
do
  if ! tar -tzf "$ARCHIVE" | grep -q "/$required$"; then
    echo "预发布发布包缺少：$required" >&2
    exit 1
  fi
done

echo "$ARCHIVE"

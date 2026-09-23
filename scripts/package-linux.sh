#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
OUT="$ROOT/dist/Imgo-linux-amd64"
ARCHIVE="$ROOT/dist/Imgo-linux-amd64.tar.gz"

rm -rf "$OUT" "$ARCHIVE" "$ARCHIVE.sha256"
mkdir -p "$OUT/bin" "$OUT/docs/reports"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o "$OUT/bin/imgo" "$ROOT/cmd/imgo"
chmod 755 "$OUT/bin/imgo"

cp "$ROOT/.env.example" "$ROOT/README.md" "$ROOT/clear.sql" "$ROOT/start.sh" "$ROOT/run.sh" "$OUT/"
cp -R "$ROOT/data" "$ROOT/deploy" "$OUT/"
for file in CORS.md DEPLOY-LINUX.md LOAD-TEST.md MIGRATION.md ORIGINAL_LICENSE.txt PROVIDERS.md TEST_REPORT.md legacy-routes.json static-assets.json; do
  cp "$ROOT/docs/$file" "$OUT/docs/$file"
done
cp "$ROOT/docs/reports/loadtest-2026-09-22.md" "$OUT/docs/reports/"
mkdir -p "$OUT/public"
rsync -a --exclude='.DS_Store' --exclude='storage/' "$ROOT/public/" "$OUT/public/"
find "$OUT" -name '.DS_Store' -o -name '._*' | xargs rm -f 2>/dev/null || true

(cd "$ROOT/dist" && COPYFILE_DISABLE=1 tar -czf "$(basename "$ARCHIVE")" "$(basename "$OUT")")
if command -v shasum >/dev/null 2>&1; then
  (cd "$ROOT/dist" && shasum -a 256 "$(basename "$ARCHIVE")" > "$(basename "$ARCHIVE").sha256")
else
  (cd "$ROOT/dist" && sha256sum "$(basename "$ARCHIVE")" > "$(basename "$ARCHIVE").sha256")
fi

echo "$ARCHIVE"

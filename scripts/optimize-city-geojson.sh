#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$ROOT_DIR/web/admin/public/maps/china-cities.geojson"
TMP="$ROOT_DIR/web/admin/public/maps/china-cities.optimized.geojson"
TARGETS=(
  "$ROOT_DIR/web/admin/public/maps/china-cities.geojson"
  "$ROOT_DIR/web/dist/maps/china-cities.geojson"
  "$ROOT_DIR/internal/admin/http/static/maps/china-cities.geojson"
)

if [[ ! -f "$SRC" ]]; then
  echo "source geojson not found: $SRC" >&2
  exit 1
fi

pushd "$ROOT_DIR/web/admin/public/maps" >/dev/null
npx --yes mapshaper china-cities.geojson \
  -filter-fields adcode,city,province \
  -simplify visvalingam weighted 8% keep-shapes \
  -o format=geojson precision=0.0001 china-cities.optimized.geojson
popd >/dev/null

for f in "${TARGETS[@]}"; do
  mkdir -p "$(dirname "$f")"
  cp "$TMP" "$f"

  # gzip sidecar
  gzip -n -9 -c "$f" > "$f.gz"

  # brotli sidecar (node built-in zlib)
  node -e "const fs=require('fs');const z=require('zlib');const p=process.argv[1];const b=fs.readFileSync(p);const out=z.brotliCompressSync(b,{params:{[z.constants.BROTLI_PARAM_QUALITY]:11}});fs.writeFileSync(p+'.br',out);" "$f"
done

rm -f "$TMP"
echo "optimized and synced china-cities.geojson (+ .gz/.br sidecars)"
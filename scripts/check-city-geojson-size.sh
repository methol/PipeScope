#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
FILE="$ROOT_DIR/web/admin/public/maps/china-cities.geojson"
GZ_FILE="$FILE.gz"
BR_FILE="$FILE.br"
MAX_RAW=$((6 * 1024 * 1024))   # 6 MiB
MAX_GZIP=$((2 * 1024 * 1024))  # 2 MiB

if [[ ! -f "$FILE" ]]; then
  echo "missing file: $FILE" >&2
  exit 1
fi

# Ensure source sidecars are reproducibly generated in standard toolchain.
"$ROOT_DIR/scripts/generate-geojson-sidecars.sh" "$FILE" >/dev/null

RAW=$(wc -c < "$FILE" | tr -d ' ')
GZIP=$(gzip -n -9 -c "$FILE" | wc -c | tr -d ' ')
SIDE_GZIP=$(wc -c < "$GZ_FILE" | tr -d ' ')

EXPECTED_BR_SHA=$(node -e "const fs=require('fs');const z=require('zlib');const p=process.argv[1];const b=fs.readFileSync(p);process.stdout.write(require('crypto').createHash('sha256').update(z.brotliCompressSync(b,{params:{[z.constants.BROTLI_PARAM_QUALITY]:11}})).digest('hex'));" "$FILE")
SIDE_BR_SHA=$(shasum -a 256 "$BR_FILE" | awk '{print $1}')

echo "china-cities.geojson raw=${RAW} bytes, gzip=${GZIP} bytes, sidecar_gzip=${SIDE_GZIP} bytes"

if (( RAW > MAX_RAW )); then
  echo "ERROR: raw size exceeds limit ${MAX_RAW} bytes" >&2
  exit 1
fi
if (( GZIP > MAX_GZIP )); then
  echo "ERROR: gzip size exceeds limit ${MAX_GZIP} bytes" >&2
  exit 1
fi
if (( SIDE_GZIP != GZIP )); then
  echo "ERROR: gzip sidecar size mismatch (expected ${GZIP}, got ${SIDE_GZIP})" >&2
  exit 1
fi
if [[ "$SIDE_BR_SHA" != "$EXPECTED_BR_SHA" ]]; then
  echo "ERROR: brotli sidecar mismatch with deterministic compression" >&2
  exit 1
fi

echo "size guard passed"
#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
FILE="$ROOT_DIR/web/admin/public/maps/china-cities.geojson"
MAX_RAW=$((6 * 1024 * 1024))   # 6 MiB
MAX_GZIP=$((2 * 1024 * 1024))  # 2 MiB

if [[ ! -f "$FILE" ]]; then
  echo "missing file: $FILE" >&2
  exit 1
fi

RAW=$(wc -c < "$FILE" | tr -d ' ')
GZIP=$(gzip -n -9 -c "$FILE" | wc -c | tr -d ' ')

echo "china-cities.geojson raw=${RAW} bytes, gzip=${GZIP} bytes"

if (( RAW > MAX_RAW )); then
  echo "ERROR: raw size exceeds limit ${MAX_RAW} bytes" >&2
  exit 1
fi
if (( GZIP > MAX_GZIP )); then
  echo "ERROR: gzip size exceeds limit ${MAX_GZIP} bytes" >&2
  exit 1
fi

echo "size guard passed"
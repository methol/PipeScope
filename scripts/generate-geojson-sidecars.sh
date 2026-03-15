#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <geojson-file>" >&2
  exit 1
fi

FILE="$1"
if [[ ! -f "$FILE" ]]; then
  echo "missing file: $FILE" >&2
  exit 1
fi

# deterministic gzip
gzip -n -9 -c "$FILE" > "$FILE.gz"

# deterministic brotli (node zlib, quality 11)
node -e "const fs=require('fs');const z=require('zlib');const p=process.argv[1];const b=fs.readFileSync(p);const out=z.brotliCompressSync(b,{params:{[z.constants.BROTLI_PARAM_QUALITY]:11}});fs.writeFileSync(p+'.br',out);" "$FILE"

echo "generated sidecars: $FILE.{gz,br}"

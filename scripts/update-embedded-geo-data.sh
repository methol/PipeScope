#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "$0")/.." && pwd)"
WORK_DIR="${ROOT_DIR}/data"
ASSET_DIR="${ROOT_DIR}/internal/embeddeddata/assets"
AREACITY_TAG="${AREACITY_TAG:-2023.240319.250114}"
IP2REGION_URL="${IP2REGION_URL:-https://raw.githubusercontent.com/lionsoul2014/ip2region/master/data/ip2region_v4.xdb}"
AREACITY_URL="${AREACITY_URL:-https://github.com/xiangyuecn/AreaCity-JsSpider-StatsGov/releases/download/${AREACITY_TAG}/ok_geo.csv.7z}"

mkdir -p "${WORK_DIR}" "${ASSET_DIR}"

echo "[1/4] 下载 ip2region_v4.xdb"
curl -L --fail -o "${WORK_DIR}/ip2region_v4.xdb" "${IP2REGION_URL}"

echo "[2/4] 下载 ok_geo.csv.7z"
curl -L --fail -o "${WORK_DIR}/ok_geo.csv.7z" "${AREACITY_URL}"

echo "[3/4] 解压 ok_geo.csv.7z"
if command -v 7z >/dev/null 2>&1; then
  7z x -aoa -o"${WORK_DIR}" "${WORK_DIR}/ok_geo.csv.7z" >/dev/null
elif command -v 7zz >/dev/null 2>&1; then
  7zz x -aoa -o"${WORK_DIR}" "${WORK_DIR}/ok_geo.csv.7z" >/dev/null
elif command -v bsdtar >/dev/null 2>&1; then
  LC_ALL=en_US.UTF-8 bsdtar -xf "${WORK_DIR}/ok_geo.csv.7z" -C "${WORK_DIR}"
else
  echo "未找到可用解压工具(7z/7zz/bsdtar)，请先安装其中之一。" >&2
  exit 1
fi

echo "[4/4] 生成 embedded 资产"
go run ./cmd/gen-embedded-geo-data -xdb "${WORK_DIR}/ip2region_v4.xdb" -areacity "${WORK_DIR}/ok_geo.csv" -out "${ASSET_DIR}" -areacity-tag "${AREACITY_TAG}"

echo "完成：已更新 ${ASSET_DIR} 下的 embedded 资产"

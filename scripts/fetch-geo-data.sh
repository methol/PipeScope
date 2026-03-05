#!/usr/bin/env bash
set -euo pipefail

DATA_DIR="${1:-data}"
AREACITY_TAG="${AREACITY_TAG:-2023.240319.250114}"
IP2REGION_URL="${IP2REGION_URL:-https://raw.githubusercontent.com/lionsoul2014/ip2region/master/data/ip2region_v4.xdb}"
AREACITY_URL="${AREACITY_URL:-https://github.com/xiangyuecn/AreaCity-JsSpider-StatsGov/releases/download/${AREACITY_TAG}/ok_geo.csv.7z}"

mkdir -p "${DATA_DIR}"

echo "[1/3] 下载 ip2region_v4.xdb"
curl -L --fail -o "${DATA_DIR}/ip2region_v4.xdb" "${IP2REGION_URL}"

echo "[2/3] 下载 ok_geo.csv.7z"
curl -L --fail -o "${DATA_DIR}/ok_geo.csv.7z" "${AREACITY_URL}"

echo "[3/3] 解压 ok_geo.csv.7z"
if command -v 7z >/dev/null 2>&1; then
  7z x -aoa -o"${DATA_DIR}" "${DATA_DIR}/ok_geo.csv.7z" >/dev/null
elif command -v 7zz >/dev/null 2>&1; then
  7zz x -aoa -o"${DATA_DIR}" "${DATA_DIR}/ok_geo.csv.7z" >/dev/null
elif command -v bsdtar >/dev/null 2>&1; then
  LC_ALL=en_US.UTF-8 bsdtar -xf "${DATA_DIR}/ok_geo.csv.7z" -C "${DATA_DIR}"
else
  echo "未找到可用解压工具(7z/7zz/bsdtar)，请先安装其中之一。" >&2
  exit 1
fi

test -f "${DATA_DIR}/ok_geo.csv"
echo "完成：${DATA_DIR}/ip2region_v4.xdb 和 ${DATA_DIR}/ok_geo.csv"

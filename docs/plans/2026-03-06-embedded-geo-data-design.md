# PipeScope 内置 Geo/IP 数据设计

## 目标

将 `ip2region` 与地理位置匹配数据直接内置进 PipeScope 的 GitHub Release 二进制中，使用户下载单个可执行文件后即可运行，不再依赖额外数据下载或外部 AreaCity API。

## 约束

- Release 二进制应保持单文件分发。
- 默认只支持 IPv4 内置 IP 数据。
- 运行时不依赖 `data/ip2region_v4.xdb`、`data/ok_geo.csv` 或 `areacity_api_base_url`。
- 后续数据更新通过人工执行脚本完成，而不是运行时在线更新。

## 方案

### 1. 内置资产

新增 `internal/embeddeddata/assets/`：

- `ip2region_v4.xdb.gz`：gzip 压缩后的 IPv4 xdb。
- `areacity_seed.csv.gz`：从 `ok_geo.csv` 预处理得到的精简 CSV，仅保留运行时需要的字段。
- `manifest.json`：记录数据版本、来源、哈希、生成时间等元数据。

### 2. 运行时加载

- 启动时将内置 `ip2region_v4.xdb.gz` 解压到应用缓存目录，再用现有 `ip2region` 查询器初始化。
- 启动时检查 SQLite 中记录的 AreaCity seed 版本；若未导入或版本不一致，则从内置 `areacity_seed.csv.gz` 解压并重新导入 `dim_adcode`。
- 数据版本记录在 SQLite 的 `app_meta` 表。

### 3. 配置策略

- 保留现有配置结构中的 geo/IP 相关字段以减少破坏性修改。
- 主运行路径不再读取外部 geo/IP 路径或 HTTP API 配置。
- `ip2region_cache_policy` 与 `ip2region_searchers` 仍可用于控制查询行为。

### 4. 数据更新

新增 `scripts/update-embedded-geo-data.sh`：

1. 下载原始 `ip2region_v4.xdb` 与 `ok_geo.csv`
2. 调用生成器产出内置资产
3. 更新 `internal/embeddeddata/assets/manifest.json`

## 风险与取舍

- 二进制体积会显著增加，但这是预期行为。
- 首次启动会有一次内置 AreaCity seed 导入开销，但后续依赖版本检查避免重复导入。
- 为降低实现复杂度，AreaCity seed 使用精简 CSV 而非内置 SQLite 文件，复用现有 importer 逻辑。

## 验证标准

- 在无 `data/` 目录、无外部 AreaCity API 的环境中可直接启动。
- Geo enrich 仍能写入 `province/city/adcode/lat/lng`。
- 构建出的二进制包含内置数据，体积明显大于当前版本。

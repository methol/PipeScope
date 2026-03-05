# PipeScope Runbook

## 1. 准备

- Go 1.24+
- Node.js 20+
- 可写磁盘路径用于 SQLite

## 2. 本地构建

```bash
make build-web sync-web
go build ./cmd/pipescope
```

## 3. 下载真实 geo 数据

```bash
make fetch-geo-data
```

默认会下载并生成：
- `data/ip2region_v4.xdb`（来源：`lionsoul2014/ip2region`）
- `data/ok_geo.csv`（来源：`xiangyuecn/AreaCity-JsSpider-StatsGov`）

## 4. 启动 AreaCity 官方高性能查询服务（推荐）

- 使用项目：`https://github.com/xiangyuecn/AreaCity-Query-Geometry`
- 启动后默认 API 地址：`http://127.0.0.1:9527`
- 建议在该服务内使用：
  - `Init_StoreInMemory`（更高吞吐，内存更高）
  - 或 `Init_StoreInWkbsFile`（内存低，IO 开销更高）

PipeScope 配置项：

```yaml
data:
  areacity_api_base_url: "http://127.0.0.1:9527"
  areacity_api_instance: 0
```

`areacity_api_base_url` 配置后，PipeScope 将优先走 AreaCity 官方 HTTP API 查询；未配置时才回退本地 CSV+SQLite 匹配。
`assets/config.example.yaml` 默认已开启该地址，因此未启动 AreaCity API 时 PipeScope 启动会报错。

## 5. 启动

```bash
go run ./cmd/pipescope -config assets/config.example.yaml
```

默认管理端地址：`http://127.0.0.1:9100`。

## 6. 健康检查

```bash
curl http://127.0.0.1:9100/api/health
```

期望返回：`{"status":"ok"}`。

## 7. 常用排查

1. 管理页无数据
- 检查代理规则 `listen/forward` 是否可达。
- 检查 SQLite 文件是否创建成功。
- 查询 `conn_events` 是否有写入。

2. 地图没有地理点位
- 若启用了 `areacity_api_base_url`，先检查 AreaCity HTTP 服务是否可达。
- 若未启用 HTTP API，确认 `areacity_csv_path` 文件存在并且导入成功。
- 若缺少 geo 数据，地图会回退为 `unknown` 聚合。

3. 前端资源 404
- 执行 `make build-web sync-web` 重新同步内嵌静态资源。

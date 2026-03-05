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

## 3. 启动

```bash
go run ./cmd/pipescope -config assets/config.example.yaml
```

默认管理端地址：`http://127.0.0.1:9100`。

## 4. 健康检查

```bash
curl http://127.0.0.1:9100/api/health
```

期望返回：`{"status":"ok"}`。

## 5. 常用排查

1. 管理页无数据
- 检查代理规则 `listen/forward` 是否可达。
- 检查 SQLite 文件是否创建成功。
- 查询 `conn_events` 是否有写入。

2. 地图没有地理点位
- 当前若缺少 ip2region/AreaCity 数据，地图会回退为 `unknown` 聚合。
- 若需城市坐标，请准备可用的 `ip2region_xdb_path` 与 `areacity_csv_path`。

3. 前端资源 404
- 执行 `make build-web sync-web` 重新同步内嵌静态资源。

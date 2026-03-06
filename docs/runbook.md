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

## 3. 内置 geo/IP 数据

PipeScope 当前 release / 本地构建默认内置：
- IPv4 `ip2region` 数据
- AreaCity 精简省市匹配 seed

启动时会把内置 IP 库与 AreaCity seed 解压到系统用户缓存目录（例如 macOS 上的 `~/Library/Caches/pipescope/embeddeddata`），因此该目录需要可写。

运行时无需额外下载：
- `data/ip2region_v4.xdb`
- `data/ok_geo.csv`
- AreaCity HTTP API

如需更新内置数据，可执行：

```bash
make update-embedded-geo-data
```

## 4. 启动

```bash
go run ./cmd/pipescope -config assets/config.example.yaml
```

默认管理端地址：`http://127.0.0.1:9100`。

## 5. 健康检查

```bash
curl http://127.0.0.1:9100/api/health
```

期望返回：`{"status":"ok"}`。

## 6. 常用排查

1. 管理页无数据
- 检查代理规则 `listen/forward` 是否可达。
- 检查 SQLite 文件是否创建成功。
- 查询 `conn_events` 是否有写入。

2. 地图没有地理点位
- 确认程序首次启动时有权限写入用户缓存目录。
- 检查 SQLite 中 `dim_adcode` 是否已导入数据。
- 若缺少 geo 数据，地图会回退为 `unknown` 聚合。

3. 前端资源 404
- 执行 `make build-web sync-web` 重新同步内嵌静态资源。

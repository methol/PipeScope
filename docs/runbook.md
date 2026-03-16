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

4. Geo Policy 配置不生效
- 检查 `geo_policy` 是否正确缩进在 `proxy_rules[*]` 下（YAML 层级）
- 检查 `mode` 值是否为 `allow` 或 `deny`（小写）
- 白名单模式需设置 `require_allow_hit: true`
- 国家码必须大写（如 `CN`，而非 `cn`）
- 查看启动日志是否有配置校验错误

5. Geo 拦截记录查询
```sql
-- 查询被拦截的连接
SELECT rule_id, src_addr, blocked_reason, province, city
FROM conn_events
WHERE blocked_reason != ''
ORDER BY start_ts DESC
LIMIT 50;

-- 统计拦截原因分布
SELECT blocked_reason, COUNT(*) as cnt
FROM conn_events
WHERE blocked_reason != ''
GROUP BY blocked_reason;
```

6. 常见 Geo 匹配问题
- **国外 IP 无 adcode**：国外 IP 只有国家码，省份/城市/adcode 为空，仅能通过 `country` 匹配
- **同名城市歧义**：如"吉林"省市同名，建议使用 `adcodes` 精确匹配
- **IP 库覆盖不完整**：部分内网或特殊 IP 无法解析，建议在日志中观察

# PipeScope 设计文档（MVP + 管理网页）

## 1. 目标与范围

PipeScope 是一个轻量化 L4 透明 TCP 代理网关：
- 按配置做纯 TCP 转发，不解析 payload。
- 在不阻塞转发热路径前提下记录连接级元数据。
- 异步落库 SQLite，并补全 geo 信息。
- 提供内置管理网页用于可视化分析（非 CLI-only）。

本设计覆盖 Milestone A-D（加上管理端页面）。

## 2. 已确认决策

1. 部署形态：单进程一体化（Go 进程内置 API + 静态前端）。
2. 管理端鉴权：MVP 不做鉴权。
3. Admin 监听：`host/port` 可配置。
4. 管理首页：地图优先。
5. 地图层级：全国市级聚合着色，点击省后下钻区县。
6. 地图方案：ECharts + 内置 GeoJSON（不依赖在线底图）。
7. 页面范围：地图页 + 规则页 + 连接明细页。
8. 数据刷新：前端轮询（默认 5 秒）。
9. 前端技术：Vue。

## 3. 非目标

1. 不做 payload 分析、TLS 解密、HTTP 解析。
2. 不做“全协议代理套件”。
3. 不强依赖外部地图 API。
4. 不做复杂 RBAC。

## 4. 总体架构

分为四层，确保热路径最短：

1. Gateway Core（热路径）
- Listener 接收连接
- 按 rule 建立 upstream
- 双向拷贝 + 字节计数
- 连接结束后投递 `ConnSession` 到异步队列

2. Async Pipeline（冷路径）
- writer 批量消费队列
- 解析 ip2region
- province/city 归一化并匹配 adcode 坐标
- 批量写 SQLite（WAL + 事务）

3. Admin Read Layer
- 只读查询 API
- 支持地图聚合、规则统计、明细筛选

4. Frontend（内置静态资源）
- Vue + ECharts
- 默认地图页，支持省内下钻
- 轮询 API 更新

## 5. 建议目录结构

- `cmd/pipescope/`
- `internal/tcpproxy/`
- `internal/gateway/rule/`
- `internal/gateway/proxy/`
- `internal/gateway/session/`
- `internal/geo/ip2region/`
- `internal/geo/areacity/`
- `internal/geo/normalize/`
- `internal/store/sqlite/`
- `internal/admin/http/`
- `internal/admin/service/`
- `web/admin/`（Vue 源码）
- `web/dist/`（构建产物）
- `assets/`

边界约束：
- `gateway` 不直接写库。
- `admin` 只读，不影响热路径。
- geo 补全放在 writer/查询服务，不放在 copy 回路。

## 6. 配置模型（MVP）

在原规格基础上补充：

```yaml
data:
  sqlite_path: "./data/pipescope.db"
  ip2region_xdb_path: "./data/ip2region_v4.xdb"
  areacity_csv_path: "./data/ok_geo.csv"

proxy_rules:
  - id: "r1"
    listen: "0.0.0.0:10001"
    forward: "127.0.0.1:20001"

writer:
  queue_size: 200000
  batch_size: 500
  flush_interval_ms: 200
  full_queue_policy: "drop" # drop|block|sample
  sample_rate: 0.1

timeouts:
  dial_ms: 3000
  idle_ms: 600000

admin:
  enable: true
  host: "127.0.0.1"
  port: 9100
  poll_hint_ms: 5000
```

## 7. 数据模型

### 7.1 事实表 `conn_events`

字段沿用规格：
- 基础：`ts_start/ts_end/duration_ms/rule_id/listen_port/src_ip/src_port/dst_ip/dst_port/up_bytes/down_bytes/total_bytes/status`
- geo：`country/province/city/isp/adcode/lng/lat`

索引：
- `(ts_start)`
- `(rule_id, ts_start)`
- `(adcode, ts_start)`
- `(province, city, ts_start)`

### 7.2 维表 `dim_adcode`

字段沿用规格：
- `adcode,name,level,parent_adcode,lng,lat`
- 后续可扩边界字段

## 8. 关键数据流

1. Accept 连接，初始化会话。
2. Dial upstream（带 `dial_ms`）。
3. 双向 copy + up/down 计数。
4. 连接结束，归类 `status`：`ok|dial_fail|timeout|io_err`。
5. 投递会话到队列（根据 `full_queue_policy` 处理满队列）。
6. writer 批处理：geo 补全 + 批量写库。
7. admin API 查询聚合结果供前端渲染。

## 9. 管理端 API 契约

1. `GET /api/health`
2. `GET /api/overview?window=5m|15m|60m`
3. `GET /api/map/china?window=15m&metric=conn|bytes`
4. `GET /api/map/province/{adcode}?window=15m&metric=conn|bytes`
5. `GET /api/rules?window=15m`
6. `GET /api/sessions?from=...&to=...&rule_id=...&status=...&province=...&city=...&page=1&page_size=50`
7. `GET /api/geo/meta`

原则：
- 全部只读。
- 默认限制时间窗口和分页，避免重查询。

## 10. 前端页面设计（MVP）

1. 地图页（默认）
- 全国市级着色（按连接数或流量）
- 点击省份后显示区县级着色
- 支持时间窗口切换（5m/15m/60m）

2. 规则页
- 按 rule 展示连接数、流量、错误率、最近活跃

3. 连接明细页
- 按时间/rule/status/省市筛选
- 分页查看连接记录

## 11. 测试与验收

1. 单元测试
- 会话状态归类
- 名称归一化与 adcode 匹配
- 查询聚合与分页

2. 集成测试
- 多规则转发连通性
- 异常断连与 status 落库
- 队列满策略
- admin API 返回结构

3. 性能验收
- 并发压测下稳定转发
- writer 异步不明显拖慢吞吐
- 管理端轮询下查询延迟可接受

## 12. 风险与缓解

1. 区域匹配不稳定：归一化规则 + 匹配失败允许空 adcode。
2. SQLite 写压力：WAL + 批量事务 + 索引控制。
3. 查询慢：先靠时间窗口和索引；二期再考虑预聚合。

## 13. 里程碑映射

1. Milestone A：透明转发主链路。
2. Milestone B：ConnSession + 异步落库。
3. Milestone C：ip2region + dim_adcode 导入匹配。
4. Milestone D：CLI 导出 + Admin API + Vue 可视化页面。
5. Milestone E（二期）：封禁规则与拦截落库。

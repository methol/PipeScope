# PipeScope

PipeScope 是一个面向 TCP 透明转发场景的轻量观测系统，包含：

- 多规则 TCP 转发网关
- 异步批量 SQLite 落库
- 管理端 API 与内置 Vue 管理页面

## 当前使用方式概览

- 主入口：`cmd/pipescope/main.go`
- 配置加载：`internal/config`
- TCP 转发与会话采集：`internal/gateway`
- SQLite 写入与地理信息 enrich：`internal/store/sqlite`
- 管理端查询 API / 静态资源：`internal/admin`
- 端到端验证：`tests/e2e/pipescope_e2e_test.go`

运行时，PipeScope 会监听配置里的 `proxy_rules[*].listen`，将 TCP 流量转发到 `forward`，并把连接事件写入 SQLite，再通过 `http://127.0.0.1:9100` 暴露管理 API 与内置管理页面。

## 快速开始 / 使用教程

### 1. 环境要求

- Go 1.24+
- Node.js 20+（仅前端开发 / 重建内嵌资源时需要）
- 可写磁盘路径用于 SQLite
- 可写用户缓存目录用于首次释放内置 geo 数据

### 2. 安装 / 构建

#### 方式一：直接运行源码

```bash
go run ./cmd/pipescope -config assets/config.example.yaml
```

#### 方式二：本地构建二进制

```bash
make build-web sync-web
go build -o ./bin/pipescope ./cmd/pipescope
./bin/pipescope -config assets/config.example.yaml
```

#### 方式三：安装到 GOPATH/bin

```bash
go install ./cmd/pipescope
pipescope -config assets/config.example.yaml
```

> 首次启动时会自动释放内置 IPv4 `ip2region` 数据和 AreaCity seed；默认不再依赖外部 geo 文件或 AreaCity HTTP API。

### 3. 最小运行示例

先准备一个最小可转发的目标服务，例如本机已有一个 TCP echo 服务监听 `127.0.0.1:10002`，然后使用示例配置启动 PipeScope：

```bash
go run ./cmd/pipescope -config assets/config.example.yaml
```

示例配置中的核心规则是：

```yaml
proxy_rules:
  - id: "demo-rule"
    listen: "0.0.0.0:10001"
    forward: "127.0.0.1:10002"
```

此时访问路径为：

- 业务流量入口：`127.0.0.1:10001`
- 管理端：`http://127.0.0.1:9100`
- 健康检查：`http://127.0.0.1:9100/api/health`

验证服务：

```bash
curl http://127.0.0.1:9100/api/health
```

期望返回：

```json
{"status":"ok"}
```

### 4. 常用参数与配置说明

#### CLI flags

- `-config`：配置文件路径，默认 `assets/config.example.yaml`

可查看帮助：

```bash
go run ./cmd/pipescope -h
```

#### 配置文件

`assets/config.example.yaml` 的关键项：

- `data.sqlite_path`：SQLite 数据库文件路径
- `data.ip2region_cache_policy`：`noCache|vindex|content`
- `data.ip2region_searchers`：ip2region 查询实例数
- `proxy_rules`：TCP 转发规则列表
- `writer.queue_size`：采集事件队列长度
- `writer.batch_size`：批量写库条数
- `writer.flush_interval_ms`：批量刷盘间隔（毫秒）
- `writer.full_queue_policy`：队列满时策略，`drop|block|sample`
- `writer.sample_rate`：`sample` 策略的采样率
- `timeouts.dial_ms`：后端拨号超时
- `timeouts.idle_ms`：连接空闲超时
- `admin.host` / `admin.port`：管理端监听地址

当前代码还会对缺省配置应用默认值：

- `writer.queue_size=1024`
- `writer.batch_size=200`
- `writer.flush_interval_ms=1000`
- `writer.full_queue_policy=drop`
- `writer.sample_rate=0.1`
- `timeouts.dial_ms=1500`
- `timeouts.idle_ms=60000`
- `admin.host=127.0.0.1`
- `admin.port=9100`

### 5. 典型使用场景

#### 场景一：本机透明记录单条转发规则

```yaml
proxy_rules:
  - id: "ssh-proxy"
    listen: "0.0.0.0:2222"
    forward: "127.0.0.1:22"
```

适合先把 SSH / 自定义 TCP 服务挂在 PipeScope 后面，快速观察连接数、来源地址、流量大小。

#### 场景二：多规则接入多个内部服务

```yaml
proxy_rules:
  - id: "redis-a"
    listen: "0.0.0.0:16379"
    forward: "10.0.0.12:6379"
  - id: "mysql-a"
    listen: "0.0.0.0:13306"
    forward: "10.0.0.20:3306"
```

适合把多个 TCP 服务统一接到一个观测入口，通过管理页按规则查看连接热点。

#### 场景三：高流量场景降低写入压力

```yaml
writer:
  queue_size: 8192
  batch_size: 500
  flush_interval_ms: 1000
  full_queue_policy: "sample"
  sample_rate: 0.2
```

适合在优先保可用性的场景下，以采样方式保留观测能力并限制 SQLite 写入压力。

#### 场景四：Geo 前置拦截（按地域过滤连接）

PipeScope 支持在 TCP 连接建立前进行地理策略检查，可用于：
- 仅允许中国流量，拦截国外访问
- 拦截特定省份或城市
- 仅允许白名单城市访问
- **允许中国但禁止特定省份/城市**（组合模式）

##### Geo Policy 配置字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `require_allow_hit` | bool | 否 | 当未匹配任何规则时的默认行为：`true`=拒绝，`false`=放行（默认） |
| `allow` | []GeoRule | 否 | 允许规则列表（deny 规则后检查） |
| `deny` | []GeoRule | 否 | 拒绝规则列表（优先检查） |
| `GeoRule.country` | string | 是 | ISO 3166-1 alpha-2 国家码（如 `CN`、`US`、`JP`），必须大写 |
| `GeoRule.provinces` | []string | 否 | 省份名称列表，支持中文（如 `["北京", "广东"]`），会自动标准化（去"省"、"市"后缀） |
| `GeoRule.cities` | []string | 否 | 城市名称列表（如 `["深圳", "广州"]`），需配合 `provinces` 使用以避免同名城市歧义 |
| `GeoRule.adcodes` | []string | 否 | 6 位行政区划码列表（最精确，如 `["440300"]`），仅国内 IP 有效 |

##### 匹配逻辑

**匹配顺序**：
1. **deny 规则**：先检查，命中则立即拒绝
2. **allow 规则**：再检查，命中则放行
3. **默认行为**：无任何规则命中时，按 `require_allow_hit` 决定（`true`=拒绝，`false`=放行）

**规则内匹配优先级**：adcode > city+province > province > country

**规则内关系**：`provinces` 与 `cities` 是 AND 关系（需同时匹配）；`provinces`、`cities`、`adcodes` 之间是 OR 关系

**多规则匹配**：同类型规则（allow 或 deny）按配置顺序匹配，首条匹配生效

##### 配置行为表

| 有 allow 规则 | 有 deny 规则 | require_allow_hit | 命中 deny | 命中 allow | 都未命中 | 结果 |
|--------------|-------------|-------------------|-----------|-----------|---------|------|
| ✓ | ✓ | 任意 | ✓ | - | - | **拒绝** |
| ✓ | ✓ | 任意 | ✗ | ✓ | - | **放行** |
| ✓ | ✓ | true | ✗ | ✗ | ✓ | **拒绝** |
| ✓ | ✓ | false | ✗ | ✗ | ✓ | **放行** |
| ✓ | ✗ | true | - | ✓ | - | **放行** |
| ✓ | ✗ | true | - | ✗ | ✓ | **拒绝** |
| ✗ | ✓ | 任意 | ✓ | - | - | **拒绝** |
| ✗ | ✓ | 任意 | ✗ | - | - | **放行** |
| ✗ | ✗ | true | - | - | ✓ | **拒绝**（全部拒绝） |
| ✗ | ✗ | false | - | - | ✓ | **放行**（全部放行） |

##### 典型配置示例

**示例 1：仅允许中国流量（禁止国外访问）**

```yaml
proxy_rules:
  - id: "china-only"
    listen: "0.0.0.0:10001"
    forward: "127.0.0.1:10002"
    geo_policy:
      require_allow_hit: true     # 未匹配规则则拒绝
      allow:
        - country: "CN"           # 仅允许中国
      deny: []
```

**示例 2：禁止特定省份（黑名单模式）**

```yaml
proxy_rules:
  - id: "block-provinces"
    listen: "0.0.0.0:10002"
    forward: "127.0.0.1:10003"
    geo_policy:
      require_allow_hit: false    # 未匹配规则则放行
      allow: []
      deny:
        - country: "CN"
          provinces: ["福建", "广东"]
        - country: "CN"
          adcodes: ["440300"]     # 深圳（精确到行政区划码）
```

**示例 3：仅允许白名单城市（精确到行政区划码）**

```yaml
proxy_rules:
  - id: "whitelist-cities"
    listen: "0.0.0.0:10003"
    forward: "127.0.0.1:10004"
    geo_policy:
      require_allow_hit: true     # 未匹配规则则拒绝
      allow:
        - country: "CN"
          adcodes:
            - "110000"            # 北京
            - "310000"            # 上海
            - "440100"            # 广州
      deny: []
```

**示例 4：允许中国但禁止特定省份/城市（组合模式）**

```yaml
proxy_rules:
  - id: "china-except-some"
    listen: "0.0.0.0:10004"
    forward: "127.0.0.1:10005"
    geo_policy:
      require_allow_hit: true     # 必须命中 allow 规则
      allow:
        - country: "CN"           # 允许中国
      deny:                       # 但排除这些
        - country: "CN"
          provinces: ["福建", "广东"]
        - country: "CN"
          adcodes: ["440300"]     # 深圳（广东省）
```

上述配置效果：
- 福建省流量：命中 deny，**拒绝**
- 广东省深圳：命中 deny，**拒绝**
- 广东省其他城市：命中 allow，未命中 deny，**放行**
- 中国其他省份：命中 allow，未命中 deny，**放行**
- 国外流量：未命中任何规则，require_allow_hit=true，**拒绝**

##### 拦截记录与状态查询

被拦截的连接会记录到 SQLite 的 `conn_events` 表，通过以下字段标识：

| 字段 | 值 | 说明 |
|------|------|------|
| `status` | `blocked` | 连接被拦截（另有 `ok`、`dial_fail`、`timeout`、`io_err`） |
| `blocked_reason` | `geo_denied` | 命中 deny 规则 |
| `blocked_reason` | `geo_not_in_allowlist` | 白名单模式下未命中规则 |

**UI 查看路径：**
- 管理端「实时会话」页面（`http://127.0.0.1:9100`）可查看 `blocked_reason` 列
- 使用 rule_id 下拉筛选特定规则的连接

**SQL 查询示例：**

```sql
-- 查询被 geo 策略拦截的连接
SELECT * FROM conn_events WHERE blocked_reason != '' ORDER BY start_ts DESC LIMIT 100;

-- 统计各拦截原因
SELECT blocked_reason, COUNT(*) FROM conn_events WHERE blocked_reason != '' GROUP BY blocked_reason;
```

##### 常见问题与排错

**Q1: 配置后不生效，所有流量都放行？**

检查：
- `geo_policy` 是否正确缩进在 `proxy_rules[*]` 下
- `require_allow_hit` 是否设置正确：`true` 表示未匹配规则则拒绝
- 国家码是否为大写（如 `CN` 而非 `cn`）
- 是否有正确的 `allow` 规则

**Q2: 部分国内 IP 被错误拦截？**

可能原因：
- IP 库数据不完整，某些 IP 无法解析到省份/城市
- adcode 匹配仅对国内 IP 有效，国外 IP 无 adcode 字段
- 同名城市歧义（如"吉林"省市同名），建议使用 adcode 精确匹配

**Q3: 如何验证配置是否正确？**

启动时会自动校验配置，错误配置会输出警告。也可手动检查：

```bash
# 查看启动日志是否有配置错误
./bin/pipescope -config assets/config.example.yaml 2>&1 | grep -i "geo\|valid"
```

**Q4: 国外 IP 的省份/城市字段为空，如何匹配？**

国外 IP 通常只有国家码，省份/城市/adcode 字段为空。匹配时：
- 仅配置 `country` 字段即可匹配国外 IP
- 如需精确匹配国外地区，建议使用其他 IP 库或外部服务

**Q5: 规则顺序有影响吗？**

有。同类型规则（allow 或 deny）按配置顺序匹配，首条匹配生效。建议：
- 精确规则在前（如 adcode）
- 宽泛规则在后（如仅 country）

**Q6: 如何实现"允许中国但禁止某省"？**

使用组合模式：
```yaml
geo_policy:
  require_allow_hit: true
  allow:
    - country: "CN"
  deny:
    - country: "CN"
      provinces: ["福建"]
```
deny 规则优先检查，福建流量会被拒绝；其他中国流量命中 allow 规则放行；国外流量未命中任何规则，require_allow_hit=true 则拒绝。

### 6. 排错 / FAQ

#### 1) 启动时报配置或文件错误怎么办？

先检查：

```bash
go run ./cmd/pipescope -config assets/config.example.yaml
```

以及：

- `-config` 路径是否正确
- `data.sqlite_path` 的父目录是否可写
- 是否仍在使用已废弃的外部 geo 配置字段（如 `ip2region_xdb_path`、`areacity_api_base_url`）

#### 2) 管理页能打开，但没有会话数据？

优先排查：

- `proxy_rules.listen` 是否真的有流量进来
- `forward` 目标是否可达
- SQLite 文件是否已创建
- `conn_events` 表是否有数据

#### 3) 地图上没有地理点位？

可能原因：

- 首次启动时无权限写用户缓存目录
- 内置 AreaCity seed 未成功导入，导致 `dim_adcode` 为空
- 某些地址解析失败时会落到 `unknown` 聚合

#### 4) 为什么前端静态资源 404？

如果你改过 `web/admin`，需要重新同步内嵌资源：

```bash
make build-web sync-web
```

#### 5) 如何重新生成中国县区底图 GeoJSON？

地图边界来自 `data/ok_geo.csv`（`deep=2`，排除港澳台县区）。
可用下面命令重新生成：

```bash
go run ./cmd/gen-china-counties-geojson \
  -input data/ok_geo.csv \
  -output web/admin/public/maps/china-counties.geojson \
  -simplify-epsilon 0.0012 \
  -precision 5
```

参数说明：

- `-simplify-epsilon`：离线简化阈值（度），值越大体积越小、边界越粗
- `-precision`：坐标保留小数位数

#### 6) 如何定位日志？

当前程序主要输出到标准错误。建议直接前台运行观察：

```bash
go run ./cmd/pipescope -config assets/config.example.yaml
```

如果要持久化日志，可自行用 shell 重定向：

```bash
./bin/pipescope -config assets/config.example.yaml >pipescope.out.log 2>pipescope.err.log
```

## 测试

```bash
go test ./...
cd web/admin && npm test && npm run build
```

## 目录结构

- `cmd/pipescope`: 主程序与编排
- `cmd/gen-embedded-geo-data`: 内置 geo 资产生成器
- `internal/gateway`: 透明转发与会话事件
- `internal/store/sqlite`: schema、writer、geo enrichment
- `internal/admin`: 查询服务、HTTP API、静态资源托管
- `web/admin`: Vue + ECharts 管理端源码
- `tests/e2e`: 端到端测试
- `docs/runbook.md`: 运行与排障手册

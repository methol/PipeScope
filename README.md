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

### 6. 排错 / FAQ

#### 1) 启动时报配置或文件错误怎么办？

先检查：

```bash
go run ./cmd/pipescope -config assets/config.example.yaml -h
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

#### 5) 如何定位日志？

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

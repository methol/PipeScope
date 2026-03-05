# PipeScope

PipeScope 是一个面向 TCP 透明转发场景的轻量观测系统，包含：

- 多规则 TCP 转发网关
- 异步批量 SQLite 落库
- 管理端 API 与内置 Vue 管理页面

## 快速开始

1. 下载真实 geo 数据库（ip2region + AreaCity）

```bash
make fetch-geo-data
```

2. 启动 AreaCity 官方高性能查询服务（推荐）

- 项目：`xiangyuecn/AreaCity-Query-Geometry`
- 启动后提供：`http://127.0.0.1:9527`
- 在配置中填写：`data.areacity_api_base_url`

3. 构建前端并同步内嵌资源

```bash
make build-web sync-web
```

4. 启动服务

```bash
go run ./cmd/pipescope -config assets/config.example.yaml
```

5. 打开管理页面

- `http://127.0.0.1:9100`

## 测试

```bash
go test ./... -v
cd web/admin && npm test && npm run build
```

## 目录结构

- `cmd/pipescope`: 主程序与编排
- `internal/gateway`: 透明转发与会话事件
- `internal/store/sqlite`: schema、writer、geo enrichment
- `internal/admin`: 查询服务、HTTP API、静态资源托管
- `web/admin`: Vue + ECharts 管理端源码
- `tests/e2e`: 端到端测试

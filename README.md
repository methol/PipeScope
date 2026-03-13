# PipeScope

PipeScope 是一个面向 TCP 透明转发场景的轻量观测系统，包含：

- 多规则 TCP 转发网关
- 异步批量 SQLite 落库
- 管理端 API 与内置 Vue 管理页面

## 快速开始

1. 构建前端并同步内嵌资源

```bash
make build-web sync-web
```

2. 启动服务

```bash
go run ./cmd/pipescope -config assets/config.example.yaml
```

PipeScope 会自动使用二进制内置的 IPv4 `ip2region` 数据与内置 AreaCity seed，不需要额外下载 geo 数据，也不需要启动 AreaCity API。

3. 打开管理页面

- `http://127.0.0.1:9100`

## 测试

```bash
go test ./... -v
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

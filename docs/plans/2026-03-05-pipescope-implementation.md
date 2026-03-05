# PipeScope MVP (Gateway + Admin Web) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 交付可运行的 PipeScope MVP：多规则 TCP 透明转发、异步 SQLite 落库、geo 补全、内置 Vue+ECharts 管理网页（地图优先 + 规则页 + 明细页）。

**Architecture:** 单进程分层架构。`gateway` 只负责 accept/dial/copy/count/enqueue；`writer` 负责批量 geo 补全与 SQLite 写入；`admin` 只读查询；`web` 由 Go 静态托管。通过队列和批处理隔离热路径与冷路径。

**Tech Stack:** Go 1.23+、SQLite（modernc.org/sqlite 或 mattn/go-sqlite3 二选一，优先 CGO-free）、Vue 3 + Vite + ECharts、YAML 配置、ip2region xdb、AreaCity ok_geo.csv。

---

执行纪律（每个任务都遵循）：`@test-driven-development`、`@systematic-debugging`（仅失败时）、`@verification-before-completion`。

### Task 1: 初始化项目骨架与配置加载

**Files:**
- Create: `go.mod`
- Create: `cmd/pipescope/main.go`
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Create: `assets/config.example.yaml`

**Step 1: Write the failing test**

```go
func TestLoadConfig(t *testing.T) {
    cfg, err := Load("testdata/config.yaml")
    if err != nil { t.Fatalf("unexpected error: %v", err) }
    if cfg.Admin.Host != "127.0.0.1" || cfg.Admin.Port != 9100 {
        t.Fatalf("admin config mismatch: %+v", cfg.Admin)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config -v`
Expected: FAIL，报 `undefined: Load` 或模块不存在。

**Step 3: Write minimal implementation**

```go
type Config struct {
    Data struct {
        SQLitePath      string `yaml:"sqlite_path"`
        IP2RegionXDB    string `yaml:"ip2region_xdb_path"`
        AreaCityCSVPath string `yaml:"areacity_csv_path"`
    } `yaml:"data"`
    // proxy_rules, writer, timeouts, admin...
}

func Load(path string) (*Config, error) {
    b, err := os.ReadFile(path)
    if err != nil { return nil, err }
    var cfg Config
    if err := yaml.Unmarshal(b, &cfg); err != nil { return nil, err }
    return &cfg, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config -v`
Expected: PASS。

**Step 5: Commit**

```bash
git add go.mod cmd/pipescope/main.go internal/config/config.go internal/config/config_test.go assets/config.example.yaml
git commit -m "chore: bootstrap project and config loader"
```

### Task 2: ConnSession 模型与状态归类

**Files:**
- Create: `internal/gateway/session/session.go`
- Create: `internal/gateway/session/session_test.go`

**Step 1: Write the failing test**

```go
func TestFinalizeDialFail(t *testing.T) {
    s := New("r1", 10001, "1.1.1.1:1234", "2.2.2.2:80")
    s.MarkDialFail(errors.New("refused"))
    e := s.Finalize()
    if e.Status != "dial_fail" { t.Fatalf("status=%s", e.Status) }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/gateway/session -v`
Expected: FAIL，`undefined: New/MarkDialFail/Finalize`。

**Step 3: Write minimal implementation**

```go
type ConnSession struct {
    RuleID string
    StartTS int64
    EndTS int64
    UpBytes int64
    DownBytes int64
    Status string
}

func (s *ConnSession) Finalize() Event {
    // fill duration and total bytes
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/gateway/session -v`
Expected: PASS。

**Step 5: Commit**

```bash
git add internal/gateway/session/session.go internal/gateway/session/session_test.go
git commit -m "feat: add conn session model and status classification"
```

### Task 3: 多规则代理运行器（监听 + 拨号 + 双向 copy 计数）

**Files:**
- Create: `internal/gateway/rule/rule.go`
- Create: `internal/gateway/proxy/runner.go`
- Create: `internal/gateway/proxy/copy.go`
- Create: `internal/gateway/proxy/runner_test.go`

**Step 1: Write the failing test**

```go
func TestProxyForwardsBytes(t *testing.T) {
    // start local upstream echo server
    // start runner with one rule
    // dial listen addr and exchange payload
    // assert event up/down/total > 0
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/gateway/proxy -run TestProxyForwardsBytes -v`
Expected: FAIL，转发器尚未实现。

**Step 3: Write minimal implementation**

```go
func proxyConn(ctx context.Context, c net.Conn, rule Rule, out chan<- session.Event) {
    up, err := d.DialContext(ctx, "tcp", rule.Forward)
    if err != nil { /* emit dial_fail */ return }
    // io.CopyBuffer in both directions, count bytes atomically
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/gateway/proxy -run TestProxyForwardsBytes -v`
Expected: PASS。

**Step 5: Commit**

```bash
git add internal/gateway/rule/rule.go internal/gateway/proxy/runner.go internal/gateway/proxy/copy.go internal/gateway/proxy/runner_test.go
git commit -m "feat: implement multi-rule transparent tcp forwarding"
```

### Task 4: 超时与错误状态覆盖（dial/idle）

**Files:**
- Modify: `internal/gateway/proxy/runner.go`
- Modify: `internal/gateway/proxy/copy.go`
- Create: `internal/gateway/proxy/timeout_test.go`

**Step 1: Write the failing test**

```go
func TestDialTimeoutStatus(t *testing.T) {
    // unreachable forward addr with small dial timeout
    // assert status == "timeout" or "dial_fail" by policy
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/gateway/proxy -run TestDialTimeoutStatus -v`
Expected: FAIL，当前未区分超时。

**Step 3: Write minimal implementation**

```go
if ne, ok := err.(net.Error); ok && ne.Timeout() {
    sess.MarkTimeout(err)
} else {
    sess.MarkDialFail(err)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/gateway/proxy -run TestDialTimeoutStatus -v`
Expected: PASS。

**Step 5: Commit**

```bash
git add internal/gateway/proxy/runner.go internal/gateway/proxy/copy.go internal/gateway/proxy/timeout_test.go
git commit -m "feat: add dial and idle timeout handling"
```

### Task 5: SQLite schema 与仓储初始化

**Files:**
- Create: `internal/store/sqlite/schema.sql`
- Create: `internal/store/sqlite/store.go`
- Create: `internal/store/sqlite/store_test.go`

**Step 1: Write the failing test**

```go
func TestInitSchemaCreatesTables(t *testing.T) {
    db := openTempDB(t)
    s := New(db)
    if err := s.InitSchema(context.Background()); err != nil { t.Fatal(err) }
    requireTable(t, db, "conn_events")
    requireTable(t, db, "dim_adcode")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/store/sqlite -run TestInitSchemaCreatesTables -v`
Expected: FAIL，缺少 `InitSchema`。

**Step 3: Write minimal implementation**

```go
func (s *Store) InitSchema(ctx context.Context) error {
    _, err := s.db.ExecContext(ctx, schemaSQL)
    return err
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/store/sqlite -run TestInitSchemaCreatesTables -v`
Expected: PASS。

**Step 5: Commit**

```bash
git add internal/store/sqlite/schema.sql internal/store/sqlite/store.go internal/store/sqlite/store_test.go
git commit -m "feat: add sqlite schema and store bootstrap"
```

### Task 6: 异步队列与批量 writer（不含 geo）

**Files:**
- Create: `internal/store/sqlite/writer.go`
- Create: `internal/store/sqlite/writer_test.go`

**Step 1: Write the failing test**

```go
func TestWriterBatchInsert(t *testing.T) {
    // enqueue N events, start writer with batch_size=3
    // wait flush and assert row count == N
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/store/sqlite -run TestWriterBatchInsert -v`
Expected: FAIL，writer 未实现。

**Step 3: Write minimal implementation**

```go
func (w *Writer) Run(ctx context.Context) error {
    ticker := time.NewTicker(w.flushInterval)
    // collect into batch by size or interval
    // single transaction insert
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/store/sqlite -run TestWriterBatchInsert -v`
Expected: PASS。

**Step 5: Commit**

```bash
git add internal/store/sqlite/writer.go internal/store/sqlite/writer_test.go
git commit -m "feat: add async queue consumer and batch sqlite writer"
```

### Task 7: ip2region 封装与省市归一化

**Files:**
- Create: `internal/geo/ip2region/searcher.go`
- Create: `internal/geo/ip2region/searcher_test.go`
- Create: `internal/geo/normalize/name.go`
- Create: `internal/geo/normalize/name_test.go`

**Step 1: Write the failing test**

```go
func TestParseRegion(t *testing.T) {
    got := ParseRegion("中国|广东省|深圳市|电信|CN")
    if got.Province != "广东" || got.City != "深圳" { t.Fatal(got) }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/geo/... -v`
Expected: FAIL，`ParseRegion`/归一化函数不存在。

**Step 3: Write minimal implementation**

```go
func NormalizeProvince(s string) string {
    s = strings.TrimSuffix(s, "省")
    s = strings.TrimSuffix(s, "市")
    return strings.TrimSpace(s)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/geo/... -v`
Expected: PASS。

**Step 5: Commit**

```bash
git add internal/geo/ip2region/searcher.go internal/geo/ip2region/searcher_test.go internal/geo/normalize/name.go internal/geo/normalize/name_test.go
git commit -m "feat: add ip2region adapter and location normalization"
```

### Task 8: AreaCity 导入与 adcode 匹配

**Files:**
- Create: `internal/geo/areacity/importer.go`
- Create: `internal/geo/areacity/matcher.go`
- Create: `internal/geo/areacity/importer_test.go`
- Create: `internal/geo/areacity/testdata/ok_geo_sample.csv`

**Step 1: Write the failing test**

```go
func TestImportAndMatchByProvinceCity(t *testing.T) {
    // import sample csv into dim_adcode
    // match("广东", "深圳") => has adcode/lng/lat
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/geo/areacity -v`
Expected: FAIL，导入器和匹配器缺失。

**Step 3: Write minimal implementation**

```go
func (m *Matcher) Match(province, city string) (DimAdcode, bool, error) {
    // normalized exact match on (province, city)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/geo/areacity -v`
Expected: PASS。

**Step 5: Commit**

```bash
git add internal/geo/areacity/importer.go internal/geo/areacity/matcher.go internal/geo/areacity/importer_test.go internal/geo/areacity/testdata/ok_geo_sample.csv
git commit -m "feat: import areacity csv and map to adcode coordinates"
```

### Task 9: writer geo 补全流水线

**Files:**
- Modify: `internal/store/sqlite/writer.go`
- Modify: `internal/store/sqlite/writer_test.go`
- Create: `internal/store/sqlite/enrich.go`

**Step 1: Write the failing test**

```go
func TestWriterEnrichesGeoFields(t *testing.T) {
    // fake ip2region returns 广东/深圳
    // fake matcher returns adcode+lat/lng
    // assert inserted conn_events has geo fields
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/store/sqlite -run TestWriterEnrichesGeoFields -v`
Expected: FAIL，未执行 geo enrichment。

**Step 3: Write minimal implementation**

```go
if geo, err := w.region.Lookup(evt.SrcIP); err == nil {
    evt.Province = normalize.NormalizeProvince(geo.Province)
    evt.City = normalize.NormalizeCity(geo.City)
    if dim, ok, _ := w.matcher.Match(evt.Province, evt.City); ok {
        evt.Adcode, evt.Lat, evt.Lng = dim.Adcode, dim.Lat, dim.Lng
    }
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/store/sqlite -run TestWriterEnrichesGeoFields -v`
Expected: PASS。

**Step 5: Commit**

```bash
git add internal/store/sqlite/writer.go internal/store/sqlite/writer_test.go internal/store/sqlite/enrich.go
git commit -m "feat: enrich connection events with geo and adcode"
```

### Task 10: Admin 查询服务（地图/规则/明细）

**Files:**
- Create: `internal/admin/service/service.go`
- Create: `internal/admin/service/query.go`
- Create: `internal/admin/service/service_test.go`

**Step 1: Write the failing test**

```go
func TestChinaMapAggregation(t *testing.T) {
    // seed conn_events
    // call ChinaMap(window=15m, metric=conn)
    // assert grouped by city-level adcode
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/admin/service -v`
Expected: FAIL，查询服务未实现。

**Step 3: Write minimal implementation**

```go
func (s *Service) ChinaMap(ctx context.Context, q MapQuery) ([]MapPoint, error) {
    // SELECT adcode, city, SUM(total_bytes), COUNT(*) ... GROUP BY adcode
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/admin/service -v`
Expected: PASS。

**Step 5: Commit**

```bash
git add internal/admin/service/service.go internal/admin/service/query.go internal/admin/service/service_test.go
git commit -m "feat: add admin read service for map rules and sessions"
```

### Task 11: Admin HTTP API 与静态资源托管

**Files:**
- Create: `internal/admin/http/server.go`
- Create: `internal/admin/http/handlers.go`
- Create: `internal/admin/http/server_test.go`

**Step 1: Write the failing test**

```go
func TestGetHealth(t *testing.T) {
    srv := newTestServer(t)
    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
    srv.Handler().ServeHTTP(rr, req)
    if rr.Code != 200 { t.Fatalf("code=%d", rr.Code) }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/admin/http -v`
Expected: FAIL，路由未注册。

**Step 3: Write minimal implementation**

```go
mux.HandleFunc("/api/health", h.handleHealth)
mux.HandleFunc("/api/map/china", h.handleMapChina)
// ...rules/sessions/overview/province
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/admin/http -v`
Expected: PASS。

**Step 5: Commit**

```bash
git add internal/admin/http/server.go internal/admin/http/handlers.go internal/admin/http/server_test.go
git commit -m "feat: add read-only admin http api"
```

### Task 12: Vue 管理端（地图优先 + 规则页 + 明细页）

**Files:**
- Create: `web/admin/package.json`
- Create: `web/admin/vite.config.ts`
- Create: `web/admin/src/main.ts`
- Create: `web/admin/src/App.vue`
- Create: `web/admin/src/pages/MapPage.vue`
- Create: `web/admin/src/pages/RulesPage.vue`
- Create: `web/admin/src/pages/SessionsPage.vue`
- Create: `web/admin/src/api/client.ts`
- Create: `web/admin/src/styles.css`
- Test: `web/admin/src/pages/MapPage.test.ts`

**Step 1: Write the failing test**

```ts
it("renders map page and calls china map api", async () => {
  // mount MapPage, mock /api/map/china
  // expect request fired with default window=15m
})
```

**Step 2: Run test to verify it fails**

Run: `cd web/admin && npm test`
Expected: FAIL，页面和 API 客户端未实现。

**Step 3: Write minimal implementation**

```ts
setInterval(() => fetchChinaMap({ window: state.window, metric: state.metric }), 5000)
```

```vue
<template>
  <div class="layout">
    <MapPage v-if="tab==='map'" />
    <RulesPage v-else-if="tab==='rules'" />
    <SessionsPage v-else />
  </div>
</template>
```

**Step 4: Run test to verify it passes**

Run: `cd web/admin && npm test`
Expected: PASS。

**Step 5: Commit**

```bash
git add web/admin
git commit -m "feat: add vue admin ui with map rules and sessions pages"
```

### Task 13: 前端构建产物接入 Go 与主程序编排

**Files:**
- Modify: `cmd/pipescope/main.go`
- Create: `internal/admin/http/static_embed.go`
- Create: `Makefile`
- Create: `cmd/pipescope/main_test.go`

**Step 1: Write the failing test**

```go
func TestServeAdminIndex(t *testing.T) {
    // start app with admin enabled
    // GET / should return index.html
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/pipescope -v`
Expected: FAIL，静态资源未挂载。

**Step 3: Write minimal implementation**

```go
//go:embed ../../web/dist/*
var webFS embed.FS

func main() {
    // load config
    // start proxy runner
    // start writer
    // start admin server(host,port)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/pipescope -v`
Expected: PASS。

**Step 5: Commit**

```bash
git add cmd/pipescope/main.go internal/admin/http/static_embed.go Makefile cmd/pipescope/main_test.go
git commit -m "feat: wire gateway writer admin and embedded web assets"
```

### Task 14: 端到端验证与文档收尾

**Files:**
- Create: `tests/e2e/pipescope_e2e_test.go`
- Create: `docs/runbook.md`
- Modify: `README.md`

**Step 1: Write the failing test**

```go
func TestE2EForwardRecordAndQuery(t *testing.T) {
    // launch app with temp config/db
    // send tcp payload through rule
    // query /api/map/china and /api/sessions
    // assert records visible
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./tests/e2e -v`
Expected: FAIL，E2E glue missing。

**Step 3: Write minimal implementation**

```go
// helper: start app, wait health, send data, poll api until row appears
```

**Step 4: Run full verification**

Run:
- `go test ./... -v`
- `cd web/admin && npm test && npm run build`

Expected: 全部 PASS；生成 `web/dist`。

**Step 5: Commit**

```bash
git add tests/e2e/pipescope_e2e_test.go docs/runbook.md README.md web/dist
git commit -m "test: add e2e coverage and operational docs"
```

---

## 执行顺序与检查点

1. 先完成 Task 1-4（可跑通转发链路）。
2. 再完成 Task 5-9（可靠落库 + geo 补全）。
3. 完成 Task 10-13（API + UI + 编排）。
4. 最后 Task 14（端到端与文档）。

每个任务完成后必须执行：
- 单任务相关测试
- `git diff --stat` 自检
- 提交前确认没有无关文件改动

## 故障处理准则

1. 任一测试失败先用 `@systematic-debugging` 定位，不跳过测试。
2. writer 或查询性能异常时，先检查索引与 SQL 计划，再考虑加预聚合。
3. 地图数据缺失时，优先核对省市归一化与 `dim_adcode` 导入质量。

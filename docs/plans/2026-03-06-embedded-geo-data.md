# Embedded Geo Data Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 让 PipeScope 的发布二进制内置 IPv4 IP 库与地理位置匹配数据，下载后无需额外下载 geo/IP 数据或启动 AreaCity API。

**Architecture:** 新增 `internal/embeddeddata` 包统一提供内置资产、版本信息与启动时加载逻辑。`ip2region` 资产以 gzip 形式内置并落盘到缓存目录；AreaCity seed 以精简 CSV gzip 形式内置，并按版本导入 SQLite 的 `dim_adcode` 表。主程序启动时只走内置数据路径。

**Tech Stack:** Go 1.24、go:embed、gzip、SQLite（modernc.org/sqlite）、现有 `ip2region` 与 `areacity` 代码。

---

### Task 1: 为 SQLite 增加内置数据版本元信息能力

**Files:**
- Modify: `internal/store/sqlite/schema.sql`
- Test: `internal/store/sqlite/store_test.go`

**Step 1: Write the failing test**

在 `store_test.go` 增加断言，验证初始化 schema 后存在 `app_meta` 表。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/store/sqlite -run TestInitSchemaCreatesTables -v`
Expected: FAIL，缺少 `app_meta` 表。

**Step 3: Write minimal implementation**

在 `schema.sql` 中新增 `app_meta(key TEXT PRIMARY KEY, value TEXT NOT NULL DEFAULT '')`。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/store/sqlite -run TestInitSchemaCreatesTables -v`
Expected: PASS。

### Task 2: 扩展 AreaCity importer 支持从 reader 导入与全量替换

**Files:**
- Modify: `internal/geo/areacity/importer.go`
- Modify: `internal/geo/areacity/importer_test.go`

**Step 1: Write the failing test**

增加测试：使用 `ReplaceCSVReader` 导入一份精简 CSV 后，`dim_adcode` 中旧记录被替换，新记录可被 `Matcher` 查到。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/geo/areacity -run TestReplaceCSVReader -v`
Expected: FAIL，`ReplaceCSVReader` 未定义。

**Step 3: Write minimal implementation**

为 importer 提供 `ImportCSVReader` / `ReplaceCSVReader`，复用现有 CSV 解析逻辑并在 replace 模式下先清空 `dim_adcode`。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/geo/areacity -run TestReplaceCSVReader -v`
Expected: PASS。

### Task 3: 新增 embeddeddata 包并覆盖核心加载行为

**Files:**
- Create: `internal/embeddeddata/embed.go`
- Create: `internal/embeddeddata/manifest.go`
- Create: `internal/embeddeddata/ip2region.go`
- Create: `internal/embeddeddata/areacity.go`
- Create: `internal/embeddeddata/embeddeddata_test.go`
- Create: `internal/embeddeddata/assets/*`

**Step 1: Write the failing test**

增加测试：
- `EnsureIP2RegionXDB` 能把 gzip 资产写到缓存目录
- `EnsureAreaCitySeed` 首次导入后写入版本，二次调用不重复导入

**Step 2: Run test to verify it fails**

Run: `go test ./internal/embeddeddata -v`
Expected: FAIL，包或方法不存在。

**Step 3: Write minimal implementation**

实现 manifest 读取、gzip 解压、缓存文件准备、SQLite 版本检查与 seed 导入。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/embeddeddata -v`
Expected: PASS。

### Task 4: 主程序切换到内置 geo/IP 启动路径

**Files:**
- Modify: `cmd/pipescope/main.go`
- Modify: `assets/config.example.yaml`
- Test: `cmd/pipescope/main_test.go`

**Step 1: Write the failing test**

增加测试，验证在不提供外部 geo/IP 配置时，主流程仍能初始化 geo enrich 依赖。

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/pipescope -v`
Expected: FAIL，仍依赖外部路径或 HTTP API。

**Step 3: Write minimal implementation**

主程序调用 `embeddeddata` 初始化：
- 准备内置 xdb 路径
- 导入内置 AreaCity seed
- 构造 matcher/searcher

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/pipescope -v`
Expected: PASS。

### Task 5: 增加数据更新脚本与生成器

**Files:**
- Create: `cmd/gen-embedded-geo-data/main.go`
- Create: `cmd/gen-embedded-geo-data/main_test.go`
- Create: `scripts/update-embedded-geo-data.sh`
- Modify: `Makefile`

**Step 1: Write the failing test**

为生成器核心逻辑补最小测试，验证它能从小样本 CSV 导出精简 seed 与 manifest。

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/gen-embedded-geo-data -v`
Expected: FAIL，生成器不存在。

**Step 3: Write minimal implementation**

实现生成器与脚本，支持下载原始数据并更新 `internal/embeddeddata/assets`。

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/gen-embedded-geo-data -v`
Expected: PASS。

### Task 6: 更新文档并验证核心路径

**Files:**
- Modify: `README.md`
- Modify: `docs/runbook.md`
- Modify: `tests/e2e/pipescope_e2e_test.go`（仅在必要时更新配置样例）

**Step 1: Write the failing test**

若命令或运行说明与实现不符，先补充最小验证或 smoke 命令。

**Step 2: Run verification**

Run:
- `go test ./internal/store/sqlite ./internal/geo/areacity ./internal/embeddeddata ./cmd/pipescope -v`
- `go build ./cmd/pipescope`

Expected: 相关测试通过，二进制构建成功。

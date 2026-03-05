# GitHub Multi-Platform Build Design

## 背景

当前 PipeScope 的本地构建流程为：

- 前端构建与静态资源同步：`make build-web sync-web`
- 后端构建：`go build ./cmd/pipescope`

并通过 `internal/admin/http/static_embed.go` 中的 `//go:embed static/*` 将静态资源打包进二进制，实现开箱即用。

项目尚未配置 GitHub Actions 工作流，无法在远端统一产出多平台可用二进制并自动发布。

## 目标

新增 GitHub 构建脚本（workflow），实现：

1. 支持三种触发时机：
   - Push 到 `main`
   - Push tag（`v*`）
   - 手动触发（`workflow_dispatch`）
2. 支持四个平台构建：
   - `linux/amd64`
   - `linux/arm64`
   - `darwin/amd64`
   - `darwin/arm64`
3. 产物仅为二进制文件（不打包压缩包）
4. 确保静态资源已内嵌到二进制中
5. 在 tag 触发时自动创建 GitHub Release 并上传四个平台二进制

## 非目标

- 暂不支持 Windows 平台
- 暂不接入 GoReleaser
- 暂不引入签名、SBOM、校验和文件
- 暂不改动运行时代码与业务逻辑

## 方案对比

### 方案 A：纯 GitHub Actions + matrix（推荐）

- 在 `.github/workflows/release.yml` 中实现完整链路
- 使用 matrix 生成四个平台构建
- 使用 `actions/upload-artifact` 存档
- 使用 `softprops/action-gh-release` 在 tag 事件发布

优点：
- 与当前项目结构和 Makefile 高度贴合
- 落地快，维护简单
- 可逐步演进到更复杂发行流程

缺点：
- 命名与发布规则由 workflow 自行维护

### 方案 B：GoReleaser

优点：
- 发行能力成熟、可扩展性强

缺点：
- 当前需求较轻，引入成本相对更高

### 方案 C：Makefile 主导 cross-build + workflow 调用

优点：
- 本地和 CI 命令统一

缺点：
- Makefile 复杂度上升，workflow 仍需处理发布条件

结论：采用方案 A。

## 设计细节

### 1. 工作流结构

新增 `.github/workflows/release.yml`，包含以下核心部分：

- `on`:
  - `push.branches: [main]`
  - `push.tags: ["v*"]`
  - `workflow_dispatch`
- `jobs.build`：
  - 使用 `strategy.matrix` 定义四平台
  - 安装 Node.js 与 Go
  - 安装前端依赖并构建前端
  - 执行 `make sync-web` 同步 embed 静态资源
  - 按 matrix 目标执行 `go build`
  - 上传二进制为 artifact
- `jobs.release`：
  - 仅在 tag 事件执行
  - 下载所有 artifact
  - 创建 GitHub Release 并上传二进制

### 2. 静态资源内嵌保障

构建顺序必须先执行：

1. `npm ci` + `npm run build`（在 `web/admin`）
2. `make sync-web`（把前端产物同步到 `internal/admin/http/static`）
3. `go build`（触发 `//go:embed static/*`）

这样可保证发布二进制自带最新管理页面资源。

### 3. 二进制命名

命名约定：`pipescope-<goos>-<goarch>`

示例：

- `pipescope-linux-amd64`
- `pipescope-linux-arm64`
- `pipescope-darwin-amd64`
- `pipescope-darwin-arm64`

### 4. Release 策略

- `main`/`workflow_dispatch`：只上传 artifact（用于持续验证）
- `tag(v*)`：自动发布 GitHub Release 并上传四个二进制

## 变更范围

- 新增：`.github/workflows/release.yml`
- 可选文档更新（若需要）：`README.md`（增加发布说明）

不修改 Go 业务代码。

## 验证策略

### CI 验证

1. 手动触发 workflow，确认四个平台构建都成功
2. 检查 artifacts 是否存在四个目标二进制
3. 打 tag（如 `v0.1.0-test`）后确认自动创建 Release 并包含四个二进制

### 本地最小验证

- 可本地执行 `make build-web sync-web && go build ./cmd/pipescope` 确认不破坏既有流程

## 风险与回滚

风险：
- GitHub Actions 权限不足导致 Release 上传失败
- Node/Go 版本漂移导致构建不稳定

缓解：
- 在 workflow 中显式指定 Go 与 Node 版本
- 使用 `permissions: contents: write` 仅赋予发布所需权限

回滚：
- 删除或回退 `.github/workflows/release.yml` 即可，不影响运行时代码。

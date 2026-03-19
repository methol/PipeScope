# Analytics Options Cache Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce repeated DB work for internal/admin `AnalyticsOptions` requests by adding a one-minute success-only cache keyed by the full query, with same-key in-flight dedupe, while keeping analytics and map behavior compatible.

**Architecture:** Keep the public `Service.AnalyticsOptions` method and extract its DB query logic behind an internal loader boundary. Add a small service-local cache wrapper backed by `github.com/hashicorp/golang-lru/v2/expirable` and `golang.org/x/sync/singleflight`, then cover cache-hit, no-error-cache, and in-flight dedupe behavior with focused service tests before running the required Go and frontend verification commands.

**Tech Stack:** Go 1.24, SQLite, Vue 3, Vitest, Vite

---

## Execution Notes

- User explicitly forbids git worktrees; all work stays in `/Users/methol/code/github.com/methol/PipeScope`.
- Network access is restricted in this environment, so dependency use may need local in-repo replacements if the modules are not already cached.
- Keep HTTP and frontend production behavior unchanged unless a test proves otherwise.
- Review must be done manually in-session; do not use the `codex review` command.

## File Map

- Modify: `go.mod`
- Modify: `internal/admin/service/service.go`
- Modify: `internal/admin/service/analytics_test.go`
- Create: `internal/admin/service/analytics_options_cache.go`
- Create if needed for offline dependency resolution: `third_party/github.com/hashicorp/golang-lru/v2/go.mod`
- Create if needed for offline dependency resolution: `third_party/github.com/hashicorp/golang-lru/v2/expirable/expirable.go`
- Create if needed for offline dependency resolution: `third_party/golang.org/x/sync/go.mod`
- Create if needed for offline dependency resolution: `third_party/golang.org/x/sync/singleflight/singleflight.go`
- Create later: `docs/process-checklists/2026-03-19-analytics-options-cache-review.md`

## Chunk 1: Lock cache behavior with TDD

### Task 1: Add failing cache-behavior service tests

**Files:**
- Modify: `internal/admin/service/analytics_test.go`
- Test: `internal/admin/service/analytics_test.go`

- [ ] **Step 1: Write the failing tests**

  Add focused tests proving that:
  - repeated identical queries reuse one successful load
  - failed loads are not cached
  - concurrent identical queries are deduplicated to one in-flight load

- [ ] **Step 2: Run the focused RED command**

  Run: `go test ./internal/admin/service -run 'TestAnalyticsOptions(CachesSuccessfulResponses|DoesNotCacheErrors|DeduplicatesInflightRequests)$'`

  Expected: FAIL because the cache and dedupe behavior do not exist yet.

## Chunk 2: Implement the cache and make the new tests pass

### Task 2: Add the minimal production cache

**Files:**
- Modify: `go.mod`
- Modify: `internal/admin/service/service.go`
- Create: `internal/admin/service/analytics_options_cache.go`
- Create if needed: `third_party/github.com/hashicorp/golang-lru/v2/go.mod`
- Create if needed: `third_party/github.com/hashicorp/golang-lru/v2/expirable/expirable.go`
- Create if needed: `third_party/golang.org/x/sync/go.mod`
- Create if needed: `third_party/golang.org/x/sync/singleflight/singleflight.go`
- Test: `internal/admin/service/analytics_test.go`

- [ ] **Step 1: Implement the smallest service change**

  Add:
  - a full-query cache key
  - a one-minute expirable LRU cache
  - a same-key `singleflight` boundary for in-flight requests
  - a loader split so only successful results are cached

- [ ] **Step 2: Run the focused GREEN command**

  Run: `go test ./internal/admin/service -run 'TestAnalyticsOptions(CachesSuccessfulResponses|DoesNotCacheErrors|DeduplicatesInflightRequests)$'`

  Expected: PASS.

- [ ] **Step 3: Run the full service package**

  Run: `go test ./internal/admin/service`

  Expected: PASS with existing analytics behavior preserved.

## Chunk 3: Full verification and review

### Task 3: Run the required verification commands

**Files:**
- Test: `internal/admin/service/analytics_test.go`
- Test: `internal/admin/http/server_test.go`
- Test: `web/admin/src/pages/MapPage.test.ts`

- [ ] **Step 1: Run the required Go verification**

  Run: `go test ./internal/admin/service ./internal/admin/http ./cmd/pipescope`

  Expected: PASS.

- [ ] **Step 2: Run the required frontend page test**

  Run: `cd web/admin && npm test -- --run src/pages/MapPage.test.ts`

  Expected: PASS.

- [ ] **Step 3: Run the required frontend build**

  Run: `cd web/admin && npm run build`

  Expected: PASS.

### Task 4: Request and receive review in-session

**Files:**
- Create: `docs/process-checklists/2026-03-19-analytics-options-cache-review.md`

- [ ] **Step 1: Review the implementation manually**

  Inspect:
  - `git diff -- go.mod internal/admin/service/service.go internal/admin/service/analytics_options_cache.go internal/admin/service/analytics_test.go`

- [ ] **Step 2: Record findings, fixes, and verification**

  Capture up to three review rounds, including:
  - finding severity
  - decision
  - applied fix or rationale
  - fresh verification evidence after each fix

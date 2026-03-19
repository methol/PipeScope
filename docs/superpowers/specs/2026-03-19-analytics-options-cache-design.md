# Analytics Options Cache Design

## Context

- Scope is limited to the internal/admin analytics options path:
  - [`internal/admin/service/service.go`](/Users/methol/code/github.com/methol/PipeScope/internal/admin/service/service.go)
  - [`internal/admin/service/analytics_test.go`](/Users/methol/code/github.com/methol/PipeScope/internal/admin/service/analytics_test.go)
  - [`internal/admin/http/handlers.go`](/Users/methol/code/github.com/methol/PipeScope/internal/admin/http/handlers.go)
  - [`web/admin/src/pages/AnalyticsPage.test.ts`](/Users/methol/code/github.com/methol/PipeScope/web/admin/src/pages/AnalyticsPage.test.ts)
  - [`web/admin/src/pages/MapPage.test.ts`](/Users/methol/code/github.com/methol/PipeScope/web/admin/src/pages/MapPage.test.ts)
- `Service.AnalyticsOptions` currently performs four DB queries for every request to build the rules, provinces, cities, and statuses filter options.
- Both analytics and map flows depend on `/api/analytics/options`, so the change must remain response-compatible.
- The approved behavior is fixed:
  - cache only successful `AnalyticsOptions` responses
  - use `singleflight` only for in-flight dedupe
  - do not cache errors
  - cache key must include `window`, `rule_id`, `province`, `city`, `status`, and `src_ip`
  - TTL should be about one minute

## Options

### Option 1: Cache inside `Service.AnalyticsOptions`

- Add a reusable cache helper in `internal/admin/service` with an `expirable` LRU and a `singleflight.Group`.
- Keep HTTP and frontend unchanged.
- Use the full query shape as the cache key.

Recommendation: yes. This is the smallest production diff and keeps the cache close to the only call site that needs it today.

### Option 2: Cache at the HTTP handler layer

- Cache JSON responses in `internal/admin/http`.
- Leave service unchanged.

Trade-off: duplicates query-key logic outside the data layer and couples transport details to caching.

### Option 3: Cache every analytics read path

- Generalize caching for analytics and analytics-options together.

Trade-off: larger scope, more invalidation risk, and unnecessary complexity for the approved task.

## Chosen Design

### Service cache boundary

- Keep `Service.AnalyticsOptions` as the public API.
- Move the current DB aggregation logic into an internal loader method.
- Add a reusable internal cache helper that:
  - looks up an `AnalyticsOptions` value by a comparable query key
  - deduplicates only concurrent loads for the same key with `singleflight`
  - stores only successful results in an `expirable` LRU with a fixed one-minute TTL

### Query key

- The cache key will include:
  - `window`
  - `rule_id`
  - `province`
  - `city`
  - `status`
  - `src_ip`
- Different filter combinations remain isolated, so cross-filtered analytics and map pages keep their current behavior.

### Error handling

- Loader errors return directly to the caller.
- Failed loads are never inserted into the cache.
- `singleflight` only suppresses duplicate in-flight work; once the failing call completes, the next caller tries the DB again.

### Testing

- Keep the existing SQLite-backed analytics tests as the behavioral regression suite.
- Add focused service tests that prove:
  - repeated identical queries reuse one successful load
  - failed loads are retried and are not cached
  - same-key concurrent calls are deduplicated to one in-flight load

## Approval Note

- The user already approved the cache policy and instructed implementation to proceed immediately, so this document records that approved design rather than reopening the decision.

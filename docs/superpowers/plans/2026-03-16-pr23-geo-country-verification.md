# PR #23 Geo Country Verification Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add auditable geo/policy/persistence regression coverage for PR #23, make any minimal code fixes required, and verify the branch with full `go test ./...`.

**Architecture:** Keep changes narrow. Prefer test fixtures and mockable `ip2region` lookup results over depending on unstable offline IP databases, while still asserting the real field mapping used by runtime code. Only touch production code where tests expose a concrete gap, especially country persistence and session query readability.

**Tech Stack:** Go, `modernc.org/sqlite`, embedded geo seed data, `ip2region` lookup fixtures

---

## Chunk 1: Geo Sample And Policy Coverage

### Task 1: Add auditable geo sample lookup tests

**Files:**
- Modify: `internal/gateway/geo/lookup_test.go`
- Test: `internal/gateway/geo/lookup_test.go`

- [ ] **Step 1: Write the failing test**

Add a table-driven test that feeds sample public IP strings through `LookupFunc` using `Searcher.SetLookupFn(...)` and a seeded SQLite `areacity.Matcher`, then asserts `country`, `province`, `city`, and `adcode` for at least:
- Sichuan Chengdu
- Hubei Wuhan
- Guizhou Zunyi
- Xinjiang
- Tibet

Annotate each sample with a short comment explaining that the IP string is a real public-format sample while the `ip2region` raw result is fixture-injected for auditable and stable mapping.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/gateway/geo -run 'TestLookupFunc_' -count=1`
Expected: at least one new assertion fails before implementation is updated.

- [ ] **Step 3: Write minimal implementation**

Only if the new lookup assertions expose a production-code gap. Keep lookup country/province/city/adcode mapping minimal.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/gateway/geo -run 'TestLookupFunc_' -count=1`
Expected: PASS

### Task 2: Add explicit allow/deny policy regression matrix

**Files:**
- Modify: `internal/gateway/geo/policy_test.go`
- Test: `internal/gateway/geo/policy_test.go`

- [ ] **Step 1: Write the failing test**

Add policy coverage for:
- allow: `CN`
- deny: `CN` with provinces `新疆`, `西藏`
- both `require_allow_hit=true` and `require_allow_hit=false`

Assert Sichuan/Hubei/Guizhou are allowed, and Xinjiang/Tibet are blocked.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/gateway/geo -run 'TestMatcher_' -count=1`
Expected: new case fails if behavior or coverage is incomplete.

- [ ] **Step 3: Write minimal implementation**

Only if matcher behavior is wrong. Do not broaden scope beyond the required allow/deny handling.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/gateway/geo -run 'TestMatcher_' -count=1`
Expected: PASS

## Chunk 2: Country Persistence And Query Readability

### Task 3: Add persistence/query regression for `country`

**Files:**
- Modify: `internal/store/sqlite/writer_test.go`
- Modify: `internal/admin/service/service_test.go`
- Modify: `internal/admin/service/service.go`
- Test: `internal/store/sqlite/writer_test.go`
- Test: `internal/admin/service/service_test.go`

- [ ] **Step 1: Write the failing tests**

Add:
- a store/writer assertion that `conn_events.country` is persisted for enriched or event-provided geo
- a service-level sessions query test that reads `country` back through `Service.Sessions`

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/store/sqlite ./internal/admin/service -run 'TestWriter|TestSessions' -count=1`
Expected: failure if session scanning or persistence is incomplete.

- [ ] **Step 3: Write minimal implementation**

Fix only the concrete gaps the tests expose. If schema/migration/writer are already sufficient, only adjust the reader path.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/store/sqlite ./internal/admin/service -run 'TestWriter|TestSessions' -count=1`
Expected: PASS

## Chunk 3: Verification, Review Chain, And Commit

### Task 4: Verify branch, document evidence, and commit

**Files:**
- Modify: `PROCESS_CHECKLIST.md`

- [ ] **Step 1: Run targeted package tests**

Run: `go test ./internal/gateway/geo ./internal/store/sqlite ./internal/admin/service -count=1`
Expected: PASS

- [ ] **Step 2: Run full verification**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 3: Request code review**

Use superpower:requesting-code-review skill and capture the review target plus output summary in `PROCESS_CHECKLIST.md`.

- [ ] **Step 4: Receive and evaluate code review**

Use superpower:receiving-code-review skill, implement only validated findings, and record fix/no-fix decisions in `PROCESS_CHECKLIST.md`.

- [ ] **Step 5: Re-run verification and commit**

Run:
```bash
go test ./...
git status --short
git add PROCESS_CHECKLIST.md docs/superpowers/plans/2026-03-16-pr23-geo-country-verification.md internal/gateway/geo/lookup_test.go internal/gateway/geo/policy_test.go internal/admin/service/service.go internal/admin/service/service_test.go internal/store/sqlite/writer_test.go
git commit -m "test: verify geo country allow deny coverage"
```
Expected: tests pass and commit is created locally without push

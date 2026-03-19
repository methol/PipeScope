# Analytics Options Cache Review

## Review Mode

- Manual in-session review.
- `codex review` was not used, per instruction.
- Subagent review was not used because this run stayed in the current session and current constraints did not require delegation.

## Round 1

- Severity: important
- Finding: the initial local `golang.org/x/sync/singleflight` replacement implemented only the minimal happy path and did not preserve the upstream package's panic and `runtime.Goexit` handling, which could leave waiters blocked in edge cases.
- Fix: replaced the minimal shim with the upstream `singleflight` implementation copied from the locally available module cache into `third_party/golang.org/x/sync/singleflight/singleflight.go`.
- Verification:
  - `GOCACHE=/tmp/pipescope-go-build go test ./internal/admin/service ./internal/admin/http ./cmd/pipescope -count=1`
  - `cd web/admin && npm test -- --run src/pages/MapPage.test.ts`
  - `cd web/admin && npm run build`

## Round 2

- Severity: none
- Finding: no additional material issues found in the cache keying, success-only caching policy, or response compatibility path.
- Action: no code change required.

## Round 3

- Not needed.

# PROCESS_CHECKLIST

- Task: 实时页 limit 下拉 + 统计页源IP查询 + 地图文案/边界改进
- Date: 2026-03-17
- Repo: /Users/methol/code/github.com/methol/PipeScope
- Branch: feature/realtime-analytics-map (main repo branch after worktree violation correction)

## Step Status
- [x] Step1 self-check
- [ ] Step2 brainstorming
- [ ] Step3 writing-plans
- [x] Step4 executing-plans
- [ ] Step5 requesting-code-review
- [x] Step6 receiving-code-review round1
- [x] Step7 receiving-code-review round2
- [x] Step8 receiving-code-review round3
- [x] Step9 receiving-code-review round4 / final fix
- [x] Step10 receiving-code-review round5 / closure verify

## Notes
- Archived from the repo-root draft `PROCESS_CHECKLIST.md`; keep the root draft out of commits and maintain this copy as the committed record.
- 2026-03-17 worktree violation correction:
  - Removed forbidden worktree: `/Users/methol/code/github.com/methol/PipeScope/.worktrees/feature-realtime-analytics-map`
  - Recreated normal branch in main repo from `origin/main`
  - Commit mapping:
    - `14c06df` -> `2cf808e`
    - `494e63d` -> `7d1e02e`
  - Source tracker from removed worktree was reviewed and folded into this checklist before cleanup.
- 2026-03-17 receiving-code-review round1:
  - 实时页移除 `all` limit 语义，避免 5 秒自动刷新叠加无限量查询；前后端都回退到有界 limit。
  - 地图 city key 改为规范化 6 位 adcode，消除海南/湖北/新疆兵团等重复 key 场景下的 hover 错名。
  - 统计页 options 改为随 `window/rule/province/city/status/src_ip` 联动刷新，并加上请求序号避免旧响应覆盖新筛选。
  - `.tmp/` 已加入 `.gitignore`，本地临时产物不再出现在待索引列表。
  - 追加 SQLite 实测，钉住 analytics options 在交叉过滤下的真实查询行为。
- 2026-03-17 receiving-code-review round2:
  - 统计页在 options 刷新后统一回收失效的 `ruleID/province/status/city`，不再只处理 city。
  - `loadOptions()` 内部改为“拉取 -> 校验 -> 必要时清空并再拉取”的稳定化循环，确保 options 最终和清理后的查询条件一致。
  - cleanup 期间临时屏蔽 watcher 自触发，避免因为清空筛选而形成循环请求或请求风暴。
  - 新增前端回归用例，覆盖“先选旧筛选，再输入使其失效的 `src_ip`，随后 search 不再带旧筛选”。
- 2026-03-17 receiving-code-review round3:
  - 产品口径更新：实时页 limit 固定为 `100 / 1000 / 10000`，不再保留 `all` 语义。
  - `SessionsPage` 下拉与请求参数同步切到 `100 / 1000 / 10000`，保留 5 秒自动刷新但始终有界。
  - 后端把 sessions/map 的 limit 解析与服务层保护统一到默认 `100`、最大 `10000`，`limit=all` 和非法值只回退默认值，不再触发无限查询。
  - 追加 limit 钳制回归测试，覆盖 HTTP 入口与 service 层 helper。
- 2026-03-17 receiving-code-review round4 / final fix:
  - `AnalyticsPage.vue` 的城市失效回收补上 province 维度，避免跨省同名城市在 province 被清空后误保留 stale city。
  - 新增回归用例：从 `广东/长安区` 收窄到只剩 `河北/长安区` 时，city 必须被清空，后续 `search()` 不再带旧 city。
  - 执行 `make build`，按 Makefile 既有流程完成 `web/admin` 构建、`web/dist` 同步以及 `internal/admin/http/static` 嵌入静态目录刷新。
  - 校验新 bundle：embedded 产物已包含 `src_ip` 查询参数、`100/1000/10000` realtime limit 选项，以及地图 `city_key` / `省界` 相关逻辑。
- 2026-03-17 receiving-code-review round5 / closure verify:
  - `AnalyticsPage.vue` 的 options 联动仅在 `src_ip` 为空或为完整 IP（IPv4/IPv6）时触发；半截输入不再触发 options 请求，也不再清空已选 `rule/province/city/status`。
  - 完整 `src_ip` 仍会进入 options 查询，并继续触发 stale filter 回收与联动收敛。
  - 最终复审基于当前 staged diff 逐项检查 `sessions/map/analytics` 相关改动，未发现新的高/中风险问题。

## Test Results
- Self-check: PASS
- 2026-03-17 `go test ./...`: PASS
- 2026-03-17 `npm test -- src/pages/SessionsPage.test.ts src/pages/AnalyticsPage.test.ts src/pages/mapCity.test.ts src/pages/MapPage.test.ts`: PASS
- 2026-03-17 red: `npm test -- src/pages/SessionsPage.test.ts src/pages/mapCity.test.ts src/pages/MapPage.test.ts src/pages/AnalyticsPage.test.ts` -> FAIL（命中 `all` limit、重复 city_key 命名、analytics options 未随 `src_ip` 刷新）
- 2026-03-17 red: `go test ./internal/admin/http ./internal/admin/service -run 'TestSessionsEndpointFallsBackForAllLimit|TestAnalyticsOptionsTracksCrossFiltersInSQLite' -count=1` -> FAIL（`limit=all` 仍被解析为无限）
- 2026-03-17 green: `npm test -- src/pages/SessionsPage.test.ts src/pages/mapCity.test.ts src/pages/MapPage.test.ts src/pages/AnalyticsPage.test.ts` -> PASS
- 2026-03-17 green: `go test ./internal/admin/http ./internal/admin/service -run 'TestSessionsEndpointFallsBackForAllLimit|TestAnalyticsOptionsTracksCrossFiltersInSQLite' -count=1` -> PASS
- 2026-03-17 red: `npm test -- --run src/pages/AnalyticsPage.test.ts` -> FAIL（旧 `rule/province/status` 清空后没有重新同步 options，`search()` 仍带旧筛选）
- 2026-03-17 green: `npm test -- --run src/pages/AnalyticsPage.test.ts` -> PASS
- 2026-03-17 verify: `go test ./internal/admin/http ./internal/admin/service` -> PASS
- 2026-03-17 verify: `npm test -- --run src/pages/AnalyticsPage.test.ts src/pages/SessionsPage.test.ts src/pages/MapPage.test.ts src/pages/mapCity.test.ts` -> PASS
- 2026-03-17 verify: `go test ./internal/admin/http ./internal/admin/service` -> PASS（limit 默认值/10000 上限/`all` 回退均通过）
- 2026-03-17 verify: `npm test -- --run src/pages/AnalyticsPage.test.ts src/pages/SessionsPage.test.ts src/pages/MapPage.test.ts src/pages/mapCity.test.ts` -> PASS（23 tests）
- 2026-03-17 red: `npm test -- --run src/pages/AnalyticsPage.test.ts -t "clears stale city when province collapses but the same city name still exists in another province"` -> FAIL（province 被清空后仍保留 stale city）
- 2026-03-17 green: `npm test -- --run src/pages/AnalyticsPage.test.ts -t "clears stale city when province collapses but the same city name still exists in another province"` -> PASS
- 2026-03-17 verify: `npm test -- --run src/pages/AnalyticsPage.test.ts src/pages/SessionsPage.test.ts src/pages/MapPage.test.ts src/pages/mapCity.test.ts` -> PASS（24 tests）
- 2026-03-17 verify: `go test ./internal/admin/http ./internal/admin/service` -> PASS
- 2026-03-17 verify: `make build` -> PASS（`internal/admin/http/static/assets/index-DsxxM4lf.js` 已刷新）
- 2026-03-17 verify: `rg -n "src_ip|10000|city_key|省界|china-cities|1000|100" internal/admin/http/static/assets/index-*.js` -> PASS（bundle 已含 analytics/src_ip、realtime limits、map join/boundary 代码）
- 2026-03-17 verify: `go test ./internal/admin/http ./internal/admin/service` -> PASS（current run; `internal/admin/http` / `internal/admin/service` 均通过）
- 2026-03-17 verify: `npm test -- --run src/pages/AnalyticsPage.test.ts src/pages/SessionsPage.test.ts src/pages/MapPage.test.ts src/pages/mapCity.test.ts` -> PASS（25 tests；包含“半截 src_ip 不联动清空筛选”回归用例）
- 2026-03-17 verify: final staged diff review -> PASS（未发现新的高/中风险问题）

## Final
- Status: LOCAL_VERIFIED_READY_FOR_ARCHIVE

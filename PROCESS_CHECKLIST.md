# PROCESS_CHECKLIST

## Task

地图页 hover/tooltip/省界可见性最小热修

## Branch

`fix/map-hover-tooltip-boundary-20260318`

## Step 1: brainstorming

- Prompt evidence: `use superpower:brainstorming skill`
- Output:
  - Option 1: 在现有 `geo + map + lines` 结构上做最小热修
  - Option 2: 删除 `lines` overlay
  - Option 3: 替换为新的省级 GeoJSON 数据源
- Decision:
  - 采用 Option 1
  - 原因：满足用户限定的 3 个改动点，且不改变后端接口或地图资产链路
- Design artifact:
  - `docs/superpowers/specs/2026-03-18-map-hover-tooltip-boundary-hotfix-design.md`

## Step 2: writing-plans

- Prompt evidence: `use superpower:writing-plans skill`
- Plan artifact:
  - `docs/superpowers/plans/2026-03-18-map-hover-tooltip-boundary-hotfix.md`
- Plan conclusion:
  - 先在 `MapPage.test.ts` 锁定失败用例，再只修改 `MapPage.vue` 的交互开关、省界样式和返回数量文案

## Step 3: executing-plans

- Prompt evidence: `use superpower:executing-plans skill`
- TDD evidence:
  - RED command: `npm test -- --run src/pages/MapPage.test.ts`
  - RED result: FAIL（3 个新增断言失败：`geo` 交互、省界样式、返回城市数文案）
  - GREEN command: `npm test -- --run src/pages/MapPage.test.ts`
  - GREEN result: PASS（13 tests）
- Code changes:
  - `web/admin/src/pages/MapPage.vue`
  - `web/admin/src/pages/MapPage.test.ts`

## Step 4: requesting-code-review

- Prompt evidence: `use superpower:requesting-code-review skill`
- Review artifact:
  - `docs/process-checklists/2026-03-18-map-hover-tooltip-boundary-hotfix-review.md`
- Round 1 findings:
  - medium: branch base contained an out-of-scope city-list change that replaced the fixed 12-row list with Top-based list expansion and counted visible items instead of actual returned API items
- Round 2 findings:
  - no actionable issues

## Step 5: receiving-code-review

- Prompt evidence: `use superpower:receiving-code-review skill`
- Round 1 action:
  - removed `visibleCityItems`
  - restored the city list to `sortedCityItems.slice(0, 12)`
  - changed returned-city text to use `cityItems.value.length`
- Round 2 action:
  - no code change

## Step 6: verification-before-completion

- Prompt evidence: `use superpower:verification-before-completion skill`
- Required commands:
  - `npm test -- --run src/pages/mapCity.test.ts src/pages/MapPage.test.ts`
  - `make build-web sync-web`
- Status:
  - `npm test -- --run src/pages/mapCity.test.ts src/pages/MapPage.test.ts` -> PASS（2 files, 19 tests）
  - `make build-web sync-web` -> PASS（仅保留既有 Vite chunk-size warning）

## Review Rounds

- Round 1: actionable issue found and fixed
- Round 2: no actionable issues
- Round 3: not needed

## Final

- Status: PASS

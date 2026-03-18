# PROCESS_CHECKLIST

## Task
地图页热修：恢复 hover/tooltip、增强省界可见性、展示当前窗口返回城市数，并让城市列表按 Top 上限显示而非固定 12 条。

## Step 1: brainstorming
- Prompt evidence: `use superpower:brainstorming skill`
- Design evidence:
  - `docs/superpowers/specs/2026-03-18-map-hover-tooltip-boundary-hotfix-design.md`
- Summary:
  - 根因收敛到前端 `MapPage.vue` 的 `geo` 交互配置、`lines` 省界 overlay 样式，以及模板里的 `sortedCityItems.slice(0, 12)` 硬编码。
  - 保持后端 `/api/map/china` 语义不变，继续保留 tooltip 中的省/市、连接数、流量三项信息。

## Step 2: writing-plans
- Prompt evidence: `use superpower:writing-plans skill`
- Plan evidence:
  - `docs/superpowers/plans/2026-03-18-map-hover-tooltip-boundary-hotfix.md`
- Summary:
  - 先补 RED 回归测试，再做 `MapPage.vue` 的最小实现改动，最后执行指定测试、构建和 review 流程。

## Step 3: executing-plans
- Prompt evidence: `use superpower:executing-plans skill`
- Execution summary:
  - 在 `web/admin/src/pages/MapPage.test.ts` 新增/扩展回归测试，覆盖 hover/tooltip、强化省界样式、返回城市数提示、列表不再截断为 12、Top 上限裁剪。
  - 在 `web/admin/src/pages/MapPage.vue` 中：
    - 保持 `geo.silent = false` 与 hover label/tooltip 链路可用。
    - 保持 `lines` overlay `silent: true`，并加深颜色、提高宽度与 z-order。
    - 新增 `visibleCityItems`，列表与计数提示统一按 Top 上限显示。
- RED evidence:
  - `npm test -- --run src/pages/MapPage.test.ts` -> FAIL（3 个失败：返回城市数文案 + 两个列表条数断言）
- GREEN evidence:
  - `npm test -- --run src/pages/MapPage.test.ts` -> PASS（15/15）

## Step 4: requesting-code-review（Round 1）
- Prompt evidence: `use superpower:requesting-code-review skill`
- Review method:
  - manual review against the user-confirmed hotfix requirements and final diff
- Review scope:
  - `web/admin/src/pages/MapPage.vue`
  - `web/admin/src/pages/MapPage.test.ts`
  - `PROCESS_CHECKLIST.md`
- Findings:
  - no actionable issues
- Review notes:
  - hover/tooltip 修复仍然保留“省/市 + 连接数 + 流量（无数据为 0）”格式。
  - 省界 overlay 仍为 `silent: true`，不会重新抢占城市 hover 事件。
  - 城市列表不再依赖 12 条硬编码，而是明确以 Top 为上限进行显示。

## Step 5: receiving-code-review（Round 1）
- Prompt evidence: `use superpower:receiving-code-review skill`
- Validation:
  - Round 1 无可执行问题，因此无需新增代码修复。
  - 复查后确认当前“返回城市数”文案使用页面当前按 Top 展示的城市数，以保证“Top 为上限”这一解释与列表显示一致，且不改后端接口语义。
- Action:
  - no code change

## Step 6: review loop closure
- Round 1: no actionable issues
- Round 2: not needed
- Round 3: not needed

## Tests
- `npm test -- --run src/pages/MapPage.test.ts`
  - RED: FAIL（3 failures expected before the list/count fix）
  - GREEN: PASS（15 tests）
- `npm test -- --run src/pages/mapCity.test.ts src/pages/MapPage.test.ts` ✅
  - 2 files passed, 21 tests passed
- `make build-web sync-web` ✅
  - build passed; only existing Vite chunk-size warning remained

## Final Verdict
- **PASS**
- 结论：用户确认的 4 个地图页热修点已在当前修复分支闭环，前端测试与构建同步均通过，满足提交/推送/PR 条件。

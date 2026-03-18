# Map Province Boundary Render Fix Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 恢复地图页按城市热度着色的可读性，移除会把整图渲染成异常蓝紫线网的省界 overlay，同时保留 `v0.1.16` 已修复的 hover 标签与 city key 行为。

**Architecture:** 采用最小修复方案，直接停用 `web/admin/src/pages/MapPage.vue` 里基于 `extractProvinceBoundarySegments()` 生成的 `lines` series，不回滚城市 GeoJSON 标准化、`city_key` join、tooltip/hover 命名逻辑。回归验证通过 `MapPage` 组件测试锁定“不再渲染推导省界 overlay”以及“无数据区域仍显示可读城市名”。

**Tech Stack:** Vue 3、ECharts 5、Vitest、TypeScript、Markdown 任务留痕。

---

## Constraints

- 用户明确禁止使用 `git worktree` / `git worktrees`；本计划直接在当前分支执行。
- 每个 Stage 完成后必须立即提交。
- 不能回滚整个 `v0.1.16`，必须保留 hover 标签与 city key 修复。
- Stage 4 禁止使用 `codex review` 子命令；review 最多 3 轮。

## File Map

- Modify: `web/admin/src/pages/MapPage.test.ts`
  - 增加地图页回归测试，证明修复后不再输出省界 `lines` overlay。
- Modify: `web/admin/src/pages/MapPage.vue`
  - 删除省界 overlay 相关 import / state / option 配置，保留城市热度图与 hover/tooltip 命名逻辑。
- Modify: `tasks/2026-03-18-map-province-boundary-render.md`
  - 记录 Stage 3 执行证据、Stage 4/5 review 结论与验证命令。
- Create: `docs/process-checklists/2026-03-18-map-province-boundary-render-review.md`
  - Stage 4/5 review 留痕，记录 review scope、findings、fix/no-fix 决策和轮次统计。

## Chunk 1: Disable the broken inferred province overlay

### Task 1: Add the regression test first

**Files:**
- Modify: `web/admin/src/pages/MapPage.test.ts`
- Test: `web/admin/src/pages/MapPage.test.ts`

- [ ] **Step 1: Write the failing test**

  新增一个 `MapPage` 组件测试，使用现有 `stubFetch()` 和 `flushPage()` 挂载页面后断言：
  - `lastChartOption.series` 中不存在 `type === 'lines'` 的条目；
  - 主 `map` series 仍然存在；
  - 现有可读城市名行为不被该测试场景破坏。

- [ ] **Step 2: Run the focused test to verify RED**

  Run: `npm test -- --run src/pages/MapPage.test.ts -t "does not render inferred province boundary overlay"`

  Expected: FAIL，原因是当前 `MapPage.vue` 仍然会输出 `series[1] = { type: 'lines' }`。

- [ ] **Step 3: Keep existing hover/city-key regression coverage intact**

  不删除现有“无数据区域 tooltip/hover 显示城市名”和“直管地区 city key 不冲突”的测试，它们是本次修复的回归护栏。

### Task 2: Remove the overlay wiring with the minimal code change

**Files:**
- Modify: `web/admin/src/pages/MapPage.vue`
- Test: `web/admin/src/pages/MapPage.test.ts`
- Test: `web/admin/src/pages/mapCity.test.ts`

- [ ] **Step 1: Write the minimal implementation**

  在 `MapPage.vue` 中移除以下仅用于省界 overlay 的内容：
  - `extractProvinceBoundarySegments` import
  - `provinceBoundarySegments` state
  - `ensureChinaMap()` 内的 `extractProvinceBoundarySegments(...)` 调用
  - `render()` 内的 `provinceBoundaryData`
  - ECharts option 中 `name: '省界'` 的 `lines` series

  保留以下内容不变：
  - `normalizeCityGeoFeatures(...)`
  - `createCityJoinKeyResolver(...)`
  - `cityNameByKey` 的 tooltip / hover label 逻辑
  - 热度图 `map` series 与 `visualMap`

- [ ] **Step 2: Run the focused test to verify GREEN**

  Run: `npm test -- --run src/pages/MapPage.test.ts -t "does not render inferred province boundary overlay"`

  Expected: PASS

- [ ] **Step 3: Run the map regression suite**

  Run: `npm test -- --run src/pages/mapCity.test.ts src/pages/MapPage.test.ts`

  Expected: PASS；既有 `mapCity` helper 测试与 `MapPage` 命名/空数据回归全部保持通过。

- [ ] **Step 4: Record execution evidence**

  在 `tasks/2026-03-18-map-province-boundary-render.md` 的 Stage 3 区段记录：
  - 实际修改范围
  - RED/GREEN 命令与结果
  - 为什么本次选择方案 1（禁用 overlay）

- [ ] **Step 5: Commit Stage 3 implementation**

  ```bash
  git add web/admin/src/pages/MapPage.vue web/admin/src/pages/MapPage.test.ts tasks/2026-03-18-map-province-boundary-render.md
  git commit -m "fix(map): disable inferred province boundary overlay"
  ```

## Chunk 2: Review and close the fix safely

### Task 3: Request code review without `codex review`

**Files:**
- Modify: `docs/process-checklists/2026-03-18-map-province-boundary-render-review.md`

- [ ] **Step 1: Review the implementation diff manually**

  以 Stage 1 验收标准为准，审查 Stage 3 diff 是否仍有：
  - hover/tooltip 标签回退风险
  - 热度图数据 join 回退风险
  - 删除 overlay 后产生的新视觉/option 结构问题

- [ ] **Step 2: Record review findings**

  在 `docs/process-checklists/2026-03-18-map-province-boundary-render-review.md` 记录：
  - review 范围
  - review 方法（明确“不使用 `codex review` 子命令”）
  - findings 按严重级别列出；若无问题，明确写 `no actionable issues`

- [ ] **Step 3: Commit Stage 4 review artifact**

  ```bash
  git add docs/process-checklists/2026-03-18-map-province-boundary-render-review.md
  git commit -m "docs: record map boundary render review findings"
  ```

### Task 4: Receive and evaluate review feedback

**Files:**
- Modify: `docs/process-checklists/2026-03-18-map-province-boundary-render-review.md`
- Modify: `tasks/2026-03-18-map-province-boundary-render.md`
- Modify if needed: `web/admin/src/pages/MapPage.vue`
- Modify if needed: `web/admin/src/pages/MapPage.test.ts`

- [ ] **Step 1: Verify each review item before changing code**

  按 `receiving-code-review` 要求逐条核对 feedback 是否真实成立；若 reviewer 结论不成立，则在 checklist 里记录技术性反驳理由，不做盲改。

- [ ] **Step 2: Apply only validated fixes**

  若存在有效问题，只做与该问题直接相关的最小修复；若 Round 1 无可执行问题，则只补录“已验证无新增动作”的结论，不制造额外改动。

- [ ] **Step 3: Re-run verification after any validated fix**

  Run: `npm test -- --run src/pages/mapCity.test.ts src/pages/MapPage.test.ts`

  Expected: PASS

- [ ] **Step 4: Update review round bookkeeping**

  在 checklist 和任务文件中记录：
  - 当前轮次
  - 问题是否成立
  - 修复/不修复决策
  - 累计 review 轮次（最多 3 轮）

- [ ] **Step 5: Commit Stage 5 result**

  ```bash
  git add docs/process-checklists/2026-03-18-map-province-boundary-render-review.md tasks/2026-03-18-map-province-boundary-render.md web/admin/src/pages/MapPage.vue web/admin/src/pages/MapPage.test.ts
  git commit -m "docs: close map boundary render review round"
  ```

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-03-18-map-province-boundary-render-fix.md`. Ready to execute.

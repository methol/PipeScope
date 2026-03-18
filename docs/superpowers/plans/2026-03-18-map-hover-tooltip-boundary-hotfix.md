# Map Hover Tooltip Boundary Hotfix Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用最小前端改动恢复地图 hover/tooltip 交互，增强省界可见性，并在页面上说明当前窗口实际返回城市数。

**Architecture:** 保持现有 `MapPage.vue` 的 `geo + map + lines` 结构，不改后端接口，不引入新配置。修复点只落在 `geo` 交互开关、`lines` 样式层级，以及页面元信息文案；回归测试集中在 `MapPage.test.ts`。

**Tech Stack:** Vue 3、TypeScript、ECharts 5、Vitest、Make。

---

## Chunk 1: Lock the target behavior with tests

### Task 1: Add failing regression coverage

**Files:**
- Modify: `web/admin/src/pages/MapPage.test.ts`
- Test: `web/admin/src/pages/MapPage.test.ts`

- [ ] **Step 1: Write the failing tests**

  Add regression coverage for:
  - `geo` no longer suppressing city hover interaction
  - stronger province-boundary `lines` styling
  - current-window returned-city count hint

- [ ] **Step 2: Run the focused test file and confirm RED**

  Run: `npm test -- --run src/pages/MapPage.test.ts`

  Expected: FAIL on the three newly added assertions before production code changes.

## Chunk 2: Implement the hotfix

### Task 2: Update `MapPage.vue` with minimal code changes

**Files:**
- Modify: `web/admin/src/pages/MapPage.vue`

- [ ] **Step 1: Restore hover/tooltip interaction**

  Remove the `geo`-level interaction suppression while keeping current tooltip and emphasis label formatter logic intact.

- [ ] **Step 2: Strengthen province boundary visibility**

  Keep the `lines` series, but raise contrast/width/z-order while leaving it `silent: true`.

- [ ] **Step 3: Add returned-city count meta text**

  Show current-window returned city count together with the selected Top limit as an upper-bound explanation.

- [ ] **Step 4: Re-run the focused test file and confirm GREEN**

  Run: `npm test -- --run src/pages/MapPage.test.ts`

  Expected: PASS

## Chunk 3: Verify, review, and ship

### Task 3: Run required verification and capture process evidence

**Files:**
- Modify: `PROCESS_CHECKLIST.md`
- Create: `docs/process-checklists/2026-03-18-map-hover-tooltip-boundary-hotfix-review.md`

- [ ] **Step 1: Run the required focused test command**

  Run: `npm test -- --run src/pages/mapCity.test.ts src/pages/MapPage.test.ts`

- [ ] **Step 2: Run the required build sync command**

  Run: `make build-web sync-web`

- [ ] **Step 3: Record Step1-Step6 evidence and review rounds in `PROCESS_CHECKLIST.md`**

- [ ] **Step 4: Request and receive one code review round**

  If no actionable issues are found, record that explicitly and stop review rounds at 1/3.

- [ ] **Step 5: Commit, push branch, and create PR to `main`**

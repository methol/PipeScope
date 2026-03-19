# Map Page UI Compact Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 压缩地图页城市统计表达，把城市/流量信息移到地图右侧，用一个指标下拉统一地图着色与列表排序，并把默认窗口改成 `1d`。

**Architecture:** 在现有 `MapPage.vue` 内做最小 UI 重排，不改接口请求方式和地图数据拼接逻辑。右侧 sidebar 复用现有 `cityItems`/`sortedCityItems` 数据，排序改为始终跟随唯一的 `metric` 选择器，样式通过页面局部 class 完成；流程留痕全部落到 `docs/process-checklists/` 与 `docs/pr/draft.md`，并以真实 git/测试状态为准。

**Tech Stack:** Vue 3, TypeScript, Vitest, ECharts

---

## Execution Notes

- This plan assumes the session starts from existing local WIP on `feature/map-page-ui-compact-20260319`, not from a clean tree.
- Do not use `git worktree`.
- After each workflow stage, attempt the requested stage commit with the designated message.
- If git metadata writes are blocked, record the exact command and exact error text in the process docs instead of fabricating a commit.

## File Map

- Modify: `web/admin/src/pages/MapPage.vue`
- Modify: `web/admin/src/pages/MapPage.test.ts`
- Modify: `docs/superpowers/specs/2026-03-19-map-page-ui-compact-design.md`
- Modify: `docs/superpowers/plans/2026-03-19-map-page-ui-compact.md`
- Create or modify later: `docs/process-checklists/2026-03-19-map-page-ui-review.md`
- Create or modify later: `docs/process-checklists/pr-draft-2026-03-19-map-page-ui.md`
- Modify later: `docs/pr/draft.md`

## Chunk 1: Validate existing WIP and lock the requested UI contract

### Task 1: Verify the focused map-page requirements against the current local changes

**Files:**
- Modify if needed: `web/admin/src/pages/MapPage.test.ts`
- Modify if needed: `web/admin/src/pages/MapPage.vue`
- Test: `web/admin/src/pages/MapPage.test.ts`

- [ ] **Step 1: Review the current WIP diff before changing behavior**

  Inspect:
  - `git diff -- web/admin/src/pages/MapPage.vue web/admin/src/pages/MapPage.test.ts`

- [ ] **Step 2: Run the focused verification command**

  Run: `npm test -- --run src/pages/MapPage.test.ts -t "defaults to 1d window requests|renders compact city stats in the right sidebar|uses one metric selector for map coloring and sidebar order|shows current returned city count in compact metadata as an upper-bound hint"`

  Expected: PASS if the carried-over WIP already satisfies the four requested behaviors; otherwise, capture the failing assertions before editing.

- [ ] **Step 3: If the focused command fails, continue with TDD**

  Order:
  - adjust or add the failing test first in `web/admin/src/pages/MapPage.test.ts`
  - re-run the focused command until the failure is specific
  - make the minimal production fix in `web/admin/src/pages/MapPage.vue`
  - re-run the focused command to GREEN

- [ ] **Step 4: Attempt the feature-stage commit**

  Run:
  ```bash
  git add web/admin/src/pages/MapPage.vue web/admin/src/pages/MapPage.test.ts
  git commit -m "feat(map): compact sidebar stats and unified metric selector"
  ```

  Expected: commit succeeds; if blocked, record the exact git error in the process checklist.

## Chunk 2: Full verification and process trace alignment

### Task 2: Collect the required verification evidence

**Files:**
- Test: `web/admin/src/pages/mapCity.test.ts`
- Test: `web/admin/src/pages/MapPage.test.ts`

- [ ] **Step 1: Run the full page regression suite**

  Run: `npm test -- --run src/pages/mapCity.test.ts src/pages/MapPage.test.ts`

  Expected: PASS

- [ ] **Step 2: Run the production build**

  Run: `npm run build`

  Expected: PASS, with at most the existing Vite chunk-size warning.

### Task 3: Align review and PR trace docs with the real repository state

**Files:**
- Modify: `docs/process-checklists/2026-03-19-map-page-ui-review.md`
- Modify: `docs/process-checklists/pr-draft-2026-03-19-map-page-ui.md`
- Modify: `docs/pr/draft.md`

- [ ] **Step 1: Replace outdated git-blocker claims with the actual current state**

  Record:
  - existing branch name and base SHA
  - whether stage commits succeeded or failed
  - the exact git error if `.git/index.lock` creation is denied

- [ ] **Step 2: Record the required verification evidence**

  Include:
  - focused command and result
  - regression command and result
  - build command and result
  - review scope and review rounds

- [ ] **Step 3: Attempt the review-request stage commit**

  Run:
  ```bash
  git add docs/process-checklists/2026-03-19-map-page-ui-review.md
  git commit -m "docs(map): request review for compact map page ui"
  ```

  Expected: commit succeeds; if blocked, record the exact git error in the process checklist.

## Chunk 3: Review closure and delivery hard checks

### Task 4: Manual code review and receiving review feedback

**Files:**
- Modify: `docs/process-checklists/2026-03-19-map-page-ui-review.md`
- Modify: `docs/process-checklists/pr-draft-2026-03-19-map-page-ui.md`

- [ ] **Step 1: Review the diff without `codex review`**

  Run: `git diff -- web/admin/src/pages/MapPage.vue web/admin/src/pages/MapPage.test.ts docs/superpowers/specs/2026-03-19-map-page-ui-compact-design.md docs/superpowers/plans/2026-03-19-map-page-ui-compact.md docs/process-checklists/2026-03-19-map-page-ui-review.md docs/process-checklists/pr-draft-2026-03-19-map-page-ui.md docs/pr/draft.md`

- [ ] **Step 2: Capture findings and close them out**

  Use at most 3 rounds. For each round, record:
  - issue
  - decision
  - fix or no-fix rationale
  - test command + result

### Task 5: Final hard checks, final docs, and push attempt

**Files:**
- Modify: `docs/process-checklists/pr-draft-2026-03-19-map-page-ui.md`
- Modify: `docs/pr/draft.md`

- [ ] **Step 1: Run the staged-file hard check**

  Run: `git diff --name-only --cached | rg "^PROCESS_CHECKLIST\\.md$"`

  Expected: no output

- [ ] **Step 2: Attempt the final docs commit**

  Run:
  ```bash
  git add docs/process-checklists/pr-draft-2026-03-19-map-page-ui.md docs/pr/draft.md
  git commit -m "docs(map): apply review resolution and finalize checklist"
  ```

  Expected: commit succeeds; if blocked, record the exact git error in the process checklist.

- [ ] **Step 3: Attempt to push the branch**

  Run:
  ```bash
  git push origin feature/map-page-ui-compact-20260319
  ```

  Expected: push succeeds; if blocked, record the exact command and failure reason.

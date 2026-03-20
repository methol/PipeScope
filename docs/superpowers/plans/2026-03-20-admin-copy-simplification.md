# Admin Copy Simplification Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 删除地图页和统计页顶部冗余提示文案，并把前台可见的 `统计/分析` 统一为 `统计`。

**Architecture:** 在现有三个 Vue 文件里做最小文本级改动，不引入共享文案层，也不重排页面结构。测试先锁定目标字符串，再删掉冗余文案并验证保留状态文案仍可见；流程留痕持续写入 `PROCESS_CHECKLIST.md`，最终归档到 `docs/process-checklists/` 并在对应 `docs/pr/<PR>.md` 建引用。

**Tech Stack:** Vue 3, TypeScript, Vitest, Vite

---

## Execution Notes

- User explicitly forbids `git worktree`; execute in the current repository branch only.
- User explicitly requires `superpowers:executing-plans`; do not switch to a worktree-based or subagent-driven implementation workflow for code changes.
- Review stage must not use `codex review`.
- Reviewer subagents are allowed only for spec/plan/code review, not for implementation.
- Each workflow stage ends with an immediate commit.
- Product-scope changes are limited to the three Vue files; process-trace docs are required workflow outputs and intentionally included in this plan.

## File Map

- Modify: `PROCESS_CHECKLIST.md`
- Modify: `web/admin/src/pages/MapPage.vue`
- Modify: `web/admin/src/pages/MapPage.test.ts`
- Modify: `web/admin/src/pages/AnalyticsPage.vue`
- Modify: `web/admin/src/pages/AnalyticsPage.test.ts`
- Modify: `web/admin/src/App.vue`
- Modify: `web/admin/src/pages/App.test.ts`
- Create later: `docs/process-checklists/2026-03-20-admin-copy-simplification-review.md`
- Create later: `docs/process-checklists/pr-<PR>-2026-03-20-admin-copy-simplification.md`
- Modify later: `docs/pr/<PR>.md`

## Chunk 1: Lock the copy contract with failing tests

### Task 1: Add focused RED tests for the removed and renamed copy

**Files:**
- Modify: `web/admin/src/pages/MapPage.test.ts`
- Modify: `web/admin/src/pages/AnalyticsPage.test.ts`
- Modify: `web/admin/src/pages/App.test.ts`
- Test: `web/admin/src/pages/MapPage.test.ts`
- Test: `web/admin/src/pages/AnalyticsPage.test.ts`
- Test: `web/admin/src/pages/App.test.ts`

- [ ] **Step 1: Add or update focused expectations before touching production code**

  Preflight:
  - confirm the current branch is `feat/map-copy-cleanup-20260320`
  - confirm the branch was created after syncing `main`
  - if either check fails, stop before editing code

  Verify:
  ```bash
  git fetch origin main
  git branch --show-current
  git merge-base --is-ancestor origin/main HEAD
  ```

  Expected:
  - current branch is `feat/map-copy-cleanup-20260320`
  - merge-base check exits `0`

  Cover:
  - `App.test.ts`: analytics tab label renders `统计` and the analytics page still switches correctly
  - `AnalyticsPage.test.ts`: heading renders `统计`, `分析型页面：不自动刷新（手动检索）` is absent, and loading/error hints still render when applicable
  - `MapPage.test.ts`: removed meta prompts are absent, and loading/error/empty-state text still render when applicable

- [ ] **Step 2: Run the focused RED suite**

  Run:
  ```bash
  npm --prefix web/admin test -- --run src/pages/App.test.ts src/pages/AnalyticsPage.test.ts src/pages/MapPage.test.ts -t "renders statistics tab label as 统计|renders heading as 统计 and omits the redundant manual-refresh hint while preserving loading and error states|omits redundant map meta hints while preserving loading error and empty-state messages"
  ```

  Expected:
  - FAIL because the current UI still shows `统计/分析` and the redundant meta text.

## Chunk 2: Minimal implementation and GREEN verification

### Task 2: Remove redundant copy and rename visible labels

**Files:**
- Modify: `web/admin/src/App.vue`
- Modify: `web/admin/src/pages/AnalyticsPage.vue`
- Modify: `web/admin/src/pages/MapPage.vue`

- [ ] **Step 1: Apply the minimal production changes**

  Change set:
  - `App.vue`: rename the analytics tab button to `统计`
  - `AnalyticsPage.vue`: rename the page heading to `统计` and remove the redundant top meta hint
  - `MapPage.vue`: remove the redundant top meta hint and sidebar returned-count helper text, plus any now-unused computed copy helpers

- [ ] **Step 2: Run the focused GREEN suite**

  Run:
  ```bash
  npm --prefix web/admin test -- --run src/pages/App.test.ts src/pages/AnalyticsPage.test.ts src/pages/MapPage.test.ts -t "renders statistics tab label as 统计|renders heading as 统计 and omits the redundant manual-refresh hint while preserving loading and error states|omits redundant map meta hints while preserving loading error and empty-state messages"
  ```

  Expected:
  - PASS

- [ ] **Step 3: Run the broader page regression suite**

  Run:
  ```bash
  npm --prefix web/admin test -- --run src/pages/App.test.ts src/pages/AnalyticsPage.test.ts src/pages/MapPage.test.ts
  ```

  Expected:
  - PASS

- [ ] **Step 4: Run the production build**

  Run:
  ```bash
  npm --prefix web/admin run build
  ```

  Expected:
  - PASS, with at most the existing Vite chunk-size warning.

- [ ] **Step 5: Commit the implementation stage**

  Run:
  ```bash
  git add PROCESS_CHECKLIST.md web/admin/src/App.vue web/admin/src/pages/App.test.ts web/admin/src/pages/AnalyticsPage.vue web/admin/src/pages/AnalyticsPage.test.ts web/admin/src/pages/MapPage.vue web/admin/src/pages/MapPage.test.ts
  git commit -m "fix(ui): simplify admin copy and rename statistics tab"
  ```

  Expected:
  - commit succeeds

## Chunk 3: Review, receipt, and delivery trace

### Task 3: Request code review without `codex review`

**Files:**
- Modify: `PROCESS_CHECKLIST.md`
- Create: `docs/process-checklists/2026-03-20-admin-copy-simplification-review.md`

- [ ] **Step 1: Capture the review request context**

  Record:
  - base SHA for the implementation review range
  - head SHA after the implementation commit
  - summary of what changed
  - test/build evidence collected so far

- [ ] **Step 2: Dispatch a reviewer subagent and record findings**

  Expected:
  - one review file with categorized findings or explicit “no findings”

- [ ] **Step 3: Commit the review-request stage**

  Run:
  ```bash
  git add PROCESS_CHECKLIST.md docs/process-checklists/2026-03-20-admin-copy-simplification-review.md
  git commit -m "docs(ui): request review for admin copy simplification"
  ```

  Expected:
  - commit succeeds

### Task 4: Receive review feedback and resolve up to three rounds

**Files:**
- Modify: `PROCESS_CHECKLIST.md`
- Modify if needed: `web/admin/src/App.vue`
- Modify if needed: `web/admin/src/pages/App.test.ts`
- Modify if needed: `web/admin/src/pages/AnalyticsPage.vue`
- Modify if needed: `web/admin/src/pages/AnalyticsPage.test.ts`
- Modify if needed: `web/admin/src/pages/MapPage.vue`
- Modify if needed: `web/admin/src/pages/MapPage.test.ts`
- Modify: `docs/process-checklists/2026-03-20-admin-copy-simplification-review.md`

- [ ] **Step 1: Evaluate each review item technically before changing code**

  Requirement:
  - record whether each item is accepted or rejected, and why

- [ ] **Step 2: For each accepted item, fix one item at a time and re-test**

  Run as needed:
  ```bash
  npm --prefix web/admin test -- --run src/pages/App.test.ts src/pages/AnalyticsPage.test.ts src/pages/MapPage.test.ts
  npm --prefix web/admin run build
  ```

  Expected:
  - PASS after each accepted fix batch

- [ ] **Step 3: Commit the review-receipt stage**

  Run:
  ```bash
  git add PROCESS_CHECKLIST.md docs/process-checklists/2026-03-20-admin-copy-simplification-review.md web/admin/src/App.vue web/admin/src/pages/App.test.ts web/admin/src/pages/AnalyticsPage.vue web/admin/src/pages/AnalyticsPage.test.ts web/admin/src/pages/MapPage.vue web/admin/src/pages/MapPage.test.ts
  git commit -m "fix(ui): address review feedback for admin copy simplification"
  ```

  Expected:
  - commit succeeds; if there are no code fixes, commit the updated docs evidence only

### Task 5: Final verification, archive, push, and PR

**Files:**
- Modify: `PROCESS_CHECKLIST.md`
- Create later: `docs/process-checklists/pr-<PR>-2026-03-20-admin-copy-simplification.md`
- Modify later: `docs/pr/<PR>.md`

- [ ] **Step 1: Run fresh final verification**

  Run:
  ```bash
  npm --prefix web/admin test -- --run src/pages/App.test.ts src/pages/AnalyticsPage.test.ts src/pages/MapPage.test.ts
  npm --prefix web/admin run build
  ```

  Expected:
  - PASS

- [ ] **Step 2: Push the branch and create a PR**

  Run:
  ```bash
  git push -u origin feat/map-copy-cleanup-20260320
  gh pr create --base main --head feat/map-copy-cleanup-20260320 --title "fix: simplify admin page copy and rename statistics tab" --body-file docs/pr/draft.md
  ```

  Expected:
  - push succeeds
  - PR creation succeeds and returns a PR number

- [ ] **Step 3: Archive the checklist and add the PR doc reference**

  Actions:
  - move `PROCESS_CHECKLIST.md` content to `docs/process-checklists/pr-<PR>-2026-03-20-admin-copy-simplification.md`
  - add a process-record reference in `docs/pr/<PR>.md`

- [ ] **Step 4: Run the staged-file hard check before the final docs commit**

  Run:
  ```bash
  git diff --name-only --cached | rg "^PROCESS_CHECKLIST\\.md$"
  ```

  Expected:
  - no output

- [ ] **Step 5: Commit the final traceability docs**

  Run:
  ```bash
  git add docs/process-checklists/pr-<PR>-2026-03-20-admin-copy-simplification.md docs/pr/<PR>.md
  git commit -m "docs(ui): archive process checklist for admin copy simplification"
  ```

  Expected:
  - commit succeeds

# Map City List Scroll Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把地图页右侧城市列表限制在固定区域内滚动，避免长列表继续拉高整页，同时保持现有右侧布局、单一指标联动和默认 `1d` 行为不变。

**Architecture:** 在 `MapPage.vue` 内做最小模板和样式调整：为右侧城市列表增加一个页面局部滚动容器，使用 `max-height` 与 `overflow-y: auto` 控制长度。数据来源、排序来源和地图渲染逻辑不变；测试在 `MapPage.test.ts` 中先锁定滚动容器结构，再验证现有行为未回退。

**Tech Stack:** Vue 3, TypeScript, Vitest, Vite

---

## Execution Notes

- User explicitly requires working in the main repository without `git worktree`; follow that instead of the default worktree guidance.
- This harness currently blocks git metadata writes under `.git`, so every stage still needs a real commit attempt, but any failure must be recorded with the exact command and exact error text.
- Keep the production diff limited to `web/admin/src/pages/MapPage.vue` and `web/admin/src/pages/MapPage.test.ts` unless a small doc/process update is required.

## File Map

- Modify: `web/admin/src/pages/MapPage.vue`
- Modify: `web/admin/src/pages/MapPage.test.ts`
- Modify later: `PROCESS_CHECKLIST.md`
- Create later: `docs/process-checklists/2026-03-19-map-city-list-scroll-review.md`
- Create later: `docs/process-checklists/pr-draft-2026-03-19-map-city-list-scroll.md`
- Modify later: `docs/pr/draft.md`

## Chunk 1: Lock the scroll-container behavior with TDD

### Task 1: Add a failing test for the bounded city-list container

**Files:**
- Modify: `web/admin/src/pages/MapPage.test.ts`
- Test: `web/admin/src/pages/MapPage.test.ts`

- [ ] **Step 1: Write the failing test**

  Add one focused assertion that the sidebar city list is wrapped by a dedicated scroll container, for example `.city-list-scroll`.

- [ ] **Step 2: Run the focused RED command**

  Run: `npm test -- --run src/pages/MapPage.test.ts -t "renders city list inside a bounded scroll container"`

  Expected: FAIL because the wrapper does not exist yet.

### Task 2: Implement the minimal template/style fix and verify GREEN

**Files:**
- Modify: `web/admin/src/pages/MapPage.vue`
- Test: `web/admin/src/pages/MapPage.test.ts`

- [ ] **Step 1: Add the minimal production change**

  Update the sidebar markup to wrap `.city-list` in a dedicated container and add page-local CSS with:
  - reasonable `max-height`
  - `overflow-y: auto`
  - no changes to metric sorting, filtering, or default window logic

- [ ] **Step 2: Run the focused GREEN command**

  Run: `npm test -- --run src/pages/MapPage.test.ts -t "renders city list inside a bounded scroll container"`

  Expected: PASS.

- [ ] **Step 3: Run the required page test file**

  Run: `npm test -- --run src/pages/MapPage.test.ts`

  Expected: PASS with the existing regression coverage still green.

- [ ] **Step 4: Attempt the execution-stage commit**

  Run:
  ```bash
  git add web/admin/src/pages/MapPage.vue web/admin/src/pages/MapPage.test.ts
  git commit -m "fix(map): constrain city list to a scroll area"
  ```

  Expected: commit succeeds; if blocked, record the exact error in `PROCESS_CHECKLIST.md`.

## Chunk 2: Verification, review trace, and archive docs

### Task 3: Collect the required verification evidence

**Files:**
- Test: `web/admin/src/pages/MapPage.test.ts`

- [ ] **Step 1: Re-run the required focused test command**

  Run: `npm test -- --run src/pages/MapPage.test.ts`

  Expected: PASS.

- [ ] **Step 2: Run the production build**

  Run: `npm run build`

  Expected: PASS, with at most existing non-blocking build warnings.

### Task 4: Request and receive review in-session

**Files:**
- Create: `docs/process-checklists/2026-03-19-map-city-list-scroll-review.md`
- Modify: `PROCESS_CHECKLIST.md`

- [ ] **Step 1: Review the implementation manually without `codex review`**

  Inspect:
  - `git diff -- web/admin/src/pages/MapPage.vue web/admin/src/pages/MapPage.test.ts`

- [ ] **Step 2: Record findings, decisions, and any fixes**

  For each review round, record:
  - finding severity
  - technical decision
  - fix/no-fix rationale
  - test/build evidence after any change

- [ ] **Step 3: Attempt the review-stage commit**

  Run:
  ```bash
  git add docs/process-checklists/2026-03-19-map-city-list-scroll-review.md
  git commit -m "docs(map): request review for city list scroll"
  ```

  Expected: commit succeeds; if blocked, record the exact error.

### Task 5: Archive the checklist and run the final hard check

**Files:**
- Create: `docs/process-checklists/pr-draft-2026-03-19-map-city-list-scroll.md`
- Modify: `docs/pr/draft.md`
- Modify: `PROCESS_CHECKLIST.md`

- [ ] **Step 1: Archive the working checklist**

  Copy the final contents of `PROCESS_CHECKLIST.md` into `docs/process-checklists/pr-draft-2026-03-19-map-city-list-scroll.md`.

- [ ] **Step 2: Add the draft PR reference**

  Update `docs/pr/draft.md` with a new link to the archived checklist.

- [ ] **Step 3: Run the staged-file hard check**

  Run: `git diff --name-only --cached | rg '^PROCESS_CHECKLIST\\.md$'`

  Expected: no output.

- [ ] **Step 4: Attempt the final commit**

  Run:
  ```bash
  git add docs/process-checklists/pr-draft-2026-03-19-map-city-list-scroll.md docs/pr/draft.md
  git commit -m "docs(map): finalize city list scroll checklist"
  ```

  Expected: commit succeeds; if blocked, record the exact error.

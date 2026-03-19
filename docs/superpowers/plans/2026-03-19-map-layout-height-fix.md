# Map Page Layout Height Fix Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 增大地图页桌面端地图视口，并让右侧栏在共享的限定高度内滚动，消除地图下方的大块空白，同时保持右侧栏布局、唯一指标联动和默认 `1d` 行为不变。

**Architecture:** 只在 [`web/admin/src/pages/MapPage.vue`](/Users/methol/code/github.com/methol/PipeScope/web/admin/src/pages/MapPage.vue) 内做最小模板和样式调整：增加 map shell / sidebar body 两个局部容器，在桌面端给 `map-layout` 一个共享的响应式高度，并让 `.chart` 与 `.city-list-scroll` 都服从这套高度契约。测试先在 [`web/admin/src/pages/MapPage.test.ts`](/Users/methol/code/github.com/methol/PipeScope/web/admin/src/pages/MapPage.test.ts) 锁定新的结构，再实现样式，最后跑页面测试和构建验证。

**Tech Stack:** Vue 3, TypeScript, Vitest, Vite

---

## Execution Notes

- User explicitly forbids `git worktree`; do all work in the current repository checkout.
- This harness blocks `.git` writes, so every branch / commit / push / PR action still needs a real attempt and its exact failure output recorded in `PROCESS_CHECKLIST.md`.
- Keep production changes focused on [`web/admin/src/pages/MapPage.vue`](/Users/methol/code/github.com/methol/PipeScope/web/admin/src/pages/MapPage.vue) and [`web/admin/src/pages/MapPage.test.ts`](/Users/methol/code/github.com/methol/PipeScope/web/admin/src/pages/MapPage.test.ts).
- Review must be done manually in-session; do not use the `codex review` subcommand.

## File Map

- Modify: `web/admin/src/pages/MapPage.test.ts`
- Modify: `web/admin/src/pages/MapPage.vue`
- Modify later: `PROCESS_CHECKLIST.md`
- Create later: `docs/process-checklists/2026-03-19-map-layout-height-fix-review.md`
- Create later: `docs/process-checklists/pr-draft-2026-03-19-map-layout-height-fix.md`
- Modify later: `docs/pr/draft.md`

## Chunk 1: Lock the bounded-height layout structure with TDD

### Task 1: Add a failing test for the shared layout shells

**Files:**
- Modify: `web/admin/src/pages/MapPage.test.ts`
- Test: `web/admin/src/pages/MapPage.test.ts`

- [ ] **Step 1: Write the failing test**

  Add one focused test asserting that:
  - the chart is wrapped by a dedicated `.map-main-shell`
  - the sidebar content is wrapped by `.map-sidebar-body`
  - `.city-list-scroll` lives inside that bounded sidebar body

- [ ] **Step 2: Run the focused RED command**

  Run: `npm test -- --run src/pages/MapPage.test.ts -t "renders map and sidebar inside shared bounded layout shells"`

  Expected: FAIL because the new wrappers do not exist yet.

### Task 2: Implement the minimal layout fix and verify GREEN

**Files:**
- Modify: `web/admin/src/pages/MapPage.vue`
- Test: `web/admin/src/pages/MapPage.test.ts`

- [ ] **Step 1: Add the minimal production change**

  In [`web/admin/src/pages/MapPage.vue`](/Users/methol/code/github.com/methol/PipeScope/web/admin/src/pages/MapPage.vue):
  - add the minimal wrappers required by Task 1
  - introduce a page-local shared desktop height using bounded responsive CSS
  - override the page-local `.chart` height to fill the larger map shell
  - make the sidebar body flex and keep `.city-list-scroll` as the internal overflow region
  - preserve metric selection, sorting, filters, and default `1d`

- [ ] **Step 2: Run the focused GREEN command**

  Run: `npm test -- --run src/pages/MapPage.test.ts -t "renders map and sidebar inside shared bounded layout shells"`

  Expected: PASS.

- [ ] **Step 3: Run the required page test file**

  Run: `npm test -- --run src/pages/MapPage.test.ts`

  Expected: PASS with the existing map-page regressions still green.

- [ ] **Step 4: Attempt the execution-stage commit**

  Run:
  ```bash
  git add web/admin/src/pages/MapPage.vue web/admin/src/pages/MapPage.test.ts
  git commit -m "fix(map): balance map viewport and sidebar height"
  ```

  Expected: commit succeeds; if blocked, record the exact error in `PROCESS_CHECKLIST.md`.

## Chunk 2: Verification, review trace, and archive docs

### Task 3: Collect verification evidence

**Files:**
- Test: `web/admin/src/pages/MapPage.test.ts`

- [ ] **Step 1: Run the required test command**

  Run: `npm test -- --run src/pages/MapPage.test.ts`

  Expected: PASS.

- [ ] **Step 2: Run the required build command**

  Run: `npm run build`

  Expected: PASS, with at most the existing non-blocking Vite chunk-size warning.

### Task 4: Request and receive review in-session

**Files:**
- Create: `docs/process-checklists/2026-03-19-map-layout-height-fix-review.md`
- Modify: `PROCESS_CHECKLIST.md`

- [ ] **Step 1: Review the implementation manually without `codex review`**

  Inspect:
  - `git diff -- web/admin/src/pages/MapPage.vue web/admin/src/pages/MapPage.test.ts`

- [ ] **Step 2: Record findings and technical decisions**

  For each review round, capture:
  - severity
  - decision
  - fix/no-fix rationale
  - fresh verification evidence after any fix

- [ ] **Step 3: Attempt the review-stage commit**

  Run:
  ```bash
  git add docs/process-checklists/2026-03-19-map-layout-height-fix-review.md
  git commit -m "docs(map): request review for layout height hotfix"
  ```

  Expected: commit succeeds; if blocked, record the exact error.

### Task 5: Archive the checklist and prepare PR docs

**Files:**
- Create: `docs/process-checklists/pr-draft-2026-03-19-map-layout-height-fix.md`
- Modify: `docs/pr/draft.md`
- Modify: `PROCESS_CHECKLIST.md`

- [ ] **Step 1: Archive the working checklist**

  Copy the final contents of `PROCESS_CHECKLIST.md` into `docs/process-checklists/pr-draft-2026-03-19-map-layout-height-fix.md`.

- [ ] **Step 2: Add the draft PR reference**

  Update [`docs/pr/draft.md`](/Users/methol/code/github.com/methol/PipeScope/docs/pr/draft.md) with the new archived checklist link.

- [ ] **Step 3: Run the staged-file hard check**

  Run: `git diff --name-only --cached | rg '^PROCESS_CHECKLIST\\.md$'`

  Expected: no output.

- [ ] **Step 4: Attempt the final docs commit**

  Run:
  ```bash
  git add docs/process-checklists/pr-draft-2026-03-19-map-layout-height-fix.md docs/pr/draft.md
  git commit -m "docs(map): finalize layout height fix checklist"
  ```

  Expected: commit succeeds; if blocked, record the exact error.

- [ ] **Step 5: Attempt push and PR creation**

  Run:
  ```bash
  git push -u origin hotfix/map-layout-height-fix-20260319
  gh pr create --base main --head hotfix/map-layout-height-fix-20260319 --title "fix: balance map page height on desktop" --body-file docs/pr/draft.md
  ```

  Expected: push and PR succeed; if blocked, record the exact error text and stop claiming success.

# Map Page Iconfont Follow-up Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current `连/流` text chips in the map sidebar with accessible iconfont-style stat badges without regressing the right sidebar layout, unified metric selector behavior, or default `1d` requests.

**Architecture:** Keep the change local to `MapPage.vue` and its tests. Use page-local inline SVG badges styled to read like lightweight iconfont chips, preserve the existing tooltip contract via `title`, and keep the current computed sorting/render flow unchanged except for the badge markup and CSS.

**Tech Stack:** Vue 3, TypeScript, Vitest, ECharts

---

## Execution Notes

- User constraint overrides the usual worktree requirement: do not use `git worktree`.
- Execute on the current branch only: `feature/map-page-ui-compact-20260319`.
- Attempt the requested stage commit after each stage; if blocked, record the exact command and exact error.
- Review/fix loop may use at most 3 rounds.

## File Map

- Create: `docs/superpowers/specs/2026-03-19-map-page-iconfont-design.md`
- Create: `docs/superpowers/plans/2026-03-19-map-page-iconfont-plan.md`
- Modify: `web/admin/src/pages/MapPage.vue`
- Modify: `web/admin/src/pages/MapPage.test.ts`
- Create: `docs/process-checklists/pr-33-2026-03-19-map-iconfont-followup.md`
- Create or modify: `docs/pr/33.md`

## Chunk 1: Lock docs for the iconfont follow-up

### Task 1: Save the approved design artifact

**Files:**
- Create: `docs/superpowers/specs/2026-03-19-map-page-iconfont-design.md`

- [ ] **Step 1: Write the spec**

  Include:
  - follow-up scope relative to the existing compact sidebar work
  - 2-3 options with recommendation
  - chosen iconfont-style badge design
  - explicit non-regression items

- [ ] **Step 2: Attempt the brainstorming-stage commit**

  Run:
  ```bash
  git add docs/superpowers/specs/2026-03-19-map-page-iconfont-design.md
  git commit -m "docs(map): brainstorm iconfont style for city stats"
  ```

### Task 2: Save the execution plan

**Files:**
- Create: `docs/superpowers/plans/2026-03-19-map-page-iconfont-plan.md`

- [ ] **Step 1: Write the plan**

  Cover:
  - TDD-first test update
  - minimal production edits in `MapPage.vue`
  - checklist/PR archive updates
  - required verification commands

- [ ] **Step 2: Attempt the planning-stage commit**

  Run:
  ```bash
  git add docs/superpowers/plans/2026-03-19-map-page-iconfont-plan.md
  git commit -m "docs(map): plan iconfont follow-up for map stats"
  ```

## Chunk 2: TDD the icon-style stat badges

### Task 3: Update the focused test first

**Files:**
- Modify: `web/admin/src/pages/MapPage.test.ts`

- [ ] **Step 1: Change the compact-sidebar assertion to expect icon-style badges**

  Assert:
  - the row still renders in the right sidebar
  - connection/traffic badges still have tooltip text
  - visible `连/流` prefixes are gone
  - icon containers exist for each badge
  - accessibility copy exists (`sr-only` or equivalent)

- [ ] **Step 2: Run the focused RED command**

  Run:
  ```bash
  npm test -- --run src/pages/MapPage.test.ts -t "renders compact city stats in the right sidebar|uses one metric selector for map coloring and sidebar order|defaults to 1d window requests"
  ```

  Expected: fail on the sidebar icon-style expectation before production code changes.

### Task 4: Implement the minimal UI change

**Files:**
- Modify: `web/admin/src/pages/MapPage.vue`

- [ ] **Step 1: Replace short-text prefixes with icon-style stat badges**

  Implement:
  - inline SVG icon for connections
  - inline SVG icon for traffic
  - `aria-hidden="true"` on decorative icons
  - hidden label text for accessibility
  - keep `title` on each badge

- [ ] **Step 2: Adjust page-local CSS**

  Keep:
  - current right-sidebar layout
  - current responsive stacking

  Add:
  - icon carrier styling
  - stronger distinction between connection and traffic badges
  - `.sr-only` helper scoped to the page

- [ ] **Step 3: Run the focused GREEN command**

  Run:
  ```bash
  npm test -- --run src/pages/MapPage.test.ts -t "renders compact city stats in the right sidebar|uses one metric selector for map coloring and sidebar order|defaults to 1d window requests"
  ```

- [ ] **Step 4: Attempt the feature-stage commit**

  Run:
  ```bash
  git add web/admin/src/pages/MapPage.vue web/admin/src/pages/MapPage.test.ts
  git commit -m "feat(map): use icon-style chips for city stats"
  ```

## Chunk 3: Review trail, verification, and delivery attempts

### Task 5: Record the review request stage

**Files:**
- Create: `docs/process-checklists/pr-33-2026-03-19-map-iconfont-followup.md`
- Create or modify: `docs/pr/33.md`

- [ ] **Step 1: Save the checklist archive**

  Include:
  - all required superpower stage labels
  - commit attempts and outcomes
  - review rounds
  - verification evidence placeholders/results

- [ ] **Step 2: Add the checklist reference to `docs/pr/33.md`**

- [ ] **Step 3: Attempt the review-request stage commit**

  Run:
  ```bash
  git add docs/process-checklists/pr-33-2026-03-19-map-iconfont-followup.md docs/pr/33.md
  git commit -m "docs(map): request review for iconfont follow-up"
  ```

### Task 6: Review, verify, and close

**Files:**
- Modify: `docs/process-checklists/pr-33-2026-03-19-map-iconfont-followup.md`
- Modify if review finds issues: `web/admin/src/pages/MapPage.vue`
- Modify if review finds issues: `web/admin/src/pages/MapPage.test.ts`
- Modify if needed: `docs/pr/33.md`

- [ ] **Step 1: Review the diff manually without `codex review`**

  Inspect:
  ```bash
  git diff -- docs/superpowers/specs/2026-03-19-map-page-iconfont-design.md docs/superpowers/plans/2026-03-19-map-page-iconfont-plan.md web/admin/src/pages/MapPage.vue web/admin/src/pages/MapPage.test.ts docs/process-checklists/pr-33-2026-03-19-map-iconfont-followup.md docs/pr/33.md
  ```

- [ ] **Step 2: If feedback is needed, apply it using receiving-code-review discipline**

  Limit: max 3 rounds.

- [ ] **Step 3: Run the required regression suite**

  Run:
  ```bash
  npm test -- --run src/pages/mapCity.test.ts src/pages/MapPage.test.ts
  ```

- [ ] **Step 4: Run the production build**

  Run:
  ```bash
  npm run build
  ```

- [ ] **Step 5: Run the staged-file hard check**

  Run:
  ```bash
  git diff --name-only --cached | rg "^PROCESS_CHECKLIST\\.md$"
  ```

  Expected: no output.

- [ ] **Step 6: Attempt the final archive commit**

  Run:
  ```bash
  git add docs/process-checklists/pr-33-2026-03-19-map-iconfont-followup.md docs/pr/33.md web/admin/src/pages/MapPage.vue web/admin/src/pages/MapPage.test.ts
  git commit -m "docs(map): apply review resolution and archive checklist"
  ```

- [ ] **Step 7: Attempt push and PR update**

  Run:
  ```bash
  git push origin feature/map-page-ui-compact-20260319
  gh pr edit 33 --body-file docs/pr/33.md
  ```
